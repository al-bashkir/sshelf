# tmux

Detection:

- In tmux when `$TMUX` is set.

Mode settings:

- `defaults.tmux = auto|force|never`
- `defaults.open_mode = auto|current|tmux-window|tmux-pane`

Behaviors:

- Inside tmux: can open a new window or split panes.
- Outside tmux: opens the host in session `tmux_session` and attaches to it. If the session does not exist, it is created; if it exists, the host opens as a new window in it.

Multi-select:

- `open_mode=current` cannot open multiple interactive sessions; multi-select requires tmux.

One window with panes:

- Opens a single tmux window and splits panes for each host.
- Applies layout/sync and pane border settings from defaults/group.

Reconnect on lost connection (all tmux modes):

- tmux runs each ssh through the hidden wrapper `sshelf __pane HOST -- SSH_ARGV...`.
- ssh exits 255 (connection lost or failed): the pane stays open with
  `Connection to HOST lost: [r] reconnect  [Enter] close`.
  - `r`/`R`: run the same ssh again in the same pane.
  - `Enter`, `q`, Ctrl-C, Ctrl-D: close the pane.
  - other keys are ignored.
- Any other exit (user ended the session): the pane closes at once.
- While the prompt shows, `synchronize-panes` is off for that pane only (tmux >= 3.2),
  so synced keys do not cross between the dead pane and live panes.
- `open_mode=current` without tmux execs ssh directly, with no wrapper.
