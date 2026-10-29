package sshcmd

import (
	"reflect"
	"testing"

	"github.com/al-bashkir/sshelf/internal/config"
)

func TestApplyGroupRemoteCommand(t *testing.T) {
	base := Settings{RemoteCommand: "keep"}
	if got := ApplyGroup(base, config.Group{RemoteCommand: "uptime"}); got.RemoteCommand != "uptime" {
		t.Errorf("set: got %q want uptime", got.RemoteCommand)
	}
	for _, blank := range []string{"", "  \t"} {
		if got := ApplyGroup(base, config.Group{RemoteCommand: blank}); got.RemoteCommand != "keep" {
			t.Errorf("blank %q: got %q want keep", blank, got.RemoteCommand)
		}
	}
}

func TestBuildCommandBracketHostEdgeCases(t *testing.T) {
	s := Settings{Port: 2200}
	tests := []struct {
		host string
		want []string
	}{
		{"[h]:22", []string{"ssh", "--", "h"}},
		{"[h]:2222", []string{"ssh", "-p", "2222", "--", "h"}},
		{"[h]:1", []string{"ssh", "-p", "1", "--", "h"}},
		{"[]:22", []string{"ssh", "-p", "2200", "--", "[]:22"}},
		{"[h]:0", []string{"ssh", "-p", "2200", "--", "[h]:0"}},
		{"[h]:-1", []string{"ssh", "-p", "2200", "--", "[h]:-1"}},
		{"[h]:x", []string{"ssh", "-p", "2200", "--", "[h]:x"}},
		{"[h]:", []string{"ssh", "-p", "2200", "--", "[h]:"}},
		{"[h]", []string{"ssh", "-p", "2200", "--", "[h]"}},
		{"h", []string{"ssh", "-p", "2200", "--", "h"}},
	}
	for _, tc := range tests {
		if got := BuildCommand(tc.host, s); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: got %v want %v", tc.host, got, tc.want)
		}
	}
}

func TestBracketHostBase(t *testing.T) {
	for host, want := range map[string]string{"[h]:22": "h", "[h]:1": "h"} {
		if got, ok := BracketHostBase(host); !ok || got != want {
			t.Errorf("%q: got %q,%v", host, got, ok)
		}
	}
	for _, host := range []string{"[]:22", "[h]:0", "[h]:-1", "[h]:x", "[h]", "h"} {
		if got, ok := BracketHostBase(host); ok || got != "" {
			t.Errorf("%q: got %q,%v want none", host, got, ok)
		}
	}
}

func TestBuildCommandSingleExtraArg(t *testing.T) {
	got := BuildCommand("h", Settings{ExtraArgs: []string{"-A"}})
	want := []string{"ssh", "-A", "--", "h"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v want %v", got, want)
	}
}
