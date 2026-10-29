# Build

Local development:

```bash
go test ./...
go run ./cmd/sshelf
go build -o build/sshelf ./cmd/sshelf
```

Always build to the `build/` directory; never place binaries in the repo root.

Useful flags:

- `--config <path>`
- `--known-hosts <path>` (repeatable)
- `--no-tmux`
- `--debug`
