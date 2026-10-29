package ui

import (
	"fmt"
	"slices"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

func filterOpts(n int) Options {
	inv := config.Inventory{}
	var hs []string
	for i := range n {
		h := fmt.Sprintf("host%04d.example.com", i)
		hs = append(hs, h)
		inv.Hosts = append(inv.Hosts, config.Host{Host: h, Hidden: i%10 == 0})
		if i%7 == 0 {
			inv.HiddenHosts = append(inv.HiddenHosts, h)
		}
	}
	return Options{Config: config.DefaultConfig(), Inventory: inv, Hosts: hs}
}

func rowFor(m *hostsModel, host string) (hostRow, bool) {
	for _, it := range m.list.Items() {
		if r := it.(hostRow); r.host == host {
			return r, true
		}
	}
	return hostRow{}, false
}

func TestHostsFilterRowsBadgesAndHiddenCount(t *testing.T) {
	inv := config.Inventory{
		Hosts: []config.Host{
			{Host: "alpha", Hidden: true},
			{Host: "beta"},
			{Host: "dup"},
			{Host: "dup", Hidden: true}, // first match wins
		},
		HiddenHosts: []string{" gamma "},
	}
	hs := []string{"alpha", "beta", "gamma", "[beta]:2222", "[alpha]:22", "dup", "plain"}
	m := newHostsModel(Options{Config: config.DefaultConfig(), Inventory: inv, Hosts: hs})

	want := []string{"beta", "[beta]:2222", "dup", "plain"}
	if got := shownHosts(m); !slices.Equal(got, want) {
		t.Fatalf("shown %v, want %v", got, want)
	}
	if got := m.hiddenCount(); got != 3 {
		t.Fatalf("hiddenCount=%d, want 3", got)
	}
	for host, cfg := range map[string]bool{"beta": true, "[beta]:2222": true, "dup": true, "plain": false} {
		if r, _ := rowFor(m, host); r.hasCfg != cfg {
			t.Errorf("%s hasCfg=%v, want %v", host, r.hasCfg, cfg)
		}
	}

	m.showHidden = true
	m.applyFilter("")
	if got := shownHosts(m); len(got) != len(hs) {
		t.Fatalf("showHidden shows %v", got)
	}
	for _, h := range []string{"alpha", "gamma", "[alpha]:22"} {
		if r, _ := rowFor(m, h); !r.hidden {
			t.Errorf("%s not flagged hidden", h)
		}
	}
	if r, _ := rowFor(m, "beta"); r.hidden {
		t.Error("beta flagged hidden")
	}
	if got := m.hiddenCount(); got != 3 {
		t.Fatalf("hiddenCount=%d with showHidden, want 3", got)
	}
}

func BenchmarkHostsFilter(b *testing.B) {
	m := newHostsModel(filterOpts(3000))
	queries := []string{"h", "ho", "hos", "host1", "host12", ""}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.applyFilter(queries[i%len(queries)])
		_ = m.hiddenCount()
	}
}
