package main

import (
	"context"
	"os"

	"github.com/Wei-Shaw/sub2api/internal/ccgo/shellwrapper"
)

func main() {
	os.Exit(shellwrapper.Runner{Args: os.Args[1:]}.Run(context.Background()))
}
