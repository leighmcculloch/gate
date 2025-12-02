package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"4d63.com/testcli"
	"github.com/stretchr/testify/assert"
)

func setupGit(t *testing.T) {
	dir := testcli.MkdirTemp(t)
	os.Setenv("HOME", dir)
	testcli.Exec(t, "git config --global user.email 'tests@example.com'")
	testcli.Exec(t, "git config --global user.name 'Tests'")
	testcli.Exec(t, "git config --global init.defaultBranch main")
}

func gitExec(t *testing.T, command string) string {
	_, stdout, _ := testcli.Exec(t, command)
	return strings.TrimSpace(stdout)
}

func TestCaptureNoRepos(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)
	assert.Equal(t, `{
  "repositories": []
}
`, stdout)
}

func TestCaptureSingleRepo(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")

	commit := gitExec(t, "git rev-parse HEAD")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)
	assert.Equal(t, fmt.Sprintf(`{
  "repositories": [
    {
      "path": ".",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    }
  ]
}
`, commit), stdout)
}

func TestCaptureSingleRepoWithRemote(t *testing.T) {
	setupGit(t)

	// Create bare remote
	remote := testcli.MkdirTemp(t)
	testcli.Chdir(t, remote)
	testcli.Exec(t, "git init --bare")

	// Create local repo with remote
	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)
	testcli.Exec(t, "git init")
	testcli.Exec(t, "git remote add origin "+remote)
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")

	commit := gitExec(t, "git rev-parse HEAD")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)
	assert.Equal(t, fmt.Sprintf(`{
  "repositories": [
    {
      "path": ".",
      "remote_url": "%s",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    }
  ]
}
`, remote, commit), stdout)
}

func TestCaptureUncommittedChangesWarning(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")
	// Create uncommitted change
	testcli.WriteFile(t, "file2", []byte("uncommitted"))

	commit := gitExec(t, "git rev-parse HEAD")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "warning: . has uncommitted changes\n", stderr)
	assert.Equal(t, fmt.Sprintf(`{
  "repositories": [
    {
      "path": ".",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    }
  ]
}
`, commit), stdout)
}

func TestCaptureMultipleRepos(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)

	// Create repo1
	testcli.Mkdir(t, "repo1")
	testcli.Chdir(t, "repo1")
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")
	commit1 := gitExec(t, "git rev-parse HEAD")
	testcli.Chdir(t, "..")

	// Create repo2
	testcli.Mkdir(t, "repo2")
	testcli.Chdir(t, "repo2")
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")
	commit2 := gitExec(t, "git rev-parse HEAD")
	testcli.Chdir(t, "..")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)
	assert.Equal(t, fmt.Sprintf(`{
  "repositories": [
    {
      "path": "repo1",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    },
    {
      "path": "repo2",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    }
  ]
}
`, commit1, commit2), stdout)
}

func TestCaptureWorktree(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)

	// Create main repo
	testcli.Mkdir(t, "main-repo")
	testcli.Chdir(t, "main-repo")
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")
	commit := gitExec(t, "git rev-parse HEAD")
	// Add worktree with new branch
	testcli.Exec(t, "git worktree add -b feature-branch ../worktree-branch")
	testcli.Chdir(t, "..")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)
	assert.Equal(t, fmt.Sprintf(`{
  "repositories": [
    {
      "path": "main-repo",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    },
    {
      "path": "worktree-branch",
      "branch": "feature-branch",
      "commit": "%s",
      "is_worktree": true,
      "main_checkout_path": "../main-repo"
    }
  ]
}
`, commit, commit), stdout)
}

func TestApplyCloneRepo(t *testing.T) {
	setupGit(t)

	// Create a bare remote to clone from
	remote := testcli.MkdirTemp(t)
	testcli.Chdir(t, remote)
	testcli.Exec(t, "git init --bare")

	// Create a temporary repo to push to the remote
	tmpRepo := testcli.MkdirTemp(t)
	testcli.Chdir(t, tmpRepo)
	testcli.Exec(t, "git init")
	testcli.Exec(t, "git remote add origin "+remote)
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")
	testcli.Exec(t, "git push -u origin main")

	// Get the commit SHA
	commit := gitExec(t, "git rev-parse HEAD")

	// Create target directory for apply
	targetDir := testcli.MkdirTemp(t)
	testcli.Chdir(t, targetDir)

	// Create JSON input
	jsonInput := fmt.Sprintf(`{
  "repositories": [
    {
      "path": "cloned-repo",
      "remote_url": "%s",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    }
  ]
}`, remote, commit)

	args := []string{"gate", "apply"}
	exitCode, _, stderr := testcli.Main(t, args, strings.NewReader(jsonInput), run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, fmt.Sprintf(`cloning cloned-repo from %s
  checked out main at %s
`, remote, commit[:12]), stderr)

	// Verify the repo was created
	testcli.Chdir(t, "cloned-repo")
	actualCommit := gitExec(t, "git rev-parse HEAD")
	assert.Equal(t, commit, actualCommit)
}

func TestApplySkipsExistingRepo(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)

	// Create an existing directory
	testcli.Mkdir(t, "existing-repo")

	jsonInput := `{
  "repositories": [
    {
      "path": "existing-repo",
      "remote_url": "https://example.com/repo.git",
      "branch": "main",
      "commit": "abc123abc123abc123abc123abc123abc123abc1",
      "is_worktree": false,
      "main_checkout_path": null
    }
  ]
}`

	args := []string{"gate", "apply"}
	exitCode, _, stderr := testcli.Main(t, args, strings.NewReader(jsonInput), run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "warning: existing-repo already exists, skipping\n", stderr)
}

func TestApplyWorktree(t *testing.T) {
	setupGit(t)

	// Create a bare remote
	remote := testcli.MkdirTemp(t)
	testcli.Chdir(t, remote)
	testcli.Exec(t, "git init --bare")

	// Create and push to remote
	tmpRepo := testcli.MkdirTemp(t)
	testcli.Chdir(t, tmpRepo)
	testcli.Exec(t, "git init")
	testcli.Exec(t, "git remote add origin "+remote)
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")
	testcli.Exec(t, "git push -u origin main")
	commit := gitExec(t, "git rev-parse HEAD")

	// Create target directory
	targetDir := testcli.MkdirTemp(t)
	testcli.Chdir(t, targetDir)

	// Create JSON input with main repo and worktree
	jsonInput := fmt.Sprintf(`{
  "repositories": [
    {
      "path": "main-repo",
      "remote_url": "%s",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    },
    {
      "path": "worktree-dir",
      "branch": "feature",
      "commit": "%s",
      "is_worktree": true,
      "main_checkout_path": "../main-repo"
    }
  ]
}`, remote, commit, commit)

	args := []string{"gate", "apply"}
	exitCode, _, stderr := testcli.Main(t, args, strings.NewReader(jsonInput), run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, fmt.Sprintf(`cloning main-repo from %s
  checked out main at %s
adding worktree worktree-dir from main-repo
  checked out feature at %s
`, remote, commit[:12], commit[:12]), stderr)

	// Verify worktree was created
	testcli.Chdir(t, "worktree-dir")
	branch := gitExec(t, "git rev-parse --abbrev-ref HEAD")
	assert.Equal(t, "feature", branch)
}

func TestCaptureNestedRepos(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)

	// Create outer repo
	testcli.Mkdir(t, "outer")
	testcli.Chdir(t, "outer")
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Outer commit'")
	outerCommit := gitExec(t, "git rev-parse HEAD")

	// Create nested inner repo
	testcli.Mkdir(t, "inner")
	testcli.Chdir(t, "inner")
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file2", []byte("inner content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Inner commit'")
	innerCommit := gitExec(t, "git rev-parse HEAD")
	testcli.Chdir(t, "../..")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	// Outer repo has uncommitted changes (the inner directory)
	assert.Equal(t, "warning: outer has uncommitted changes\n", stderr)
	assert.Equal(t, fmt.Sprintf(`{
  "repositories": [
    {
      "path": "outer",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    },
    {
      "path": "outer/inner",
      "branch": "main",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    }
  ]
}
`, outerCommit, innerCommit), stdout)
}

func TestCaptureDetachedHead(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")
	commit := gitExec(t, "git rev-parse HEAD")
	// Detach HEAD
	testcli.Exec(t, "git checkout --detach HEAD")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)
	assert.Equal(t, fmt.Sprintf(`{
  "repositories": [
    {
      "path": ".",
      "branch": "HEAD",
      "commit": "%s",
      "is_worktree": false,
      "main_checkout_path": null
    }
  ]
}
`, commit), stdout)
}
