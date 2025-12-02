package main

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// git runs a git command in the specified directory and returns stdout
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// isGitRepo checks if a directory is a git repository
func isGitRepo(path string) bool {
	_, err := git(path, "rev-parse", "--git-dir")
	return err == nil
}

// isWorktree checks if a directory is a worktree (not the main checkout)
// Returns true if worktree, and the path to the main checkout relative to the worktree
func isWorktree(path string) (bool, string) {
	// Get the git common dir (shared across worktrees)
	commonDir, err := git(path, "rev-parse", "--git-common-dir")
	if err != nil {
		return false, ""
	}

	// Get the git dir for this specific worktree
	gitDir, err := git(path, "rev-parse", "--git-dir")
	if err != nil {
		return false, ""
	}

	// If they're the same (both ".git"), it's the main checkout
	// If git-dir is inside git-common-dir/worktrees/, it's a worktree
	if commonDir == gitDir || commonDir == ".git" {
		return false, ""
	}

	// It's a worktree - find the main checkout path
	// The common dir points to the .git folder of the main checkout
	absCommonDir := commonDir
	if !filepath.IsAbs(commonDir) {
		absCommonDir = filepath.Join(path, commonDir)
	}
	absCommonDir, _ = filepath.Abs(absCommonDir)

	// Main checkout is the parent of the .git folder
	mainCheckout := filepath.Dir(absCommonDir)

	// Make it relative to the worktree
	absPath, _ := filepath.Abs(path)
	relPath, err := filepath.Rel(absPath, mainCheckout)
	if err != nil {
		return true, mainCheckout
	}

	return true, relPath
}

// getBranch returns the current branch name, or "HEAD" if detached
func getBranch(path string) string {
	branch, err := git(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return branch
}

// getCommit returns the current HEAD commit SHA
func getCommit(path string) string {
	commit, err := git(path, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return commit
}

// getRemoteURL returns the origin remote URL
func getRemoteURL(path string) string {
	url, err := git(path, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return url
}

// hasUncommittedChanges checks if there are uncommitted changes
func hasUncommittedChanges(path string) bool {
	status, err := git(path, "status", "--porcelain")
	if err != nil {
		return false
	}
	return status != ""
}

// clone clones a repository
func clone(url, path string) error {
	cmd := exec.Command("git", "clone", url, path)
	return cmd.Run()
}

// checkout checks out a specific branch and resets to a commit
func checkout(path, branch, commit string) error {
	// Try to checkout the branch first
	if branch != "" && branch != "HEAD" {
		// Try checking out existing branch
		if err := exec.Command("git", "-C", path, "checkout", branch).Run(); err != nil {
			// Branch doesn't exist locally, create it
			exec.Command("git", "-C", path, "checkout", "-b", branch).Run()
		}
	}

	// Reset to the specific commit
	if commit != "" {
		cmd := exec.Command("git", "-C", path, "reset", "--hard", commit)
		return cmd.Run()
	}
	return nil
}

// addWorktree adds a new worktree
func addWorktree(mainPath, worktreePath, branch, commit string) error {
	// Create the worktree at the specified branch
	cmd := exec.Command("git", "-C", mainPath, "worktree", "add", worktreePath, branch)
	if err := cmd.Run(); err != nil {
		// If branch doesn't exist, create it
		cmd = exec.Command("git", "-C", mainPath, "worktree", "add", "-b", branch, worktreePath)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	// Reset to the specific commit if provided
	if commit != "" {
		cmd = exec.Command("git", "-C", worktreePath, "reset", "--hard", commit)
		return cmd.Run()
	}
	return nil
}
