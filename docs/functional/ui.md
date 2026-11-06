# UI

Code-level UI structure: [UI Structure](../engineering/ui.md)

Screens:

- Hosts: list of hosts + fuzzy search + multi-select.
- Groups: list of groups + CRUD.
- Group Hosts: hosts inside a group.
- Settings: defaults editor. The colorscheme and accent pickers preview the theme under the cursor; `Esc`, or leaving Settings without a save, shows the saved theme again.

Rendering rules:

- Main list screens render inside a framed tabbed layout.
- Popups (custom host, command connect, pickers, forms) are centered modals and must not replace the whole screen.

Keybindings (high level):

- Global: `Ctrl+f` focus search, `Tab` toggle search/list focus, `gg` go to current page top, `G` go to current page bottom, `Esc` clear/blur/back, `?` help, `q` quit (confirm configurable).
- Tabs: `Alt+1` opens Hosts, `Alt+2` opens Groups, `Alt+3` opens Settings.

Hosts:

- `Enter` connect (current or selected).
- `O` connect in current window/pane (replaces TUI process with ssh).
- `Space` toggle selection.
- `Ctrl+a` select all, `Ctrl+d` clear selection.
- `Ctrl+o` connect with custom command (inline popup).
- `c` connect to custom host (popup); in the popup `Ctrl+g` adds the typed hosts to a group.
- `a` add selected hosts to group (group picker).
- `e` edit host config (popup).
- `y` copy host config (only if a `[[hosts]]` override exists).
- `o` open in one tmux window with panes.
- `r` reload known_hosts (disabled when `defaults.load_known_hosts=false`).
- `Ctrl+h` hide/unhide current host.
- `H` toggle display of hidden hosts.

Groups:

- Groups show as a tree: a subgroup sits under its parent, two cells further in. A search lists the matches flat.
- The host count of a group includes the hosts of its subgroups.
- `n` new group (popup form). The `Parent` picker puts the group inside another group.
- `e` edit group. The `Parent` picker does not offer the group itself or its subgroups.
- `d` delete group (confirm overlay).
- `y` copy group.
- `Enter` open group hosts.
- `C` connect all.
- `Ctrl+o` connect all with custom command.
- `o` open all in one tmux window with panes.
- `a` add hosts (picker).
- `c` custom host + connect.

Group Hosts:

- Same multi-select/connect keys as Hosts (Enter, O, Space, Ctrl+a, Ctrl+d, Ctrl+o, o).
- `a` add hosts (picker).
- `c` custom host + connect.
- `d` remove host(s) from group (confirm).
- `e` edit host config.
- `y` copy host config.
- `Esc` go back to Groups.

Connect confirmation:

- When connecting to more than `connect_confirm_threshold` hosts at once, a confirmation dialog is shown listing the hosts.
- Default threshold is 5. Set to 0 to disable.
