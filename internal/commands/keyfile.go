package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/napalu/goopt/v2"
	"github.com/napalu/sympa/internal/age"
	"github.com/napalu/sympa/internal/agent"
	"github.com/napalu/sympa/internal/options"
	"github.com/napalu/sympa/internal/store"
)

const (
	rekeyJournalFile = ".rekey-journal"
	rekeyBackupDir   = ".rekey-backup"
)

type rekeyJournal struct {
	Version            int      `json:"version"`
	OldKeyfileFP       string   `json:"old_keyfile_fp,omitempty"`
	NewKeyfileFP       string   `json:"new_keyfile_fp,omitempty"`
	ChangingPassphrase bool     `json:"changing_passphrase"`
	Removing           bool     `json:"removing"`
	Secrets            []string `json:"secrets"`
}

// passphraseEntry holds derived passphrases for a single raw passphrase.
type passphraseEntry struct {
	oldDerived string // derive(raw, oldKeyfile) — for decrypting from backup
	newDerived string // derive(raw, newKeyfile) — for re-encrypting without --passphrase
}

// newPassphraseEntry derives old+new effective passphrases from a raw passphrase.
func newPassphraseEntry(raw string, oldKeyfile, newKeyfile []byte, removing bool) (*passphraseEntry, error) {
	e := &passphraseEntry{}
	var err error

	e.oldDerived = raw
	if len(oldKeyfile) > 0 {
		e.oldDerived, err = age.DerivePassphrase(raw, oldKeyfile)
		if err != nil {
			return nil, err
		}
	}

	switch {
	case len(newKeyfile) > 0:
		e.newDerived, err = age.DerivePassphrase(raw, newKeyfile)
	case removing:
		e.newDerived = raw
	case len(oldKeyfile) > 0:
		e.newDerived, err = age.DerivePassphrase(raw, oldKeyfile)
	default:
		e.newDerived = raw
	}
	return e, err
}

func handleKeyfileVerify(p *goopt.Parser, _ *goopt.Command) error {
	cfg, ok := goopt.GetStructCtxAs[*options.Config](p)
	if !ok {
		return errNoConfig
	}
	s := store.New()
	if err := requireInit(s); err != nil {
		return err
	}

	meta, err := s.ReadMetadata()
	if err != nil {
		return err
	}
	if meta.KeyfileFingerprint == "" {
		fmt.Println("This store was initialized without a keyfile.")
		return nil
	}

	if cfg.Keyfile == "" {
		return fmt.Errorf("store has a keyfile fingerprint but no --keyfile provided")
	}

	content, err := os.ReadFile(cfg.Keyfile)
	if err != nil {
		return fmt.Errorf("reading keyfile: %w", err)
	}
	fp := age.KeyfileFingerprint(content)

	if fp == meta.KeyfileFingerprint {
		fmt.Printf("Keyfile matches store fingerprint: %s\n", fp)
		return nil
	}
	return fmt.Errorf("keyfile mismatch:\n  store expects: %s\n  keyfile has:   %s", meta.KeyfileFingerprint, fp)
}

func handleKeyfileRekey(p *goopt.Parser, _ *goopt.Command) error {
	cfg, ok := goopt.GetStructCtxAs[*options.Config](p)
	if !ok {
		return errNoConfig
	}
	s := store.New()
	if err := requireInit(s); err != nil {
		return err
	}

	rekey := cfg.KeyfileMgmt.Rekey
	if rekey.Resume && rekey.Abort {
		return fmt.Errorf("cannot use --resume and --abort together")
	}
	if rekey.Abort {
		return handleRekeyAbort(s)
	}
	if rekey.Resume {
		return handleRekeyResume(s, cfg)
	}

	meta, err := s.ReadMetadata()
	if err != nil {
		return err
	}
	return handleRekeyStart(s, cfg, meta)
}

func handleRekeyStart(s *store.Store, cfg *options.Config, meta store.StoreMetadata) error {
	storeDir := s.Dir()

	// Check for interrupted rekey
	j, err := readRekeyJournal(storeDir)
	if err != nil {
		return err
	}
	if j != nil {
		return fmt.Errorf("interrupted rekey detected; use --resume to continue or --abort to discard")
	}
	cleanupRekey(storeDir)

	newKeyfilePath := cfg.KeyfileMgmt.Rekey.NewKeyfile
	removing := cfg.KeyfileMgmt.Rekey.Remove
	changingPassphrase := cfg.KeyfileMgmt.Rekey.Passphrase
	changingKeyfile := newKeyfilePath != "" || removing

	// Validate flags
	if !changingKeyfile && !changingPassphrase {
		return fmt.Errorf("specify a new keyfile path, --remove, or --passphrase")
	}
	if removing && newKeyfilePath != "" {
		return fmt.Errorf("cannot use --remove with a new keyfile path")
	}
	if removing && meta.KeyfileFingerprint == "" {
		return fmt.Errorf("store has no keyfile to remove")
	}
	if meta.KeyfileFingerprint != "" && cfg.Keyfile == "" {
		return fmt.Errorf("store requires --keyfile (-k) to authenticate")
	}
	if err := requireKeyfileConsistency(s, cfg.Keyfile); err != nil {
		return err
	}

	// Read old keyfile
	var oldKeyfileContent []byte
	if cfg.Keyfile != "" {
		oldKeyfileContent, err = os.ReadFile(cfg.Keyfile)
		if err != nil {
			return fmt.Errorf("reading old keyfile: %w", err)
		}
	}

	// Read new keyfile
	var newKeyfileContent []byte
	if newKeyfilePath != "" {
		newKeyfileContent, err = os.ReadFile(newKeyfilePath)
		if err != nil {
			return fmt.Errorf("reading new keyfile: %w", err)
		}
	}

	// Prompt for primary passphrase
	primaryRaw, err := age.ReadPassphrase("Current passphrase: ")
	if err != nil {
		return err
	}

	// List secrets
	secrets, err := s.AllSecrets()
	if err != nil {
		return fmt.Errorf("listing secrets: %w", err)
	}

	if len(secrets) > 0 {
		// Create backup
		if err := backupSecrets(storeDir, secrets); err != nil {
			cleanupRekey(storeDir)
			return err
		}

		// Write journal
		journal := &rekeyJournal{
			Version:            1,
			OldKeyfileFP:       meta.KeyfileFingerprint,
			ChangingPassphrase: changingPassphrase,
			Removing:           removing,
			Secrets:            secrets,
		}
		if newKeyfilePath != "" {
			journal.NewKeyfileFP = age.KeyfileFingerprint(newKeyfileContent)
		}
		if err := writeRekeyJournal(storeDir, journal); err != nil {
			cleanupRekey(storeDir)
			return err
		}

		// Discovery pass: verify credentials and discover partitioning
		mapping, err := discoverPassphrases(storeDir, secrets, primaryRaw, oldKeyfileContent, newKeyfileContent, removing)
		if err != nil {
			return err
		}

		// Determine global new passphrase (only with --passphrase)
		var globalNewDerived string
		if changingPassphrase {
			newRaw, err := age.ReadPassphraseConfirm("New passphrase: ")
			if err != nil {
				return err
			}
			entry, err := newPassphraseEntry(newRaw, nil, newKeyfileContent, removing)
			if err != nil {
				return err
			}
			globalNewDerived = entry.newDerived
		}

		// Rekey pass: streaming decrypt → re-encrypt → write
		if err := rekeySecrets(storeDir, s, secrets, mapping, globalNewDerived); err != nil {
			return err
		}
	}

	// Update metadata
	switch {
	case newKeyfilePath != "":
		meta.KeyfileFingerprint = age.KeyfileFingerprint(newKeyfileContent)
	case removing:
		meta.KeyfileFingerprint = ""
	}
	if err := s.WriteMetadata(meta); err != nil {
		return fmt.Errorf("updating metadata: %w", err)
	}

	cleanupRekey(storeDir)
	flushAgentCache()

	fmt.Fprintf(os.Stderr, "Re-encrypted %d secrets.\n", len(secrets))
	if newKeyfilePath != "" {
		fmt.Fprintf(os.Stderr, "New keyfile fingerprint: %s\n", meta.KeyfileFingerprint)
	} else if removing {
		fmt.Fprintln(os.Stderr, "Keyfile removed from store.")
	}
	return nil
}

func handleRekeyResume(s *store.Store, cfg *options.Config) error {
	storeDir := s.Dir()
	j, err := readRekeyJournal(storeDir)
	if err != nil {
		return err
	}
	if j == nil {
		return fmt.Errorf("no interrupted rekey found")
	}

	// Validate old keyfile
	var oldKeyfileContent []byte
	if j.OldKeyfileFP != "" {
		if cfg.Keyfile == "" {
			return fmt.Errorf("interrupted rekey requires old keyfile (--keyfile / -k)\n  expected fingerprint: %s", j.OldKeyfileFP)
		}
		oldKeyfileContent, err = os.ReadFile(cfg.Keyfile)
		if err != nil {
			return fmt.Errorf("reading old keyfile: %w", err)
		}
		if fp := age.KeyfileFingerprint(oldKeyfileContent); fp != j.OldKeyfileFP {
			return fmt.Errorf("old keyfile fingerprint mismatch:\n  journal expects: %s\n  provided:        %s", j.OldKeyfileFP, fp)
		}
	} else if cfg.Keyfile != "" {
		return fmt.Errorf("interrupted rekey did not use a keyfile; omit --keyfile")
	}

	// Validate new keyfile
	var newKeyfileContent []byte
	newKeyfilePath := cfg.KeyfileMgmt.Rekey.NewKeyfile
	if j.NewKeyfileFP != "" {
		if newKeyfilePath == "" {
			return fmt.Errorf("interrupted rekey requires new keyfile\n  expected fingerprint: %s", j.NewKeyfileFP)
		}
		newKeyfileContent, err = os.ReadFile(newKeyfilePath)
		if err != nil {
			return fmt.Errorf("reading new keyfile: %w", err)
		}
		if fp := age.KeyfileFingerprint(newKeyfileContent); fp != j.NewKeyfileFP {
			return fmt.Errorf("new keyfile fingerprint mismatch:\n  journal expects: %s\n  provided:        %s", j.NewKeyfileFP, fp)
		}
	} else if newKeyfilePath != "" {
		return fmt.Errorf("interrupted rekey did not add a new keyfile; omit the keyfile argument")
	}

	// Prompt for primary passphrase
	primaryRaw, err := age.ReadPassphrase("Current passphrase: ")
	if err != nil {
		return err
	}

	// Discovery pass from backup
	mapping, err := discoverPassphrases(storeDir, j.Secrets, primaryRaw, oldKeyfileContent, newKeyfileContent, j.Removing)
	if err != nil {
		return err
	}

	// Determine global new passphrase (only if changing)
	var globalNewDerived string
	if j.ChangingPassphrase {
		newRaw, err := age.ReadPassphraseConfirm("New passphrase: ")
		if err != nil {
			return err
		}
		entry, err := newPassphraseEntry(newRaw, nil, newKeyfileContent, j.Removing)
		if err != nil {
			return err
		}
		globalNewDerived = entry.newDerived
	}

	// Rekey pass (streaming)
	if err := rekeySecrets(storeDir, s, j.Secrets, mapping, globalNewDerived); err != nil {
		return err
	}

	// Update metadata
	meta, err := s.ReadMetadata()
	if err != nil {
		return err
	}
	switch {
	case j.NewKeyfileFP != "":
		meta.KeyfileFingerprint = j.NewKeyfileFP
	case j.Removing:
		meta.KeyfileFingerprint = ""
	}
	if err := s.WriteMetadata(meta); err != nil {
		return fmt.Errorf("updating metadata: %w", err)
	}

	cleanupRekey(storeDir)
	flushAgentCache()

	fmt.Fprintf(os.Stderr, "Re-encrypted %d secrets.\n", len(j.Secrets))
	if j.NewKeyfileFP != "" {
		fmt.Fprintf(os.Stderr, "New keyfile fingerprint: %s\n", j.NewKeyfileFP)
	} else if j.Removing {
		fmt.Fprintln(os.Stderr, "Keyfile removed from store.")
	}
	return nil
}

func handleRekeyAbort(s *store.Store) error {
	storeDir := s.Dir()
	j, err := readRekeyJournal(storeDir)
	if err != nil {
		return err
	}
	if j == nil {
		return fmt.Errorf("no interrupted rekey found")
	}

	if err := restoreBackup(storeDir, j.Secrets); err != nil {
		return err
	}

	meta := store.StoreMetadata{KeyfileFingerprint: j.OldKeyfileFP}
	if err := s.WriteMetadata(meta); err != nil {
		return fmt.Errorf("restoring metadata: %w", err)
	}

	cleanupRekey(storeDir)
	fmt.Fprintln(os.Stderr, "Rekey aborted. Store restored from backup.")
	return nil
}

// discoverPassphrases verifies credentials and discovers per-secret passphrase partitioning.
// It reads from backup (read-only) and prompts for additional passphrases as needed.
func discoverPassphrases(storeDir string, secrets []string,
	primaryRaw string, oldKeyfile, newKeyfile []byte, removing bool,
) (map[string]*passphraseEntry, error) {
	backupRoot := filepath.Join(storeDir, rekeyBackupDir)

	primary, err := newPassphraseEntry(primaryRaw, oldKeyfile, newKeyfile, removing)
	if err != nil {
		return nil, err
	}

	known := []*passphraseEntry{primary}
	mapping := make(map[string]*passphraseEntry, len(secrets))
	total := len(secrets)

	for i, name := range secrets {
		fmt.Fprintf(os.Stderr, "\rVerifying secrets... %d/%d", i+1, total)
		ct, err := os.ReadFile(filepath.Join(backupRoot, name+".age"))
		if err != nil {
			fmt.Fprintln(os.Stderr)
			return nil, fmt.Errorf("reading backup %q: %w", name, err)
		}

		found := false
		for _, entry := range known {
			if _, decErr := age.Decrypt(ct, entry.oldDerived); decErr == nil {
				mapping[name] = entry
				found = true
				break
			}
		}

		if !found {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "Secret %q requires a different passphrase.\n", name)
			raw, err := age.ReadPassphrase("Passphrase for this secret: ")
			if err != nil {
				return nil, err
			}
			entry, err := newPassphraseEntry(raw, oldKeyfile, newKeyfile, removing)
			if err != nil {
				return nil, err
			}
			if _, err := age.Decrypt(ct, entry.oldDerived); err != nil {
				return nil, fmt.Errorf("decrypting %q: %w (wrong passphrase?)", name, err)
			}
			known = append(known, entry)
			mapping[name] = entry
		}
	}
	fmt.Fprintln(os.Stderr)

	if len(known) > 1 {
		fmt.Fprintf(os.Stderr, "Found %d distinct passphrases across %d secrets.\n", len(known), total)
	}

	return mapping, nil
}

// rekeySecrets streams decrypt → re-encrypt → write for each secret.
// If globalNewDerived is non-empty (--passphrase), it overrides per-entry newDerived.
func rekeySecrets(storeDir string, s *store.Store, secrets []string,
	mapping map[string]*passphraseEntry, globalNewDerived string) error {
	backupRoot := filepath.Join(storeDir, rekeyBackupDir)
	total := len(secrets)

	for i, name := range secrets {
		fmt.Fprintf(os.Stderr, "\rRe-encrypting secrets... %d/%d", i+1, total)
		entry := mapping[name]

		ct, err := os.ReadFile(filepath.Join(backupRoot, name+".age"))
		if err != nil {
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("reading backup %q: %w", name, err)
		}
		pt, err := age.Decrypt(ct, entry.oldDerived)
		if err != nil {
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("decrypting %q: %w", name, err)
		}

		newDerived := entry.newDerived
		if globalNewDerived != "" {
			newDerived = globalNewDerived
		}

		newCt, err := age.Encrypt(pt, newDerived)
		if err != nil {
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("encrypting %q: %w", name, err)
		}
		if err := s.Write(name, newCt); err != nil {
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("writing %q: %w", name, err)
		}
	}
	fmt.Fprintln(os.Stderr)
	return nil
}

// readRekeyJournal reads the journal file, returning nil if it doesn't exist.
func readRekeyJournal(storeDir string) (*rekeyJournal, error) {
	data, err := os.ReadFile(filepath.Join(storeDir, rekeyJournalFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading rekey journal: %w", err)
	}
	var j rekeyJournal
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("parsing rekey journal: %w", err)
	}
	return &j, nil
}

// writeRekeyJournal writes the journal file.
func writeRekeyJournal(storeDir string, j *rekeyJournal) error {
	data, err := json.Marshal(j)
	if err != nil {
		return fmt.Errorf("marshaling rekey journal: %w", err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, rekeyJournalFile), data, 0600); err != nil {
		return fmt.Errorf("writing rekey journal: %w", err)
	}
	return nil
}

// backupSecrets copies all .age files to the backup directory.
func backupSecrets(storeDir string, secrets []string) error {
	backupRoot := filepath.Join(storeDir, rekeyBackupDir)
	total := len(secrets)
	for i, name := range secrets {
		fmt.Fprintf(os.Stderr, "\rBacking up secrets... %d/%d", i+1, total)
		src := filepath.Join(storeDir, name+".age")
		dst := filepath.Join(backupRoot, name+".age")
		if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("creating backup directory: %w", err)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("reading %q for backup: %w", name, err)
		}
		if err := os.WriteFile(dst, data, 0600); err != nil {
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("writing backup for %q: %w", name, err)
		}
	}
	fmt.Fprintln(os.Stderr)
	return nil
}

// restoreBackup copies all .age files from backup back to the store.
func restoreBackup(storeDir string, secrets []string) error {
	backupRoot := filepath.Join(storeDir, rekeyBackupDir)
	total := len(secrets)
	for i, name := range secrets {
		fmt.Fprintf(os.Stderr, "\rRestoring secrets... %d/%d", i+1, total)
		src := filepath.Join(backupRoot, name+".age")
		dst := filepath.Join(storeDir, name+".age")
		data, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("reading backup for %q: %w", name, err)
		}
		if err := os.WriteFile(dst, data, 0600); err != nil {
			fmt.Fprintln(os.Stderr)
			return fmt.Errorf("restoring %q: %w", name, err)
		}
	}
	fmt.Fprintln(os.Stderr)
	return nil
}

// cleanupRekey removes the backup directory and journal file.
func cleanupRekey(storeDir string) {
	os.RemoveAll(filepath.Join(storeDir, rekeyBackupDir))
	os.Remove(filepath.Join(storeDir, rekeyJournalFile))
}

// flushAgentCache stops the agent to clear stale cached passphrases after rekey.
func flushAgentCache() {
	c := agent.NewClient()
	if c.Ping() {
		c.Shutdown()
	}
}

func handleKeyfileGenerate(p *goopt.Parser, _ *goopt.Command) error {
	cfg, ok := goopt.GetStructCtxAs[*options.Config](p)
	if !ok {
		return errNoConfig
	}

	path := cfg.KeyfileMgmt.Generate.Path
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file %q already exists; refusing to overwrite", path)
	}

	if err := generateKeyfile(path, cfg.KeyfileMgmt.Generate.Bytes); err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading generated keyfile: %w", err)
	}
	fp := age.KeyfileFingerprint(content)
	fmt.Printf("Generated keyfile at %s\nFingerprint: %s\n", path, fp)
	return nil
}
