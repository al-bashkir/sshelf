# AGENTS.md - sshelf

## Commands

- Baseline verification: `go test ./...`
- Focused package test: `go test ./internal/sshcmd`
- Focused test: `go test ./internal/tmux -run TestResolveOpenMode`
- Run locally: `go run ./cmd/sshelf`
- Build locally: `go build -o build/sshelf ./cmd/sshelf`
- Always build into `build/`; never place binaries in the repo root.
- Release CI runs `go test ./...` then `gosec ./...`.

## Structure

- CLI entrypoint and flag parsing: `cmd/sshelf/main.go`
- Subcommands: `cmd/sshelf/cmd_connect.go`, `cmd/sshelf/cmd_list.go`, `cmd/sshelf/cmd_completion.go`
- Config and inventory TOML load/save/migration: `internal/config`
- `known_hosts` parsing/loading: `internal/hosts`
- SSH argv construction: `internal/sshcmd`
- tmux detection and argv builders: `internal/tmux`
- Bubble Tea app, screens, styles, keybindings: `internal/ui`
- UI router/state machine: `internal/ui/model_app.go`; `internal/ui/run.go` returns `ExecRequest` for `syscall.Exec`.

## Runtime Gotchas

- The app delegates connections to the system `ssh`; it does not implement SSH.
- Config is split into `config.toml` and `hosts.toml`; first run can migrate older single-file config.
- Default config dir respects `$XDG_CONFIG_HOME`, otherwise `~/.config/sshelf`.
- Config and inventory saves are atomic and chmod `0600`.
- Hashed `known_hosts` entries are skipped because they are not displayable.
- Multi-host interactive sessions require tmux modes; single-host/current-pane connections exec `ssh` directly.
- Global flags must appear before subcommands.

## Docs

- Start with `docs/README.md` for specs and engineering notes.
- Use `docs/engineering/ui.md` before changing screens, popups, routing, or modal behavior.
- Use `docs/functional/config.md`, `docs/functional/ssh.md`, and `docs/functional/tmux.md` before changing config merge, SSH argv, or tmux behavior.

## Packaging

- COPR SRPM flow is `make -f .copr/Makefile srpm`; it vendors Go deps into the source tarball.
- RPM builds use `go build -trimpath -mod=vendor -ldflags "-s -w" ./cmd/sshelf`.

## Git Workflow

- After file changes, stop and report what changed; committing is a separate explicit step.
- Only run `git commit` when the user explicitly says `commit`.
- Never commit directly to `main`.
- Never run `git push`; leave pushing to the user.
