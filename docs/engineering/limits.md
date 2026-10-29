# Limits

Explicit limitations (MVP):

- No SSH protocol implementation; calls system `ssh`.
- No full `~/.ssh/config` semantic parsing/merging (system ssh does its normal behavior when we don’t override).
- Hashed known_hosts entries (`|1|...`) are ignored.
- Multi-select interactive connections require tmux modes.
- No secret management; config stores paths and argv tokens only.
