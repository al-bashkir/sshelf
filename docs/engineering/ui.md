# UI Structure

This document describes how the TUI is wired in code (screens, popups, message flow).
For user-facing behavior and keybindings see [UI spec](../functional/ui.md).

## High-level architecture

- Router/state machine: `internal/ui/model_app.go`
- Leaf models (screens, popups): `internal/ui/model_*.go`
- Shared rendering helpers/styles:
  - Layout tabs box: `internal/ui/tab_box.go`
  - Frame + styles + accent: `internal/ui/styles.go`
  - Centered modal placement/sizing: `internal/ui/modal.go`
  - Row rendering for lists: `internal/ui/row_render.go`
  - Help modal: `internal/ui/help_modal.go`
  - Confirm modals (quit, connect, delete/remove): `internal/ui/confirm_modal.go`

The UI uses Bubble Tea's message-driven pattern: every model implements `Init()`, `Update(msg)`, `View()`.

## Screens vs overlays

There are two kinds of "modal" UX:

1) App-level screens (handled by `appModel.screen`)

- These are full Bubble Tea models managed by `model_app.go`.
- They are typically centered modals or separate full-tab content.
- Examples: host picker, group picker, group form, host form, custom host.

2) In-model overlays (handled inside a leaf model)

- The active `appModel.screen` stays the same.
- The leaf model renders an overlay based on local flags.
- Examples:
  - Help popup: `showHelp`
  - Command popup (`Ctrl+o`): `cmdPrompt` + `cmdInput`
  - Confirm flags in list models (`confirmQuit`, `confirmDelete`, `confirmRemove`, `confirmConnect`)

This split is why navigation can feel non-obvious: not every popup changes `appModel.screen`.

## Screen state machine

`internal/ui/model_app.go` defines:

```go
type screen int

const (
  screenHosts screen = iota
  screenGroups
  screenGroupForm
  screenGroupHosts
  screenHostPicker
  screenGroupPicker
  screenDefaultsForm
  screenCustomHost
  screenHostForm
)
```

Navigation graph (simplified):

```text
screenHosts  <-- Alt+1/Alt+2 -->  screenGroups  ---- Enter ---->  screenGroupHosts
    |  Alt+3                          |  Alt+3                     |
    v                             v                             |
screenDefaultsForm           screenDefaultsForm                  |

screenHosts  -- e --> screenHostForm  -- save/cancel --> screenHosts
screenGroupHosts -- e --> screenHostForm -- save/cancel --> screenGroupHosts

screenHosts  -- a --> screenGroupPicker -- done --> screenHosts
screenGroups -- a --> screenHostPicker  -- done --> screenGroups
screenGroupHosts -- a --> screenHostPicker -- done --> screenGroupHosts

screenHosts -- c --> screenCustomHost -- connect/cancel --> screenHosts
screenGroups/screenGroupHosts -- c --> screenCustomHost -- connect/cancel --> returnTo

screenGroups -- n/e/y --> screenGroupForm -- save/cancel --> screenGroups

screenGroupHosts -- Esc --> screenGroups
```

Notes:

- `screenDefaultsForm` is rendered as the Settings tab content (not a centered modal).
- Most other "forms/pickers" are centered via `placeCentered()`.

## Messages and return-to pattern

The router model does not call other models directly; it mostly uses messages:

- Open messages (start a modal/screen):
  - `openGroupFormMsg`, `openGroupFormPrefillMsg`
  - `openHostFormMsg`, `openHostFormPrefillMsg`
  - `openHostPickerMsg`
  - `openGroupPickerMsg`
  - `openDefaultsFormMsg`
  - `openCustomHostMsg`

- Close messages (from modals):
  - `groupFormCancelMsg`, `hostFormCancelMsg`
  - `groupPickerCancelMsg`
  - `hostPickerCancelMsg`
  - `customHostCancelMsg`
  - `defaultsFormCancelMsg`

- Save/done messages (write config + refresh models):
  - `groupFormSaveMsg`
  - `hostFormSaveMsg`
  - `defaultsFormSaveMsg`
  - `deleteGroupMsg`
  - `removeHostsMsg`
  - `hostPickerDoneMsg`
  - `groupPickerDoneMsg`
  - `customHostConnectMsg`, `customHostDoneMsg`, `customHostPickGroupMsg`
  - `toggleHiddenHostMsg`

Return-to is explicit:

- Host form: `openHostFormMsg.returnTo` stored in `appModel.hostFormReturnTo`
- Defaults form: `openDefaultsFormMsg.returnTo` stored in `appModel.defaultsReturnTo`
- Host picker: `openHostPickerMsg.returnTo` stored in `appModel.returnTo`
- Group picker: `gpReturnTo` on appModel

## Window sizing and placement

- Global terminal size is stored in `appModel.width/height`.
- `appModel.applyWindowSize()` forwards size to all active models.

Modal sizing helpers:

- `groupFormModalSize()`, `hostFormModalSize()`, `pickerModalSize()`, `customHostModalSize()` in `internal/ui/modal.go`

Centered placement:

- `placeCentered()` in `internal/ui/modal.go`

The router's `View()` is responsible for centering app-level modal screens.

## Rendering building blocks

- Tabbed main window for list screens: `renderMainTabBox()` in `internal/ui/tab_box.go`
  - Tabs line (Hosts/Groups/Settings)
  - Header line: search (left) + status/toast/selected (right)
  - Content: list view

- Row rendering:
  - Hosts rows + badges (has-config, hidden): `internal/ui/row_render.go`
  - Group rows + count badge: `internal/ui/row_render.go`

- Help modal:
  - `internal/ui/help_modal.go` uses accent-colored key labels.

- Connect confirmation modal:
  - `confirmConnect` flag on list models; shown when connecting to more than `connect_confirm_threshold` hosts.
  - Implemented in `internal/ui/confirm_modal.go` (`connectConfirmBox`).

## Exec and quit flow

- Leaf models may set an `execCmd []string` and request quitting.
- `ui.Run()` returns `*ui.ExecRequest` which `cmd/sshelf/main.go` executes via `syscall.Exec`.
- `ConnectSame` (`O`): always calls `syscall.Exec` with the ssh command (replaces TUI process).

## Adding a new screen/popup (checklist)

1) Define a new `screen` constant in `internal/ui/model_app.go`.
2) Add a new model pointer to `appModel` (if it needs to persist).
3) Add `open...Msg` + (optional) save/cancel message types.
4) Handle the message in `appModel.Update()`:
   - construct the model
   - apply modal sizing (if needed)
   - set `m.screen`
5) Update `appModel.applyWindowSize()` to forward sizes.
6) Update `appModel.View()` to route/center as needed.
7) Wire a keybinding in the relevant leaf model to emit the `open...Msg`.
