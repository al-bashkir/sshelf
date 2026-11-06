package config

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// GroupNode is one group in tree order.
type GroupNode struct {
	Index int // into the group list
	Depth int // 0 for a top-level group
}

// GroupTree lists the groups in tree order: each group is followed by its
// subgroups, and siblings sort by name, ignoring case. A group whose parents
// loop is left out; load and save refuse such an inventory.
func GroupTree(groups []Group) []GroupNode {
	kids := map[int][]int{}
	for i, g := range groups {
		p := -1
		if g.Parent != "" {
			p = groupIndex(groups, g.Parent)
		}
		kids[p] = append(kids[p], i)
	}
	var out []GroupNode
	var walk func(parent, depth int)
	walk = func(parent, depth int) {
		k := kids[parent]
		sort.SliceStable(k, func(a, b int) bool {
			return strings.ToLower(groups[k[a]].Name) < strings.ToLower(groups[k[b]].Name)
		})
		for _, i := range k {
			out = append(out, GroupNode{Index: i, Depth: depth})
			walk(i, depth+1)
		}
	}
	walk(-1, 0)
	return out
}

// ParentChoices lists, in tree order, the groups that group i can sit in:
// every group but i and the groups under it. i < 0 is a new group.
func ParentChoices(groups []Group, i int) []GroupNode {
	tree := GroupTree(groups)
	at := treeAt(tree, i)
	if at < 0 {
		return tree
	}
	under := below(tree, at)
	return slices.DeleteFunc(tree, func(n GroupNode) bool { return slices.Contains(under, n.Index) })
}

// ResolveGroup returns g as a connection uses it. Each setting g leaves
// empty comes from its nearest ancestor that sets it, and Hosts holds the
// hosts of g and of every group under it, each once, in tree order. A g that
// is not in groups comes back as is.
func ResolveGroup(groups []Group, g Group) Group {
	tree := GroupTree(groups)
	at := treeAt(tree, groupIndex(groups, g.Name))
	if at < 0 {
		return g
	}
	for _, a := range ancestors(tree, at) {
		g = inherit(g, groups[a])
	}
	hosts := slices.Clone(g.Hosts) // never append into the stored group's array
	for _, j := range below(tree, at) {
		for _, h := range groups[j].Hosts {
			if !slices.Contains(hosts, h) {
				hosts = append(hosts, h)
			}
		}
	}
	g.Hosts = hosts
	return g
}

// HostGroup returns the group whose settings host uses when it opens
// through g: the first group in tree order under g, g first, that lists
// host, resolved. A host no group under g lists uses g.
func HostGroup(groups []Group, g Group, host string) Group {
	tree := GroupTree(groups)
	if at := treeAt(tree, groupIndex(groups, g.Name)); at >= 0 {
		for _, j := range below(tree, at) {
			if slices.Contains(groups[j].Hosts, host) {
				return ResolveGroup(groups, groups[j])
			}
		}
	}
	return ResolveGroup(groups, g)
}

// ReparentGroups returns a copy of groups with the subgroups of the group
// named from moved to to. No group is named "", so from "" moves none.
func ReparentGroups(groups []Group, from, to string) []Group {
	out := slices.Clone(groups)
	for i := range out {
		if from != "" && out[i].Parent == from {
			out[i].Parent = to
		}
	}
	return out
}

// checkGroups checks every group name, and that every parent names a group
// and does not lead back to the group.
func checkGroups(groups []Group) error {
	inTree := make([]bool, len(groups))
	for _, n := range GroupTree(groups) {
		inTree[n.Index] = true
	}
	for i, g := range groups {
		if err := ValidateGroupName(g.Name); err != nil {
			return fmt.Errorf("group %q: %w", g.Name, err)
		}
		switch {
		case g.Parent != "" && groupIndex(groups, g.Parent) < 0:
			return fmt.Errorf("group %q: parent %q not found", g.Name, g.Parent)
		case !inTree[i]:
			return fmt.Errorf("group %q: parent %q makes a loop", g.Name, g.Parent)
		}
	}
	return nil
}

// inherit fills each setting g leaves empty from parent.
func inherit(g, parent Group) Group {
	for own, from := range map[*string]string{
		&g.User: parent.User, &g.IdentityFile: parent.IdentityFile, &g.RemoteCommand: parent.RemoteCommand,
		&g.PaneSplit: parent.PaneSplit, &g.PaneLayout: parent.PaneLayout, &g.PaneSync: parent.PaneSync,
		&g.PaneBorderFmt: parent.PaneBorderFmt, &g.PaneBorderPos: parent.PaneBorderPos,
		&g.Tmux: parent.Tmux, &g.OpenMode: parent.OpenMode,
	} {
		if strings.TrimSpace(*own) == "" {
			*own = from
		}
	}
	if g.Port == 0 {
		g.Port = parent.Port
	}
	if len(g.ExtraArgs) == 0 {
		g.ExtraArgs = parent.ExtraArgs
	}
	return g
}

// groupIndex returns the index of the group named name, or -1.
func groupIndex(groups []Group, name string) int {
	return slices.IndexFunc(groups, func(g Group) bool { return g.Name == name })
}

// treeAt returns the position of group i in tree, or -1.
func treeAt(tree []GroupNode, i int) int {
	return slices.IndexFunc(tree, func(n GroupNode) bool { return n.Index == i })
}

// ancestors returns the groups above the node at position at, nearest first.
func ancestors(tree []GroupNode, at int) []int {
	var out []int
	depth := tree[at].Depth
	for j := at - 1; j >= 0 && depth > 0; j-- {
		if tree[j].Depth < depth {
			out = append(out, tree[j].Index)
			depth = tree[j].Depth
		}
	}
	return out
}

// below returns the group at position at and every group under it.
func below(tree []GroupNode, at int) []int {
	out := []int{tree[at].Index}
	for _, n := range tree[at+1:] {
		if n.Depth <= tree[at].Depth {
			break
		}
		out = append(out, n.Index)
	}
	return out
}
