package options

import (
	"fmt"

	"github.com/napalu/goopt/v2"
)

// Config defines the entire sympa CLI structure.
type Config struct {
	Keyfile string `goopt:"kind:flag;short:k;desc:Path to keyfile for passphrase derivation"`

	Init struct {
		Exec goopt.CommandFunc
	} `goopt:"kind:command;desc:Initialize a new secret store"`

	Ls struct {
		Subfolder string `goopt:"pos:0;desc:Subfolder to list"`
		Exec      goopt.CommandFunc
	} `goopt:"kind:command;desc:List secrets in the store"`

	Show struct {
		Name  string `goopt:"pos:0;desc:Secret name;required:true"`
		Clip  bool   `goopt:"short:c;desc:Copy to clipboard (auto-clears after 45s)"`
		Field string `goopt:"short:f;desc:Show a specific field value"`
		Exec  goopt.CommandFunc
	} `goopt:"kind:command;desc:Decrypt and display a secret"`

	Insert struct {
		Name      string `goopt:"pos:0;desc:Secret name;required:true"`
		Multiline bool   `goopt:"short:m;desc:Read until EOF (Ctrl+D)"`
		Force     bool   `goopt:"short:f;desc:Overwrite existing secret without confirmation"`
		NoVerify  bool   `goopt:"desc:Skip passphrase verification against existing secrets"`
		Exec      goopt.CommandFunc
	} `goopt:"kind:command;desc:Insert a new secret"`

	Edit struct {
		Name string `goopt:"pos:0;desc:Secret name;required:true"`
		Exec goopt.CommandFunc
	} `goopt:"kind:command;desc:Edit a secret with $EDITOR"`

	Rm struct {
		Name      string `goopt:"pos:0;desc:Secret name or folder;required:true"`
		Force     bool   `goopt:"short:f;desc:Skip confirmation prompt"`
		Recursive bool   `goopt:"short:r;desc:Remove directory recursively"`
		Exec      goopt.CommandFunc
	} `goopt:"kind:command;desc:Remove a secret"`

	Mv struct {
		Source string `goopt:"pos:0;desc:Source secret;required:true"`
		Dest   string `goopt:"pos:1;desc:Destination;required:true"`
		Force  bool   `goopt:"short:f;desc:Overwrite without confirmation"`
		Exec   goopt.CommandFunc
	} `goopt:"kind:command;desc:Move or rename a secret"`

	Cp struct {
		Source string `goopt:"pos:0;desc:Source secret;required:true"`
		Dest   string `goopt:"pos:1;desc:Destination;required:true"`
		Force  bool   `goopt:"short:f;desc:Overwrite without confirmation"`
		Exec   goopt.CommandFunc
	} `goopt:"kind:command;desc:Copy a secret (re-encrypts with new passphrase)"`

	Find struct {
		Pattern string `goopt:"pos:0;desc:Search pattern;required:true"`
		Exec    goopt.CommandFunc
	} `goopt:"kind:command;desc:Find secrets by name"`

	Grep struct {
		Pattern string `goopt:"pos:0;desc:Search string;required:true"`
		Exec    goopt.CommandFunc
	} `goopt:"kind:command;desc:Search within decrypted secret contents"`

	Generate struct {
		Name      string `goopt:"pos:0;desc:Secret name;required:true"`
		Length    int    `goopt:"short:l;default:32;desc:Password length"`
		NoSymbols bool   `goopt:"short:n;desc:Generate alphanumeric only"`
		Clip      bool   `goopt:"short:c;desc:Copy to clipboard (auto-clears after 45s)"`
		Force     bool   `goopt:"short:f;desc:Overwrite without confirmation"`
		NoVerify  bool   `goopt:"desc:Skip passphrase verification against existing secrets"`
		Exec      goopt.CommandFunc
	} `goopt:"kind:command;desc:Generate and store a random password"`

	Totp struct {
		Name string `goopt:"pos:0;desc:Secret name;required:true"`
		Clip bool   `goopt:"short:c;desc:Copy TOTP code to clipboard"`
		Exec goopt.CommandFunc
	} `goopt:"kind:command;desc:Generate a TOTP code from a stored secret"`

	Git struct {
		Exec goopt.CommandFunc
	} `goopt:"kind:command;greedy:true;desc:Run git commands on the store"`

	Agent struct {
		Start struct {
			Exec goopt.CommandFunc
		} `goopt:"kind:command;desc:Start the passphrase caching agent"`
		Stop struct {
			Exec goopt.CommandFunc
		} `goopt:"kind:command;desc:Stop the passphrase caching agent"`
		Status struct {
			Exec goopt.CommandFunc
		} `goopt:"kind:command;desc:Show agent status"`
	} `goopt:"kind:command;desc:Manage the passphrase caching agent"`

	KeyfileMgmt struct {
		Verify struct {
			Exec goopt.CommandFunc
		} `goopt:"kind:command;desc:Check keyfile matches store fingerprint"`
		Generate struct {
			Path  string `goopt:"pos:0;desc:Path for the new keyfile;required:true"`
			Bytes int    `goopt:"short:b;default:32;desc:Keyfile size in bytes (min 32)"`
			Exec  goopt.CommandFunc
		} `goopt:"kind:command;desc:Generate a new random keyfile"`
		Rekey struct {
			NewKeyfile string `goopt:"pos:0;desc:Path to new keyfile"`
			Remove     bool   `goopt:"short:r;desc:Remove keyfile from store"`
			Passphrase bool   `goopt:"short:p;desc:Change passphrase"`
			Resume     bool   `goopt:"desc:Resume an interrupted rekey"`
			Abort      bool   `goopt:"desc:Abort an interrupted rekey and restore backup"`
			Exec       goopt.CommandFunc
		} `goopt:"kind:command;desc:Re-encrypt all secrets with new keyfile or passphrase"`
	} `goopt:"kind:command;name:keyfile;desc:Keyfile management commands"`

	Completion struct {
		Shell string `goopt:"pos:0;desc:Shell type (bash, zsh, fish);required:true;validators:isoneof(bash,zsh,fish)"`
		Exec  goopt.CommandFunc
	} `goopt:"kind:command;desc:Generate shell completion script"`
}

// New creates a goopt parser from the given Config.
func New(cfg *Config, version, commit string) (*goopt.Parser, error) {
	return goopt.NewParserFromStruct(cfg,
		goopt.WithVersionFunc(func() string {
			return fmt.Sprintf("%s (%s)", version, commit)
		}),
		goopt.WithShowVersionInHelp(true),
		goopt.WithHelpStyle(goopt.HelpStyleCompact),
		goopt.WithFlagNameConverter(goopt.ToKebabCase),
		goopt.WithEnvVarPrefix("SYMPA_"),
		goopt.WithEnvNameConverter(goopt.ToKebabCase),
		goopt.WithAllowUnknownFlags(true),
		goopt.WithTreatUnknownAsPositionals(true),
		goopt.WithAutoLanguage(false),
	)
}
