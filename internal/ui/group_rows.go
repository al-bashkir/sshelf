package ui

import (
	"slices"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// What the Groups tab and Group hosts share: a group row is a top-level
// group in Groups, and a subgroup in Group hosts.

// groupRows lists the groups directly in the group at path ("" for the top
// level) by name. A count has the hosts a connect opens: those of the group
// and of every group under it.
func groupRows(inv config.Inventory, path string) []groupRow {
	idx := config.Children(inv.Groups, path)
	rows := make([]groupRow, 0, len(idx))
	for _, i := range idx {
		g := config.ResolveGroup(inv.Groups, inv.Groups[i])
		rows = append(rows, groupRow{index: i, name: g.Name, hostCount: len(g.Hosts)})
	}
	return rows
}

// matchGroupRows returns the rows whose name matches query, best first, with
// the matched letters marked. A blank query keeps every row.
func matchGroupRows(rows []groupRow, query string) []groupRow {
	query = strings.TrimSpace(query)
	if query == "" {
		return slices.Clone(rows)
	}
	matches := fuzzy.Find(query, groupNames(rows))
	out := make([]groupRow, 0, len(matches))
	for _, mt := range matches {
		r := rows[mt.Index]
		r.matchedIndexes = mt.MatchedIndexes
		out = append(out, r)
	}
	return out
}

// groupTrail is the breadcrumb of g: "Groups > prod > web".
func groupTrail(g config.Group) string {
	return "Groups > " + strings.ReplaceAll(config.GroupPath(g), "/", " > ")
}

// copyGroupCmd opens a new-group form filled from group i, as its sibling.
func copyGroupCmd(inv config.Inventory, i int, returnTo screen) tea.Cmd {
	g := inv.Groups[i]
	g.Name = suggestCopyGroupName(inv, g.Name)
	return msgCmd(openGroupFormPrefillMsg{group: g, returnTo: returnTo})
}

// connectGroup connects the hosts of group i and of every group under it.
func (c *connectActions) connectGroup(opts Options, i int, oneWindow bool, remoteCmd string) tea.Cmd {
	if i >= len(opts.Inventory.Groups) { // rows index the inventory, so only a stale row fails
		c.toast = toast{text: "invalid group", level: toastErr}
		return nil
	}
	g := config.ResolveGroup(opts.Inventory.Groups, opts.Inventory.Groups[i])
	if len(g.Hosts) == 0 {
		c.toast = toast{text: "group has no hosts", level: toastWarn}
		return nil
	}
	return c.connect(append([]string(nil), g.Hosts...), opts, &g, oneWindow, remoteCmd)
}

// groupDelete is the question asked before a group row is deleted.
type groupDelete struct {
	confirmDelete bool
	deleteIndex   int
}

func (d *groupDelete) askDelete(r groupRow, t *toast) tea.Cmd {
	d.confirmDelete = true
	d.deleteIndex = r.index
	*t = toast{text: "delete? (y/n)", level: toastWarn}
	return nil
}

// deleteKey answers the question: y deletes, n cancels, and any other key
// leaves it open.
func (d *groupDelete) deleteKey(msg tea.KeyMsg, t *toast) tea.Cmd {
	switch msg.String() {
	case "y", "Y", "enter":
		d.confirmDelete = false
		return msgCmd(deleteGroupMsg{index: d.deleteIndex})
	case "n", "N", "esc":
		d.confirmDelete = false
		*t = toast{}
	}
	return nil
}

// deleteBox is the question box, innerW wide, for the group in groups. The
// question is modal, so the groups cannot change while it is open.
func (d *groupDelete) deleteBox(groups []config.Group, innerW int) string {
	g := groups[d.deleteIndex]
	return deleteGroupConfirmBox(innerW, g.Name, len(g.Hosts))
}
