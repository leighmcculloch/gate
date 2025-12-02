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
	var verbose bool

	rootCmd := &cobra.Command{
		Use:   "gate",
		Short: "Capture and restore git repository state",
		Long:  "A tool to capture the state of git repositories and worktrees, and restore them on another system.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	captureCmd := &cobra.Command{
		Use:   "capture",
		Short: "Capture git repository state to JSON",
		Long:  "Scan directories above, below, and at the current location for git repositories and output their state as JSON.",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := capture(stderr, verbose)
			if err != nil {
				return err
			}

			if verbose {
				fmt.Fprintf(stderr, "writing JSON output\n")
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
			if verbose {
				fmt.Fprintf(stderr, "reading JSON from stdin\n")
			}
			data, err := io.ReadAll(stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}

			if verbose {
				fmt.Fprintf(stderr, "parsing JSON (%d bytes)\n", len(data))
			}
			var state State
			if err := json.Unmarshal(data, &state); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			if verbose {
				fmt.Fprintf(stderr, "found %d repositories to apply\n", len(state.Repositories))
			}
			return apply(&state, stderr, verbose)
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
