package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	ccgocli "github.com/Wei-Shaw/sub2api/internal/ccgo/cli"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ccgo login <token> | ccgo [--no-agent] <local_path>")
	}
	switch args[0] {
	case "login":
		if len(args) < 2 {
			return fmt.Errorf("device login is not implemented in this slice; use ccgo login <token>")
		}
		result, err := ccgocli.LoginWithToken(ccgocli.LoginOptions{Token: args[1]})
		if err != nil {
			return err
		}
		fmt.Printf("Logged in to %s with token %s\n", result.Server, result.RedactedToken)
		return nil
	case "status", "stop":
		return fmt.Errorf("ccgo %s is not implemented yet", args[0])
	default:
		noAgent := false
		localPath := args[0]
		if args[0] == "--no-agent" {
			if len(args) < 2 {
				return fmt.Errorf("usage: ccgo --no-agent <local_path>")
			}
			noAgent = true
			localPath = args[1]
		}
		result, err := ccgocli.Start(context.Background(), ccgocli.StartOptions{LocalPath: localPath, NoAgent: noAgent})
		if err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(struct {
			LocalRoot         string `json:"local_root"`
			WorkspaceID       int64  `json:"workspace_id"`
			LocalRootRedacted string `json:"local_root_redacted"`
			AgentConnected    bool   `json:"agent_connected"`
			RunID             string `json:"run_id,omitempty"`
			RunStatus         string `json:"run_status,omitempty"`
			ReusedRun         bool   `json:"reused_run,omitempty"`
		}{
			LocalRoot:         result.LocalRoot,
			WorkspaceID:       result.WorkspaceID,
			LocalRootRedacted: result.LocalRootRedacted,
			AgentConnected:    result.AgentConnected,
			RunID:             result.RunID,
			RunStatus:         result.RunStatus,
			ReusedRun:         result.ReusedRun,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		if noAgent {
			fmt.Println("Workspace resolved without starting the local agent.")
		} else {
			fmt.Println("Local agent connection ended.")
		}
		return nil
	}
}
