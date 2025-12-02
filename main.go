package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	rootCmd := &cobra.Command{
		Use:   "gate",
		Short: "Capture and restore git repository state",
		Long:  "A tool to capture the state of git repositories and worktrees, and restore them on another system.",
	}

	captureCmd := &cobra.Command{
		Use:   "capture",
		Short: "Capture git repository state to JSON",
		Long:  "Scan directories above, below, and at the current location for git repositories and output their state as JSON.",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := capture(stderr)
			if err != nil {
				return err
			}

			encoder := json.NewEncoder(stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(state)
		},
	}

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply git repository state from JSON",
		Long:  "Read JSON from stdin and set up repositories and worktrees accordingly.",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := io.ReadAll(stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}

			var state State
			if err := json.Unmarshal(data, &state); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			return apply(&state, stderr)
		},
	}

	rootCmd.AddCommand(captureCmd, applyCmd)
	rootCmd.SetArgs(args[1:])
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}
