package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// nested is prod > {web > edge, db} and a top-level alpha, stored out of
// tree order.
func nested() []Group {
	return []Group{
		{Name: "web", Parent: "prod", User: "www", Hosts: []string{"w1", "p1"}},
		{Name: "prod", User: "admin", Port: 2222, OpenMode: "tmux-pane", ExtraArgs: []string{"-A"}, Hosts: []string{"p1"}},
		{Name: "edge", Parent: "web", Hosts: []string{"e1"}},
		{Name: "alpha", Hosts: []string{"a1"}},
		{Name: "db", Parent: "prod", User: "postgres", Hosts: []string{"d1"}},
	}
}

func TestGroupTree(t *testing.T) {
	want := []GroupNode{{3, 0}, {1, 0}, {4, 1}, {0, 1}, {2, 2}} // alpha, prod, db, web, edge
	if got := GroupTree(nested()); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := GroupTree(nil); len(got) != 0 {
		t.Fatalf("no groups: got %v", got)
	}
}

func TestParentChoices(t *testing.T) {
	gs := nested()
	for _, c := range []struct {
		name string
		i    int
		want []GroupNode
	}{
		{"a new group can go anywhere", -1, GroupTree(gs)},
		{"prod cannot go under itself or its subgroups", 1, []GroupNode{{3, 0}}},
		{"web can go under prod or db", 0, []GroupNode{{3, 0}, {1, 0}, {4, 1}}},
		{"edge can go anywhere else", 2, []GroupNode{{3, 0}, {1, 0}, {4, 1}, {0, 1}}},
	} {
		if got := ParentChoices(gs, c.i); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// full is a group that sets every setting to a value made from tag.
func full(name, tag string) Group {
	return Group{Name: name, User: "u" + tag, Port: len(tag), IdentityFile: "i" + tag, ExtraArgs: []string{"-" + tag},
		RemoteCommand: "r" + tag, PaneSplit: "s" + tag, PaneLayout: "l" + tag, PaneSync: "y" + tag,
		PaneBorderFmt: "f" + tag, PaneBorderPos: "b" + tag, Tmux: "t" + tag, OpenMode: "o" + tag, Hosts: []string{"h" + tag}}
}

func TestResolveGroupEverySetting(t *testing.T) {
	parent := full("p", "1")
	t.Run("an empty setting inherits", func(t *testing.T) {
		child := Group{Name: "c", Parent: "p", Hosts: []string{"c1"}}
		want := parent
		want.Name, want.Parent, want.Hosts = "c", "p", []string{"c1"}
		if got := ResolveGroup([]Group{parent, child}, child); !reflect.DeepEqual(got, want) {
			t.Fatalf("got %+v\nwant %+v", got, want)
		}
	})
	t.Run("a set setting stays", func(t *testing.T) {
		child := full("c", "22")
		child.Parent = "p"
		if got := ResolveGroup([]Group{parent, child}, child); !reflect.DeepEqual(got, child) {
			t.Fatalf("got %+v\nwant %+v", got, child)
		}
	})
}

func TestResolveGroup(t *testing.T) {
	gs := nested()
	t.Run("a parent gets the hosts of every subgroup, each once", func(t *testing.T) {
		got := ResolveGroup(gs, gs[1])
		if want := []string{"p1", "d1", "w1", "e1"}; !reflect.DeepEqual(got.Hosts, want) {
			t.Fatalf("hosts %v, want %v", got.Hosts, want)
		}
		if got.Name != "prod" || got.User != "admin" {
			t.Fatalf("got %+v, want prod's own settings", got)
		}
	})
	t.Run("a subgroup takes each empty setting from the nearest ancestor", func(t *testing.T) {
		got := ResolveGroup(gs, gs[2]) // edge sets nothing
		if got.User != "www" || got.Port != 2222 || got.OpenMode != "tmux-pane" || !reflect.DeepEqual(got.ExtraArgs, []string{"-A"}) {
			t.Fatalf("got %+v, want user from web, the rest from prod", got)
		}
		if got.Name != "edge" || got.Parent != "web" || !reflect.DeepEqual(got.Hosts, []string{"e1"}) {
			t.Fatalf("got %+v, want edge's own name, parent and hosts", got)
		}
	})
	t.Run("a sibling of an ancestor gives nothing", func(t *testing.T) {
		gs := []Group{{Name: "prod"}, {Name: "a", Parent: "prod", IdentityFile: "a-key"}, {Name: "b", Parent: "prod"}, {Name: "c", Parent: "b"}}
		if got := ResolveGroup(gs, gs[3]); got.IdentityFile != "" {
			t.Fatalf("c took identity %q from a, which is not above it", got.IdentityFile)
		}
	})
	t.Run("resolving twice changes nothing", func(t *testing.T) {
		once := ResolveGroup(gs, gs[1])
		if twice := ResolveGroup(gs, once); !reflect.DeepEqual(twice, once) {
			t.Fatalf("twice %+v, once %+v", twice, once)
		}
	})
	t.Run("a group not in the list comes back as is", func(t *testing.T) {
		g := Group{Name: "other", Hosts: []string{"o1"}}
		if got := ResolveGroup(gs, g); !reflect.DeepEqual(got, g) {
			t.Fatalf("got %+v, want %+v", got, g)
		}
	})
}

func TestHostGroup(t *testing.T) {
	gs := nested()
	prod := ResolveGroup(gs, gs[1])
	for _, c := range []struct {
		host, wantName, wantUser string
	}{
		{"p1", "prod", "admin"},  // prod lists p1 itself, before web does
		{"w1", "web", "www"},     // listed by web
		{"e1", "edge", "www"},    // listed by edge, which inherits web's user
		{"d1", "db", "postgres"}, // listed by db
		{"typed", "prod", "admin"},
	} {
		got := HostGroup(gs, prod, c.host)
		if got.Name != c.wantName || got.User != c.wantUser || got.Port != 2222 {
			t.Errorf("%s: got %s user %q port %d, want %s user %q port 2222", c.host, got.Name, got.User, got.Port, c.wantName, c.wantUser)
		}
	}
}

func TestReparentGroups(t *testing.T) {
	gs := nested()
	got := ReparentGroups(gs, "prod", "")
	for _, i := range []int{0, 4} {
		if got[i].Parent != "" {
			t.Errorf("%s still has parent %q", got[i].Name, got[i].Parent)
		}
	}
	if got[2].Parent != "web" {
		t.Errorf("edge parent %q, want web", got[2].Parent)
	}
	if gs[0].Parent != "prod" {
		t.Error("the input changed")
	}
	if got := ReparentGroups(gs, "", "prod"); !reflect.DeepEqual(got, gs) {
		t.Errorf("from \"\" moved top-level groups: %+v", got)
	}
}

func TestInventoryParentChecks(t *testing.T) {
	for _, c := range []struct {
		name   string
		groups []Group
		want   string // "" means valid
	}{
		{"nested groups", nested(), ""},
		{"missing parent", []Group{{Name: "a", Parent: "nope"}}, `group "a": parent "nope" not found`},
		{"own parent", []Group{{Name: "a", Parent: "a"}}, `group "a": parent "a" makes a loop`},
		{"loop", []Group{{Name: "a", Parent: "b"}, {Name: "b", Parent: "a"}}, `group "a": parent "b" makes a loop`},
		{"bad name", []Group{{Name: "a b"}}, `group "a b": group name "a b" is invalid`},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "hosts.toml")
			inv := DefaultInventory()
			inv.Groups = c.groups
			_, err := SaveInventory(p, &inv)
			if !errContains(err, c.want) {
				t.Fatalf("save: err %v, want %q", err, c.want)
			}
			if c.want == "" {
				if got, _, err := LoadInventory(p); err != nil || !reflect.DeepEqual(got.Groups, c.groups) {
					t.Fatalf("load: err %v, groups %+v", err, got.Groups)
				}
				return
			}
			if _, err := os.Stat(p); !os.IsNotExist(err) {
				t.Fatal("save wrote a bad inventory")
			}
			var b strings.Builder
			for _, g := range c.groups {
				b.WriteString("[[groups]]\nname = \"" + g.Name + "\"\nparent = \"" + g.Parent + "\"\n")
			}
			if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := LoadInventory(p); !errContains(err, c.want) {
				t.Fatalf("load: err %v, want %q", err, c.want)
			}
		})
	}
}

// errContains reports whether err holds want; want "" means no error.
func errContains(err error, want string) bool {
	if want == "" {
		return err == nil
	}
	return err != nil && strings.Contains(err.Error(), want)
}
