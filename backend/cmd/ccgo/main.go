package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

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
		return fmt.Errorf("usage: ccgo login [token] | ccgo doctor [local_path] | ccgo [--no-agent] <local_path>")
	}
	switch args[0] {
	case "login":
		if len(args) > 2 {
			return fmt.Errorf("usage: ccgo login [token]")
		}
		if len(args) == 1 {
			result, err := ccgocli.Login(context.Background(), ccgocli.LoginOptions{Output: os.Stdout})
			if err != nil {
				return err
			}
			fmt.Printf("Logged in to %s with token %s\n", result.Server, result.RedactedToken)
			return nil
		}
		result, err := ccgocli.LoginWithToken(ccgocli.LoginOptions{Token: args[1]})
		if err != nil {
			return err
		}
		fmt.Printf("Logged in to %s with token %s\n", result.Server, result.RedactedToken)
		return nil
	case "doctor":
		if len(args) > 2 {
			return fmt.Errorf("usage: ccgo doctor [local_path]")
		}
		localPath := ""
		if len(args) == 2 {
			localPath = args[1]
		}
		result, err := ccgocli.Doctor(context.Background(), ccgocli.DoctorOptions{LocalPath: localPath})
		if err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	case "status":
		workspaceID, err := optionalWorkspaceID(args[1:])
		if err != nil {
			return err
		}
		status, err := ccgocli.Status(context.Background(), ccgocli.StatusOptions{WorkspaceID: workspaceID})
		if err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	case "stop":
		workspaceID, err := optionalWorkspaceID(args[1:])
		if err != nil {
			return err
		}
		result, err := ccgocli.Stop(context.Background(), ccgocli.StopOptions{
			WorkspaceID: workspaceID,
			Reason:      "ccgo stop",
		})
		if err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	case "attach":
		if len(args) < 2 {
			return fmt.Errorf("usage: ccgo attach <workspace_id>")
		}
		workspaceID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil || workspaceID <= 0 {
			return fmt.Errorf("workspace id is required")
		}
		return ccgocli.Attach(context.Background(), ccgocli.AttachOptions{
			WorkspaceID: workspaceID,
			Input:       os.Stdin,
			Output:      os.Stdout,
		})
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		result, err := ccgocli.Start(ctx, ccgocli.StartOptions{LocalPath: localPath, NoAgent: noAgent})
		if err != nil {
			return err
		}
		if !noAgent {
			fmt.Fprintf(os.Stderr, "Connected %s as workspace %d. Attaching Claude Code...\n", result.LocalRootRedacted, result.WorkspaceID)
			err := ccgocli.Attach(ctx, ccgocli.AttachOptions{
				WorkspaceID: result.WorkspaceID,
				Input:       os.Stdin,
				Output:      os.Stdout,
			})
			cancel()
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

func optionalWorkspaceID(args []string) (int64, error) {
	if len(args) == 0 {
		return 0, nil
	}
	if len(args) > 1 {
		return 0, fmt.Errorf("usage: ccgo status [workspace_id] | ccgo stop [workspace_id]")
	}
	workspaceID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || workspaceID <= 0 {
		return 0, fmt.Errorf("workspace id is required")
	}
	return workspaceID, nil
}
