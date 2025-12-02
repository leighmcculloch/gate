package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// apply reads state and sets up repositories
func apply(state *State, stderr io.Writer, verbose bool) error {
	// Sort repositories so main checkouts come before their worktrees
	repos := make([]Repository, len(state.Repositories))
	copy(repos, state.Repositories)

	if verbose {
		fmt.Fprintf(stderr, "sorting repositories (main checkouts before worktrees)\n")
	}

	sort.Slice(repos, func(i, j int) bool {
		// Main checkouts come first
		if repos[i].IsWorktree != repos[j].IsWorktree {
			return !repos[i].IsWorktree
		}
		return repos[i].Path < repos[j].Path
	})

	for i, repo := range repos {
		if verbose {
			fmt.Fprintf(stderr, "processing repository %d/%d: %s\n", i+1, len(repos), repo.Path)
		}
		if err := applyRepo(repo, stderr, verbose); err != nil {
			fmt.Fprintf(stderr, "error: %s: %v\n", repo.Path, err)
			// Continue with other repos
		}
	}

	if verbose {
		fmt.Fprintf(stderr, "apply complete\n")
	}

	return nil
}

// applyRepo sets up a single repository
func applyRepo(repo Repository, stderr io.Writer, verbose bool) error {
	// Check if path already exists
	if _, err := os.Stat(repo.Path); err == nil {
		fmt.Fprintf(stderr, "warning: %s already exists, skipping\n", repo.Path)
		return nil
	}

	if repo.IsWorktree {
		return applyWorktree(repo, stderr, verbose)
	}
	return applyMainCheckout(repo, stderr, verbose)
}

// applyMainCheckout clones and checks out a main repository
func applyMainCheckout(repo Repository, stderr io.Writer, verbose bool) error {
	if repo.RemoteURL == "" {
		return fmt.Errorf("no remote URL for main checkout")
	}

	fmt.Fprintf(stderr, "cloning %s from %s\n", repo.Path, repo.RemoteURL)

	// Create parent directory if needed
	parent := filepath.Dir(repo.Path)
	if parent != "" && parent != "." {
		if verbose {
			fmt.Fprintf(stderr, "  creating parent directory: %s\n", parent)
		}
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
	}

	// Clone the repository
	if verbose {
		fmt.Fprintf(stderr, "  running git clone\n")
	}
	if err := clone(repo.RemoteURL, repo.Path); err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}

	// Checkout the correct branch and commit
	if verbose {
		fmt.Fprintf(stderr, "  checking out branch %s\n", repo.Branch)
		fmt.Fprintf(stderr, "  resetting to commit %s\n", repo.Commit)
	}
	if err := checkout(repo.Path, repo.Branch, repo.Commit); err != nil {
		return fmt.Errorf("failed to checkout: %w", err)
	}

	fmt.Fprintf(stderr, "  checked out %s at %s\n", repo.Branch, repo.Commit[:12])
	return nil
}

// applyWorktree adds a worktree to an existing repository
func applyWorktree(repo Repository, stderr io.Writer, verbose bool) error {
	if repo.MainCheckoutPath == nil {
		return fmt.Errorf("no main checkout path for worktree")
	}

	// Calculate the absolute path to the main checkout
	mainPath := *repo.MainCheckoutPath
	if !filepath.IsAbs(mainPath) {
		// MainCheckoutPath is relative to the worktree path
		mainPath = filepath.Join(repo.Path, mainPath)
	}
	mainPath = filepath.Clean(mainPath)

	if verbose {
		fmt.Fprintf(stderr, "  resolved main checkout path: %s\n", mainPath)
	}

	// Verify main checkout exists
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		return fmt.Errorf("main checkout %s does not exist", mainPath)
	}

	fmt.Fprintf(stderr, "adding worktree %s from %s\n", repo.Path, mainPath)

	// Calculate absolute worktree path for git command
	absWorktreePath, err := filepath.Abs(repo.Path)
	if err != nil {
		absWorktreePath = repo.Path
	}

	if verbose {
		fmt.Fprintf(stderr, "  absolute worktree path: %s\n", absWorktreePath)
		fmt.Fprintf(stderr, "  running git worktree add for branch %s\n", repo.Branch)
	}

	// Add the worktree
	if err := addWorktree(mainPath, absWorktreePath, repo.Branch, repo.Commit); err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	if verbose {
		fmt.Fprintf(stderr, "  resetting to commit %s\n", repo.Commit)
	}

	fmt.Fprintf(stderr, "  checked out %s at %s\n", repo.Branch, repo.Commit[:12])
	return nil
}
