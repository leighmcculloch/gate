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
func capture(stderr io.Writer) (*State, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	repos := make(map[string]*Repository)

	// Search upward
	searchUpward(cwd, repos, stderr)

	// Search current directory and downward
	searchDownward(cwd, repos, stderr)

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

	return state, nil
}

// searchUpward walks parent directories looking for git repos
func searchUpward(startPath string, repos map[string]*Repository, stderr io.Writer) {
	cwd, _ := os.Getwd()
	current := startPath

	for {
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			break
		}

		if isGitRepo(parent) {
			relPath, err := filepath.Rel(cwd, parent)
			if err != nil {
				relPath = parent
			}
			addRepo(parent, relPath, repos, stderr)
		}

		current = parent
	}
}

// searchDownward walks subdirectories looking for git repos
func searchDownward(startPath string, repos map[string]*Repository, stderr io.Writer) {
	cwd, _ := os.Getwd()

	filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
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
			addRepo(path, relPath, repos, stderr)

			// Don't descend into git repos (they handle their own subdirs)
			// But we do want to find nested repos, so continue
		}

		return nil
	})
}

// addRepo creates a Repository entry and adds it to the map
func addRepo(absPath, relPath string, repos map[string]*Repository, stderr io.Writer) {
	// Skip if already processed
	if _, exists := repos[relPath]; exists {
		return
	}

	// Check for uncommitted changes and warn
	if hasUncommittedChanges(absPath) {
		fmt.Fprintf(stderr, "warning: %s has uncommitted changes\n", relPath)
	}

	isWt, mainPath := isWorktree(absPath)

	repo := &Repository{
		Path:       relPath,
		Branch:     getBranch(absPath),
		Commit:     getCommit(absPath),
		IsWorktree: isWt,
	}

	if isWt {
		repo.MainCheckoutPath = &mainPath
	} else {
		// Only get remote URL for main checkouts
		repo.RemoteURL = getRemoteURL(absPath)
	}

	repos[relPath] = repo
}
