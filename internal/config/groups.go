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

// GroupPath is the path of g: the path of its parent and its name, joined by
// "/". A group is known by its path; one name can be in two parents.
func GroupPath(g Group) string {
	if g.Parent == "" {
		return g.Name
	}
	return g.Parent + "/" + g.Name
}

// FindGroup returns the index of the group at path, or -1.
func FindGroup(groups []Group, path string) int {
	return slices.IndexFunc(groups, func(g Group) bool { return GroupPath(g) == path })
}

// Children returns the groups directly in the group at path ("" for the top
// level), sorted by name, ignoring case.
func Children(groups []Group, path string) []int {
	var out []int
	for i, g := range groups {
		if g.Parent == path {
			out = append(out, i)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		return strings.ToLower(groups[out[a]].Name) < strings.ToLower(groups[out[b]].Name)
	})
	return out
}

// GroupTree lists the groups in tree order: each group is followed by its
// subgroups. A parent path is always shorter than the path of its group, so
// parents cannot loop.
func GroupTree(groups []Group) []GroupNode {
	var out []GroupNode
	var walk func(path string, depth int)
	walk = func(path string, depth int) {
		for _, i := range Children(groups, path) {
			out = append(out, GroupNode{Index: i, Depth: depth})
			if p := GroupPath(groups[i]); p != path { // an unnamed group would walk into itself
				walk(p, depth+1)
			}
		}
	}
	walk("", 0)
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
	at := treeAt(tree, FindGroup(groups, GroupPath(g)))
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
	if at := treeAt(tree, FindGroup(groups, GroupPath(g))); at >= 0 {
		for _, j := range below(tree, at) {
			if slices.Contains(groups[j].Hosts, host) {
				return ResolveGroup(groups, groups[j])
			}
		}
	}
	return ResolveGroup(groups, g)
}

// ReparentGroups returns a copy of groups in which every group under the
// group at path from is under the path to instead: a rename or a move of
// that group. From "" moves none.
func ReparentGroups(groups []Group, from, to string) []Group {
	out := slices.Clone(groups)
	for i := range out {
		if from != "" && under(out[i], from) {
			out[i].Parent = to + strings.TrimPrefix(out[i].Parent, from)
		}
	}
	return out
}

// DeleteGroup returns a copy of groups without group i and every group
// under it.
func DeleteGroup(groups []Group, i int) []Group {
	path := GroupPath(groups[i])
	return slices.DeleteFunc(slices.Clone(groups), func(g Group) bool { return GroupPath(g) == path || under(g, path) })
}

// under reports whether g sits in the group at path, or deeper.
func under(g Group, path string) bool {
	return strings.HasPrefix(g.Parent+"/", path+"/")
}

// checkGroups checks every group name, that every parent path names a group,
// and that no path is used twice.
func checkGroups(groups []Group) error {
	seen := make(map[string]bool, len(groups))
	for _, g := range groups {
		path := GroupPath(g)
		switch err := ValidateGroupName(g.Name); {
		case err != nil:
			return fmt.Errorf("group %q: %w", path, err)
		case seen[path]:
			return fmt.Errorf("group %q is defined twice", path)
		case g.Parent != "" && FindGroup(groups, g.Parent) < 0:
			return fmt.Errorf("group %q: parent %q not found", path, g.Parent)
		}
		seen[path] = true
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

// treeAt returns the position of group i in tree, or -1.
func treeAt(tree []GroupNode, i int) int {
	return slices.IndexFunc(tree, func(n GroupNode) bool { return n.Index == i })
}

// ancestors returns the groups above the node at position at, nearest first.
func ancestors(tree []GroupNode, at int) []int {
	var out []int
	depth := tree[at].Depth
	for j := at - 1; j >= 0; j-- {
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
