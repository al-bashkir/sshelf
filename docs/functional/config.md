# Config

Two files in the same directory:

- `config.toml` — application settings and SSH/tmux defaults.
- `hosts.toml` — host overrides, groups, and hidden-hosts list.

Paths:

- Default directory: `$XDG_CONFIG_HOME/sshelf/` or `~/.config/sshelf/`.
- CLI flags: `-config` overrides config.toml path, `-hosts` overrides hosts.toml path.
- When only `-config` is given, hosts.toml is derived from the same directory.

Write rules (both files):

- Atomic write (tmp + rename).
- Final permissions: `0600`.

Migration:

- On first run after upgrade, if `hosts.toml` does not exist but `config.toml` contains `[[hosts]]`, `[[groups]]`, or `hidden_hosts`, those sections are extracted into a new `hosts.toml` and `config.toml` is rewritten without them.
- Migration is idempotent — if `hosts.toml` already exists, nothing happens.
- If migration fails (e.g. read-only directory), a warning is printed and the app continues.

## config.toml

```toml
version = 1

[defaults]
colorscheme = ""         # named theme: default|dracula|nord|gruvbox|catppuccin|kanagawa
                         # when set, overrides accent_color and all UI colors
accent_color = ""        # accent color when no colorscheme is set: default|blue|cyan|green|amber|red|magenta or a color string
load_known_hosts = true  # when false: Hosts list is derived from hosts.toml only
user = ""
port = 22
identity_file = ""
extra_args = []

pane_split = "vertical"  # horizontal|vertical
pane_layout = "even-vertical" # auto|tiled|even-horizontal|even-vertical|main-horizontal|main-vertical
pane_sync = "on"         # on|off

pane_border_format = "..."   # selected tmux format (default always available)
pane_border_formats = []      # user-defined formats list (add/remove via Settings UI)
pane_border_status = "bottom" # off|top|bottom

tmux = "auto"            # auto|force|never
open_mode = "auto"       # auto|current|tmux-window|tmux-pane
tmux_session = "sshelf"
confirm_quit = false
connect_confirm_threshold = 5  # ask for confirmation when connecting to more than N hosts (0 = never ask)
```

## hosts.toml

```toml
version = 1

hidden_hosts = []    # hosts to hide from the Hosts list (compact alternative to [[hosts]] hidden=true)

[[hosts]]
host = "db01.example.com"
user = "admin"
port = 2222
identity_file = "~/.ssh/db01_ed25519"
extra_args = ["-o", "ServerAliveInterval=30"]
hidden = false           # when true, hides this host from the Hosts list

[[groups]]
name = "prod"
user = "admin"
port = 22
identity_file = "~/.ssh/prod_ed25519"
extra_args = ["-o", "ServerAliveInterval=30"]
remote_command = ""  # executed as: sh -c '<remote_command>'

pane_split = ""       # optional override; empty means inherit defaults
pane_layout = ""
pane_sync = ""
pane_border_format = ""
pane_border_status = ""

tmux = ""             # optional override
open_mode = "tmux-window"

hosts = [
  "db01.prod.example.com",
  "[10.10.10.10]:2222",
]
```

Settings merge (for an SSH connection):

1) defaults (from config.toml)
2) host overrides (`[[hosts]]` exact match, from hosts.toml)
3) group overrides (if connecting via group, from hosts.toml); a group setting wins over a host setting

Notes:

- `defaults.pane_border_format` selects one format.
- `defaults.pane_border_formats` stores user-created formats; the built-in default format is always available and can't be deleted.
- Hosts can be hidden via `hidden_hosts = ["host"]` (no `[[hosts]]` entry needed) or by setting `hidden = true` in a `[[hosts]]` block.
- `connect_confirm_threshold`: a confirmation dialog is shown before connecting to more than this many hosts. Default is 5; set to 0 to disable.
- `confirm_quit` defaults to `false`; set to `true` to require `y/n` confirmation before quitting.
