package main

// Repository represents a single git repository or worktree
type Repository struct {
	Path             string  `json:"path"`
	RemoteURL        string  `json:"remote_url,omitempty"`
	Branch           string  `json:"branch"`
	Commit           string  `json:"commit"`
	IsWorktree       bool    `json:"is_worktree"`
	MainCheckoutPath *string `json:"main_checkout_path"`
}

// State represents the complete state of all repositories
type State struct {
	Repositories []Repository `json:"repositories"`
}
