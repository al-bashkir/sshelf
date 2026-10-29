# SSH

The app calls the system `ssh` binary.

Target:

- If user is set: `user@host`
- Else: `host`
- If host is `[h]:p`:
  - force port `-p p`
  - target becomes `h` (no brackets)

Args:

- `-i <identity_file>` if set
- `-p <port>` when port != 22
- `extra_args` appended as-is
- `--` before the target, so a host that starts with `-` is never read as an ssh option

Remote command:

- Group `remote_command` and `Ctrl+o` use remote execution.
- The remote command is executed as: `sh -c '<command>'`.
- For `Ctrl+o` we also add: `; exec ${SHELL:-sh}` to keep the session open.
- `-t` (force TTY) is automatically added when a remote command is set via `Ctrl+o`.

Execution modes:

- `open_mode=current`: replace the TUI process with `ssh` (`syscall.Exec`).
- `O` (ConnectSame): always replaces the TUI process with `ssh`, regardless of `open_mode`.
- tmux modes: create panes/windows and keep TUI alive.
