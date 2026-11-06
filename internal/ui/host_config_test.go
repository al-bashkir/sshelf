package ui

import (
	"reflect"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
	"github.com/al-bashkir/sshelf/internal/sshcmd"
)

func TestHostIndexLookupMatchesFindHostConfig(t *testing.T) {
	inv := config.Inventory{Hosts: []config.Host{{Host: " "}, {Host: "a", User: "u"}}}
	x := newHostIndex(inv)
	for _, h := range []string{"", "  ", "a", " a ", "[a]:22", "b"} {
		want, wantOK := sshcmd.FindHostConfig(inv.Hosts, h)
		if got, ok := x.lookup(h); ok != wantOK || !reflect.DeepEqual(got, want) {
			t.Errorf("lookup(%q) = %+v, %v; FindHostConfig gives %+v, %v", h, got, ok, want, wantOK)
		}
	}
}
