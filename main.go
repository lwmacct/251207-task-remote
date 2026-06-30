package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lwmacct/251207-task-remote/internal/appcmd"
)

func main() {
	cmd := appcmd.New()
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
