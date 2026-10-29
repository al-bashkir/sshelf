# Product

`sshelf` is a terminal UI (TUI) and CLI for managing SSH connections to hosts and host groups.

Core ideas:

- Hosts are sourced primarily from `known_hosts`.
- Groups and overrides are stored in an app config file.
- The app does not implement SSH; it builds argv and calls the system `ssh`.
- Optional tmux integration: open connections as panes/windows and keep the UI running.

MVP goals:

- Hosts list + fuzzy search.
- Connect to one host.
- Multi-select + connect to multiple hosts (tmux-driven).
- Groups CRUD + group-level overrides.
- Atomic config save (0600).
- CLI subcommands for scripting and shell integration.
- Shell completion for bash and zsh.

CLI subcommands (non-interactive):

- `sshelf connect host NAME` — connect to a host by name.
- `sshelf connect group NAME` — connect to all hosts in a group.
- `sshelf list hosts [--json]` — print known hosts.
- `sshelf list groups [--json]` — print configured groups.
- `sshelf completion bash|zsh` — print shell completion script.

Non-goals (MVP):

- Implementing the SSH protocol (no `x/crypto/ssh`).
- Full SSH config parsing/merging.
- Managing private keys/secrets (only file paths and ssh options).
