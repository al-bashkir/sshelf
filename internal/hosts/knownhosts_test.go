package hosts

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseKnownHosts(t *testing.T) {
	in := strings.Join([]string{
		"# comment",
		"",
		"@cert-authority example.com ssh-ed25519 AAAA...",
		"|1|hashedhost ssh-ed25519 AAAA...",
		"example.com,10.0.0.1 ssh-ed25519 AAAA...",
		"[10.10.10.10]:2222 ssh-rsa AAAA...",
		"example.com ssh-ed25519 AAAA...",
	}, "\n")

	hosts, skipped, err := ParseKnownHosts(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseKnownHosts error: %v", err)
	}
	if skipped != 4 {
		t.Fatalf("skipped=%d, want %d", skipped, 4)
	}

	want := []string{"10.0.0.1", "[10.10.10.10]:2222", "example.com"}
	if !reflect.DeepEqual(hosts, want) {
		t.Fatalf("hosts=%v, want %v", hosts, want)
	}
}

func TestParseKnownHostsKeepsOneFieldLine(t *testing.T) {
	hosts, skipped, err := ParseKnownHosts(strings.NewReader("lone.example\n"))
	if err != nil || skipped != 0 || !reflect.DeepEqual(hosts, []string{"lone.example"}) {
		t.Fatalf("got hosts=%v skipped=%d err=%v, want [lone.example] 0 nil", hosts, skipped, err)
	}
}

func TestParseKnownHostsLineOver64KiB(t *testing.T) {
	line := "long.example ssh-rsa " + strings.Repeat("A", 100*1024)
	hosts, _, err := ParseKnownHosts(strings.NewReader(line + "\n"))
	if err != nil || !reflect.DeepEqual(hosts, []string{"long.example"}) {
		t.Fatalf("got hosts=%v err=%v, want [long.example] nil", hosts, err)
	}
}

func TestParseKnownHostsDropsEmptyAndHashedNamesInList(t *testing.T) {
	hosts, skipped, err := ParseKnownHosts(strings.NewReader("a.example,,|1|abc,b.example ssh-ed25519 AAAA...\n"))
	if err != nil || skipped != 0 || !reflect.DeepEqual(hosts, []string{"a.example", "b.example"}) {
		t.Fatalf("got hosts=%v skipped=%d err=%v, want [a.example b.example] 0 nil", hosts, skipped, err)
	}
}

func TestParseKnownHostsLineOver1MiBFails(t *testing.T) {
	line := "huge.example ssh-rsa " + strings.Repeat("A", 1024*1024)
	if hosts, _, err := ParseKnownHosts(strings.NewReader(line + "\n")); err == nil {
		t.Fatalf("got hosts=%v err=nil, want an error", hosts)
	}
}

func TestPathErrorMessage(t *testing.T) {
	err := os.ErrNotExist
	if got := (PathError{Path: "/x", Err: err}).Error(); got != "/x: "+err.Error() {
		t.Fatalf("with path: got %q", got)
	}
	if got := (PathError{Err: err}).Error(); got != err.Error() {
		t.Fatalf("without path: got %q", got)
	}
}

func TestLoadKnownHostsDefaultsToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "known_hosts"), []byte("home.example ssh-ed25519 AAAA...\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	res, errs := LoadKnownHosts(nil)
	if len(errs) != 0 || !reflect.DeepEqual(res.Hosts, []string{"home.example"}) {
		t.Fatalf("got hosts=%v errs=%v, want [home.example] none", res.Hosts, errs)
	}
}

func TestLoadKnownHosts(t *testing.T) {
	d := t.TempDir()
	f1 := filepath.Join(d, "known_hosts_1")
	f2 := filepath.Join(d, "known_hosts_2")

	if err := os.WriteFile(f1, []byte("a.example ssh-ed25519 AAAA...\n"), 0o600); err != nil {
		t.Fatalf("write f1: %v", err)
	}
	if err := os.WriteFile(f2, []byte("b.example ssh-ed25519 AAAA...\na.example ssh-ed25519 AAAA...\n"), 0o600); err != nil {
		t.Fatalf("write f2: %v", err)
	}

	res, errs := LoadKnownHosts([]string{f1, filepath.Join(d, "missing"), f2})
	if len(errs) != 1 {
		t.Fatalf("errs=%v, want 1 error", errs)
	}

	want := []string{"a.example", "b.example"}
	if !reflect.DeepEqual(res.Hosts, want) {
		t.Fatalf("hosts=%v, want %v", res.Hosts, want)
	}
}
