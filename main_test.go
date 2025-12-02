package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"4d63.com/testcli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, "{\n  \"repositories\": []\n}\n", stdout)
}

func TestCaptureSingleRepo(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)

	var state State
	err := json.Unmarshal([]byte(stdout), &state)
	require.NoError(t, err)

	require.Len(t, state.Repositories, 1)
	assert.Equal(t, ".", state.Repositories[0].Path)
	assert.Equal(t, "main", state.Repositories[0].Branch)
	assert.Len(t, state.Repositories[0].Commit, 40) // SHA length
	assert.False(t, state.Repositories[0].IsWorktree)
	assert.Nil(t, state.Repositories[0].MainCheckoutPath)
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

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)

	var state State
	err := json.Unmarshal([]byte(stdout), &state)
	require.NoError(t, err)

	require.Len(t, state.Repositories, 1)
	assert.Equal(t, ".", state.Repositories[0].Path)
	assert.Equal(t, remote, state.Repositories[0].RemoteURL)
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

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stderr, "warning: . has uncommitted changes")

	var state State
	err := json.Unmarshal([]byte(stdout), &state)
	require.NoError(t, err)
	require.Len(t, state.Repositories, 1)
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
	testcli.Chdir(t, "..")

	// Create repo2
	testcli.Mkdir(t, "repo2")
	testcli.Chdir(t, "repo2")
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")
	testcli.Chdir(t, "..")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)

	var state State
	err := json.Unmarshal([]byte(stdout), &state)
	require.NoError(t, err)

	require.Len(t, state.Repositories, 2)
	paths := []string{state.Repositories[0].Path, state.Repositories[1].Path}
	assert.Contains(t, paths, "repo1")
	assert.Contains(t, paths, "repo2")
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
	// Add worktree with new branch
	testcli.Exec(t, "git worktree add -b feature-branch ../worktree-branch")
	testcli.Chdir(t, "..")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)

	var state State
	err := json.Unmarshal([]byte(stdout), &state)
	require.NoError(t, err)

	require.Len(t, state.Repositories, 2)

	// Find the main repo and worktree
	var mainRepo, worktree *Repository
	for i := range state.Repositories {
		if state.Repositories[i].IsWorktree {
			worktree = &state.Repositories[i]
		} else {
			mainRepo = &state.Repositories[i]
		}
	}

	require.NotNil(t, mainRepo)
	require.NotNil(t, worktree)

	assert.Equal(t, "main-repo", mainRepo.Path)
	assert.Equal(t, "main", mainRepo.Branch)
	assert.False(t, mainRepo.IsWorktree)

	assert.Equal(t, "worktree-branch", worktree.Path)
	assert.Equal(t, "feature-branch", worktree.Branch)
	assert.True(t, worktree.IsWorktree)
	require.NotNil(t, worktree.MainCheckoutPath)
	assert.Equal(t, "../main-repo", *worktree.MainCheckoutPath)
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
	state := State{
		Repositories: []Repository{
			{
				Path:       "cloned-repo",
				RemoteURL:  remote,
				Branch:     "main",
				Commit:     commit,
				IsWorktree: false,
			},
		},
	}
	jsonData, _ := json.Marshal(state)

	args := []string{"gate", "apply"}
	exitCode, _, stderr := testcli.Main(t, args, strings.NewReader(string(jsonData)), run)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stderr, "cloning cloned-repo")

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

	state := State{
		Repositories: []Repository{
			{
				Path:       "existing-repo",
				RemoteURL:  "https://example.com/repo.git",
				Branch:     "main",
				Commit:     "abc123",
				IsWorktree: false,
			},
		},
	}
	jsonData, _ := json.Marshal(state)

	args := []string{"gate", "apply"}
	exitCode, _, stderr := testcli.Main(t, args, strings.NewReader(string(jsonData)), run)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stderr, "warning: existing-repo already exists, skipping")
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
	mainPath := "../main-repo"
	state := State{
		Repositories: []Repository{
			{
				Path:       "main-repo",
				RemoteURL:  remote,
				Branch:     "main",
				Commit:     commit,
				IsWorktree: false,
			},
			{
				Path:             "worktree-dir",
				Branch:           "feature",
				Commit:           commit,
				IsWorktree:       true,
				MainCheckoutPath: &mainPath,
			},
		},
	}
	jsonData, _ := json.Marshal(state)

	args := []string{"gate", "apply"}
	exitCode, _, stderr := testcli.Main(t, args, strings.NewReader(string(jsonData)), run)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stderr, "cloning main-repo")
	assert.Contains(t, stderr, "adding worktree worktree-dir")

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

	// Create nested inner repo
	testcli.Mkdir(t, "inner")
	testcli.Chdir(t, "inner")
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file2", []byte("inner content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Inner commit'")
	testcli.Chdir(t, "../..")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	// Outer repo has uncommitted changes (the inner directory)
	assert.Contains(t, stderr, "warning: outer has uncommitted changes")

	var state State
	err := json.Unmarshal([]byte(stdout), &state)
	require.NoError(t, err)

	// Should find both repos
	require.Len(t, state.Repositories, 2)
	paths := []string{state.Repositories[0].Path, state.Repositories[1].Path}
	assert.Contains(t, paths, "outer")
	assert.Contains(t, paths, "outer/inner")
}

func TestCaptureDetachedHead(t *testing.T) {
	setupGit(t)

	dir := testcli.MkdirTemp(t)
	testcli.Chdir(t, dir)
	testcli.Exec(t, "git init")
	testcli.WriteFile(t, "file1", []byte("content"))
	testcli.Exec(t, "git add .")
	testcli.Exec(t, "git commit -m 'Initial commit'")
	// Detach HEAD
	testcli.Exec(t, "git checkout --detach HEAD")

	args := []string{"gate", "capture"}
	exitCode, stdout, stderr := testcli.Main(t, args, nil, run)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "", stderr)

	var state State
	err := json.Unmarshal([]byte(stdout), &state)
	require.NoError(t, err)

	require.Len(t, state.Repositories, 1)
	assert.Equal(t, "HEAD", state.Repositories[0].Branch)
}
