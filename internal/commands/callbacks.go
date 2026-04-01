package commands

import "github.com/napalu/sympa/internal/options"

// AssignCallbacks wires all command handlers to the config struct.
func AssignCallbacks(cfg *options.Config) {
	cfg.Init.Exec = handleInit
	cfg.Ls.Exec = handleLs
	cfg.Show.Exec = handleShow
	cfg.Insert.Exec = handleInsert
	cfg.Edit.Exec = handleEdit
	cfg.Rm.Exec = handleRm
	cfg.Mv.Exec = handleMv
	cfg.Cp.Exec = handleCp
	cfg.Find.Exec = handleFind
	cfg.Grep.Exec = handleGrep
	cfg.Generate.Exec = handleGenerate
	cfg.Totp.Exec = handleTotp
	cfg.Git.Exec = handleGit
	cfg.Agent.Start.Exec = handleAgentStart
	cfg.Agent.Stop.Exec = handleAgentStop
	cfg.Agent.Status.Exec = handleAgentStatus
	cfg.KeyfileMgmt.Verify.Exec = handleKeyfileVerify
	cfg.KeyfileMgmt.Generate.Exec = handleKeyfileGenerate
	cfg.KeyfileMgmt.Rekey.Exec = handleKeyfileRekey
	cfg.Completion.Exec = handleCompletion
}
