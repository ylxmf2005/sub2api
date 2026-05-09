package main

import (
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
		return fmt.Errorf("usage: ccgo login <token> | ccgo <local_path>")
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
		info, err := ccgocli.ResolveLocalPath(args[0])
		if err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}
}
