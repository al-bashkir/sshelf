# Structure

Entry point:

- `cmd/sshelf/main.go`

CLI subcommand files:

- `cmd/sshelf/cmd_connect.go`: `connect host|group` subcommand
- `cmd/sshelf/cmd_list.go`: `list hosts|groups` subcommand
- `cmd/sshelf/cmd_completion.go`: `completion bash|zsh` subcommand + internal `__complete` helper

Packages:

- `internal/config`: config + inventory schema, load/save (atomic, 0600), migration
- `internal/hosts`: known_hosts parsing/loading
- `internal/sshcmd`: build `ssh` argv from merged settings
- `internal/tmux`: build `tmux` argv, detect tmux, pane helpers
- `internal/ui`: Bubble Tea models/views, styling, keybindings

UI routing:

- `internal/ui/model_app.go` is the state machine switching between screens and popups.

UI structure (deeper):

- [UI Structure](ui.md)

UI files map (non-exhaustive — covers main models, shared helpers, and key utilities):

- `internal/ui/run.go`: `Run()` entrypoint, `Options`, `ExecRequest`, `ErrQuit`
- `internal/ui/model_app.go`: app router/state machine, config save/refresh
- `internal/ui/model_hosts.go`: Hosts list screen model
- `internal/ui/model_groups.go`: Groups list screen model
- `internal/ui/model_group_hosts.go`: Group Hosts list screen model
- `internal/ui/model_defaults_form.go`: Settings (defaults) editor
- `internal/ui/model_group_form.go`: Group create/edit form
- `internal/ui/model_host_form.go`: Host config create/edit form
- `internal/ui/model_host_picker.go`: picker to select hosts
- `internal/ui/model_group_picker.go`: picker to select a group
- `internal/ui/model_custom_host.go`: custom host connect popup
- `internal/ui/model_pane_border_formats.go`: pane border format picker/editor
- `internal/ui/tab_box.go`: tabbed main layout renderer
- `internal/ui/styles.go`: styles + accent color
- `internal/ui/modal.go`: modal sizing + centering
- `internal/ui/keymap.go`: shared key bindings
- `internal/ui/row_render.go`: host/group row rendering with badges
- `internal/ui/help_modal.go`: help overlay (accent-colored key labels)
- `internal/ui/helpmap.go`: `helpMap` type used by help modal
- `internal/ui/confirm_modal.go`: quit/connect/delete confirm dialogs
- `internal/ui/dispatch_tmux.go`: `dispatchConnect`, `buildSSHCommands`, `resolveConnectMode`
- `internal/ui/ssh_helpers.go`: `ensureSSHForceTTY`, `keepSessionOpenRemoteCmd`
- `internal/ui/host_config.go`: `hostConfigFor`, `findHostConfig`, `isHostHidden`
- `internal/ui/host_select.go`: shared `hostSelectList` struct (filter, selection, badges)
- `internal/ui/update_helpers.go`: shared Update() helpers (`handleConfirmQuit`, `handleConfirmConnect`, `updateSearchOrList`)
- `internal/ui/copy_helpers.go`: `suggestCopyHostKey`, `suggestCopyGroupName`
- `internal/ui/connect_group.go`: `connectHosts` (unified group/defaults connect)
- `internal/ui/pane_border_formats.go`: `paneBorderFormatChoices`, add/remove helpers
- `internal/ui/listutil.go`: shared list configuration (`configureList`, `newSearchInput`)
- `internal/ui/input_underline.go`: `configureSearch`, `setSearchBarFocused`, `setSearchFocused`
- `internal/ui/view_hosts.go`: `renderMainTabBox` and related view helpers
- `internal/ui/view_shared.go`: `renderCmdPromptModal` and other shared overlays
- `internal/ui/layout.go`: layout constants (header lines, width thresholds, modal margins)
- `internal/ui/toast.go`: `toast` type, levels, `renderToast`, `toastMsg`
- `internal/ui/form_schema.go`: `fieldKind`, `fieldDef`, form field descriptors
- `internal/ui/form_model.go`: generic form model shared by defaults/host/group editors
- `internal/ui/form_helpers.go`: `formLabel` and other form rendering utilities
- `internal/ui/form_render.go`: field-level rendering (text, picker, toggle rows)
- `internal/ui/form_validate.go`: field validators (`validatePort`, etc.)
- `internal/ui/option_picker_popup.go`: inline option-picker popup used by `formModel`
