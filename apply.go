package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// apply reads state and sets up repositories
func apply(state *State, stderr io.Writer) error {
	// Sort repositories so main checkouts come before their worktrees
	repos := make([]Repository, len(state.Repositories))
	copy(repos, state.Repositories)

	sort.Slice(repos, func(i, j int) bool {
		// Main checkouts come first
		if repos[i].IsWorktree != repos[j].IsWorktree {
			return !repos[i].IsWorktree
		}
		return repos[i].Path < repos[j].Path
	})

	for _, repo := range repos {
		if err := applyRepo(repo, stderr); err != nil {
			fmt.Fprintf(stderr, "error: %s: %v\n", repo.Path, err)
			// Continue with other repos
		}
	}

	return nil
}

// applyRepo sets up a single repository
func applyRepo(repo Repository, stderr io.Writer) error {
	// Check if path already exists
	if _, err := os.Stat(repo.Path); err == nil {
		fmt.Fprintf(stderr, "warning: %s already exists, skipping\n", repo.Path)
		return nil
	}

	if repo.IsWorktree {
		return applyWorktree(repo, stderr)
	}
	return applyMainCheckout(repo, stderr)
}

// applyMainCheckout clones and checks out a main repository
func applyMainCheckout(repo Repository, stderr io.Writer) error {
	if repo.RemoteURL == "" {
		return fmt.Errorf("no remote URL for main checkout")
	}

	fmt.Fprintf(stderr, "cloning %s from %s\n", repo.Path, repo.RemoteURL)

	// Create parent directory if needed
	parent := filepath.Dir(repo.Path)
	if parent != "" && parent != "." {
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}
	}

	// Clone the repository
	if err := clone(repo.RemoteURL, repo.Path); err != nil {
		return fmt.Errorf("failed to clone: %w", err)
	}

	// Checkout the correct branch and commit
	if err := checkout(repo.Path, repo.Branch, repo.Commit); err != nil {
		return fmt.Errorf("failed to checkout: %w", err)
	}

	fmt.Fprintf(stderr, "  checked out %s at %s\n", repo.Branch, repo.Commit[:12])
	return nil
}

// applyWorktree adds a worktree to an existing repository
func applyWorktree(repo Repository, stderr io.Writer) error {
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

	// Add the worktree
	if err := addWorktree(mainPath, absWorktreePath, repo.Branch, repo.Commit); err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	fmt.Fprintf(stderr, "  checked out %s at %s\n", repo.Branch, repo.Commit[:12])
	return nil
}
