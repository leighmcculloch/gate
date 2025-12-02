# gate

A CLI tool to capture and restore the state of Git repositories across systems. Discover all Git repositories and worktrees in a directory tree, export their state as JSON, and recreate them on another machine.

## Install

```bash
go install github.com/leighmcculloch/gate@latest
```

Or build from source:

```bash
git clone https://github.com/leighmcculloch/gate
cd gate
go build -o gate .
```

## Usage

### Capture

Scan the current directory (and directories above/below) for Git repositories and output their state as JSON:

```bash
gate capture > state.json
```

### Apply

Read JSON from stdin and clone repositories / set up worktrees:

```bash
gate apply < state.json
```

## Examples

### Backup repository state

```bash
cd ~/Code
gate capture > ~/repos-state.json
```

### Restore on a new machine

```bash
cd ~/Code
gate apply < ~/repos-state.json
```

### Pipe between machines

```bash
ssh old-machine "cd ~/Code && gate capture" | gate apply
```

### Example output

Running `gate capture` in a directory with two repositories:

```json
{
  "repositories": [
    {
      "path": "myproject",
      "remote_url": "git@github.com:user/myproject.git",
      "branch": "main",
      "commit": "abc123def456789..."
    },
    {
      "path": "myproject-feature",
      "branch": "feature-x",
      "commit": "abc123def456789...",
      "is_worktree": true,
      "main_checkout_path": "../myproject"
    }
  ]
}
```

### Working with worktrees

Gate automatically detects Git worktrees and records their relationship to the main checkout. When applying, it ensures main checkouts are cloned before their worktrees are created.

```bash
# Create a project with worktrees
git clone git@github.com:user/project.git
cd project
git worktree add ../project-feature feature-branch
cd ..

# Capture state (includes both main checkout and worktree)
gate capture > state.json

# On another machine, apply will:
# 1. Clone the main repository
# 2. Create the worktree at the correct path and branch
gate apply < state.json
```

## Warnings

Gate warns about uncommitted changes but continues capturing:

```
warning: myproject has uncommitted changes
{
  "repositories": [
    ...
  ]
}
```

Warnings are printed to stderr, JSON output goes to stdout.

When applying, Gate skips directories that already exist:

```
warning: myproject already exists, skipping
```

## JSON Schema

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | Relative path to the repository |
| `remote_url` | string | Origin remote URL (main checkouts only, omitted if empty) |
| `branch` | string | Current branch name, or "HEAD" if detached |
| `commit` | string | Full SHA of the current commit |
| `is_worktree` | bool | True if this is a worktree (omitted for main checkouts) |
| `main_checkout_path` | string | Relative path to main checkout (worktrees only, omitted for main checkouts) |

## Requirements

- Git must be installed and available in PATH
- For `apply`: network access to clone from remote URLs
