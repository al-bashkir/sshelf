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
