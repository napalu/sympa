package commands

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/napalu/sympa/internal/store"
)

// HandleComplete outputs completion candidates for shell scripts.
// Called via "sympa __complete <type> [prefix]" — bypasses the parser for speed.
func HandleComplete(args []string) {
	if len(args) == 0 {
		return
	}

	s := store.New()
	if !s.IsInitialized() {
		return
	}

	prefix := ""
	if len(args) > 1 {
		prefix = args[1]
	}

	switch args[0] {
	case "secrets":
		completeSecrets(s, prefix)
	case "folders":
		completeFolders(s, prefix)
	}
}

func completeSecrets(s *store.Store, prefix string) {
	secrets, err := s.AllSecrets()
	if err != nil {
		return
	}
	for _, sec := range secrets {
		if prefix == "" || strings.HasPrefix(sec, prefix) {
			fmt.Println(sec)
		}
	}
}

func completeFolders(s *store.Store, prefix string) {
	secrets, err := s.AllSecrets()
	if err != nil {
		return
	}
	seen := make(map[string]bool)
	var folders []string
	for _, sec := range secrets {
		dir := filepath.Dir(sec)
		for dir != "." && dir != "" {
			if !seen[dir] {
				seen[dir] = true
				if prefix == "" || strings.HasPrefix(dir, prefix) {
					folders = append(folders, dir)
				}
			}
			dir = filepath.Dir(dir)
		}
	}
	sort.Strings(folders)
	for _, f := range folders {
		fmt.Println(f)
	}
}
