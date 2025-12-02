package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// capture scans for git repositories and returns the state
func capture(stderr io.Writer, verbose bool) (*State, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if verbose {
		fmt.Fprintf(stderr, "starting capture from %s\n", cwd)
	}

	repos := make(map[string]*Repository)

	// Search upward
	if verbose {
		fmt.Fprintf(stderr, "searching parent directories\n")
	}
	searchUpward(cwd, repos, stderr, verbose)

	// Search current directory and downward
	if verbose {
		fmt.Fprintf(stderr, "searching current directory and subdirectories\n")
	}
	searchDownward(cwd, repos, stderr, verbose)

	// Convert map to sorted slice
	state := &State{
		Repositories: make([]Repository, 0, len(repos)),
	}

	// Get all paths and sort them
	paths := make([]string, 0, len(repos))
	for p := range repos {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		state.Repositories = append(state.Repositories, *repos[p])
	}

	if verbose {
		fmt.Fprintf(stderr, "found %d repositories\n", len(state.Repositories))
	}

	return state, nil
}

// searchUpward walks parent directories looking for git repos
func searchUpward(startPath string, repos map[string]*Repository, stderr io.Writer, verbose bool) {
	cwd, _ := os.Getwd()
	current := startPath

	for {
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			if verbose {
				fmt.Fprintf(stderr, "  reached filesystem root\n")
			}
			break
		}

		if verbose {
			fmt.Fprintf(stderr, "  checking %s\n", parent)
		}

		if isGitRepo(parent) {
			relPath, err := filepath.Rel(cwd, parent)
			if err != nil {
				relPath = parent
			}
			if verbose {
				fmt.Fprintf(stderr, "  found repository: %s\n", relPath)
			}
			addRepo(parent, relPath, repos, stderr, verbose)
		}

		current = parent
	}
}

// searchDownward walks subdirectories looking for git repos
func searchDownward(startPath string, repos map[string]*Repository, stderr io.Writer, verbose bool) {
	cwd, _ := os.Getwd()

	filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if verbose {
				fmt.Fprintf(stderr, "  skipping %s: %v\n", path, err)
			}
			return nil // Skip directories we can't read
		}

		if !d.IsDir() {
			return nil
		}

		// Skip .git directories
		if d.Name() == ".git" {
			return filepath.SkipDir
		}

		if isGitRepo(path) {
			relPath, err := filepath.Rel(cwd, path)
			if err != nil {
				relPath = path
			}
			if relPath == "" {
				relPath = "."
			}
			if verbose {
				fmt.Fprintf(stderr, "  found repository: %s\n", relPath)
			}
			addRepo(path, relPath, repos, stderr, verbose)

			// Don't descend into git repos (they handle their own subdirs)
			// But we do want to find nested repos, so continue
		}

		return nil
	})
}

// addRepo creates a Repository entry and adds it to the map
func addRepo(absPath, relPath string, repos map[string]*Repository, stderr io.Writer, verbose bool) {
	// Skip if already processed
	if _, exists := repos[relPath]; exists {
		if verbose {
			fmt.Fprintf(stderr, "    skipping %s (already processed)\n", relPath)
		}
		return
	}

	if verbose {
		fmt.Fprintf(stderr, "    processing %s\n", relPath)
	}

	// Check for uncommitted changes and warn
	if hasUncommittedChanges(absPath) {
		fmt.Fprintf(stderr, "warning: %s has uncommitted changes\n", relPath)
	}

	isWt, mainPath := isWorktree(absPath)

	if verbose {
		if isWt {
			fmt.Fprintf(stderr, "    detected as worktree (main checkout: %s)\n", mainPath)
		} else {
			fmt.Fprintf(stderr, "    detected as main checkout\n")
		}
	}

	branch := getBranch(absPath)
	commit := getCommit(absPath)

	if verbose {
		fmt.Fprintf(stderr, "    branch: %s, commit: %s\n", branch, commit[:12])
	}

	repo := &Repository{
		Path:       relPath,
		Branch:     branch,
		Commit:     commit,
		IsWorktree: isWt,
	}

	if isWt {
		repo.MainCheckoutPath = &mainPath
	} else {
		// Only get remote URL for main checkouts
		repo.RemoteURL = getRemoteURL(absPath)
		if verbose && repo.RemoteURL != "" {
			fmt.Fprintf(stderr, "    remote: %s\n", repo.RemoteURL)
		}
	}

	repos[relPath] = repo
}
