package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/al-bashkir/sshelf/internal/ui"
)

// proc fakes the process boundary and records what run asked of it.
type proc struct {
	tuiOpts *ui.Options
	tuiErr  error
	execed  [][]string
	execErr error
}

// sshelf runs one command line; "$D" in args is the fixture dir.
func sshelf(t *testing.T, p *proc, dir, cmdline string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	c := cli{
		stdout: &out,
		stderr: &errb,
		tui:    func(o ui.Options) error { p.tuiOpts = &o; return p.tuiErr },
		exec:   func(argv []string) error { p.execed = append(p.execed, argv); return p.execErr },
	}
	args := strings.Fields(strings.ReplaceAll(cmdline, "$D", dir))
	return run(args, c), out.String(), errb.String()
}

// fixture writes configs into a temp XDG dir. The default config loads
// known_hosts and has groups web (w1, w2) and empty; inv/ lists hosts from
// the inventory; nest/ has group top with no hosts over leaf (l1); bad/ and
// badinv/ do not load.
func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", filepath.Join(dir, "home"))
	t.Setenv("TMUX", "") // never reach a real tmux server
	for name, body := range map[string]string{
		"sshelf/config.toml": "version = 1\n[defaults]\nload_known_hosts = true\n",
		"sshelf/hosts.toml":  "version = 1\n[[groups]]\nname = \"web\"\nhosts = [\"w1\", \"w2\"]\n[[groups]]\nname = \"empty\"\n",
		"known_hosts":        "kh1 ssh-ed25519 AAAA\nkh2,kh3 ssh-rsa AAAA\n",
		"inv/config.toml":    "version = 1\n[defaults]\nload_known_hosts = false\n",
		"inv/hosts.toml":     "version = 1\n[[hosts]]\nhost = \"inv1\"\n[[groups]]\nname = \"g\"\nhosts = [\"inv2\"]\n",
		"bad/config.toml":    "not = = toml\n",
		"badinv/hosts.toml":  "version = 1\n[[groups]]\nname = \"bad/name\"\n",
		"nest/config.toml":   "version = 1\n",
		"nest/hosts.toml":    "version = 1\n[[groups]]\nname = \"top\"\nuser = \"admin\"\n[[groups]]\nname = \"leaf\"\nparent = \"top\"\nhosts = [\"l1\"]\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const kh = "-known-hosts $D/known_hosts "

func TestRun(t *testing.T) {
	tc := []struct {
		cmdline string
		code    int
		stdout  string
		errHas  string
	}{
		{"-h", 0, "", "Usage:\n  sshelf [flags]"},
		{"-bogus", 2, "", "flag provided but not defined: -bogus"},
		{"bogus", 1, "", `unknown command "bogus"`},
		{"__pane", 2, "", ""},
		{"__pane -- sshelf-no-such-binary", 2, "", ""}, // "__pane" itself is not the HOST

		{kh + "list", 1, "", "list requires a subcommand"},
		{kh + "list groups", 0, "web (2 hosts)\nempty (0 hosts)\n", ""},
		{kh + "l g --json", 0, "[\n  {\n    \"name\": \"web\",\n    \"hosts\": [\n      \"w1\",\n      \"w2\"\n    ]\n  },\n  {\n    \"name\": \"empty\",\n    \"hosts\": []\n  }\n]\n", ""},
		{kh + "list hosts", 0, "kh1\nkh2\nkh3\n", ""},
		{kh + "list h -json", 0, "[\n  \"kh1\",\n  \"kh2\",\n  \"kh3\"\n]\n", ""},
		{"list hosts -json", 0, "[]\n", ""}, // no ~/.ssh/known_hosts in the fixture
		{kh + "list x", 1, "", `unknown list subcommand "x"`},
		{kh + "list groups -bad", 1, "", "flag provided but not defined: -bad"},
		{"-config $D/inv/config.toml list hosts", 0, "inv1\ninv2\n", ""},
		{"-config $D/inv/config.toml -hosts $D/sshelf/hosts.toml list groups", 0, "web (2 hosts)\nempty (0 hosts)\n", ""},
		{"-config $D/bad/config.toml list groups", 1, "", "warning: config migration failed"},
		{"-config $D/inv/config.toml -hosts $D/badinv/hosts.toml list groups", 1, "", `group "bad/name"`},

		{"connect", 1, "", "connect requires a subcommand"},
		{"connect group", 1, "", "connect group requires a name"},
		{"c host", 1, "", "connect host requires a name"},
		{"connect x", 1, "", `unknown connect subcommand "x"`},
		{"connect group nope", 1, "", `group "nope" not found`},
		{"connect g EMPTY", 1, "", `group "EMPTY" has no hosts`},
		{"-no-tmux connect group WEB", 1, "", "multi-host requires tmux"},

		{"completion", 1, "", "completion requires a shell"},
		{"comp fish", 1, "", `unknown shell "fish"`},

		{kh + "__complete groups", 0, "web\nempty\n", ""},
		{kh + "__complete hosts", 0, "kh1\nkh2\nkh3\n", ""},
		{"__complete x", 0, "", ""},
		{"__complete", 0, "", ""},
	}
	for _, c := range tc {
		t.Run(c.cmdline, func(t *testing.T) {
			p := &proc{}
			code, out, errOut := sshelf(t, p, fixture(t), c.cmdline)
			if code != c.code || out != c.stdout || !strings.Contains(errOut, c.errHas) {
				t.Fatalf("code=%d stdout=%q stderr=%q\nwant code=%d stdout=%q stderr has %q", code, out, errOut, c.code, c.stdout, c.errHas)
			}
			if p.tuiOpts != nil || len(p.execed) != 0 {
				t.Fatalf("unexpected tui=%v exec=%q", p.tuiOpts, p.execed)
			}
		})
	}
}

func TestRunCompletionScripts(t *testing.T) {
	for shell, want := range map[string]string{"bash": "complete -F _sshelf sshelf", "zsh": "#compdef sshelf"} {
		code, out, _ := sshelf(t, &proc{}, fixture(t), "completion "+shell)
		if code != 0 || !strings.Contains(out, want) {
			t.Fatalf("%s: code=%d, stdout lacks %q", shell, code, want)
		}
	}
}

func TestRunConnectHostExecsSSH(t *testing.T) {
	p := &proc{}
	code, _, errOut := sshelf(t, p, fixture(t), "-no-tmux connect host web1")
	if code != 0 || len(p.execed) != 1 {
		t.Fatalf("code=%d exec=%q stderr=%q", code, p.execed, errOut)
	}
	if argv := p.execed[0]; argv[0] != "ssh" || argv[len(argv)-1] != "web1" {
		t.Fatalf("exec argv=%q, want ssh ... web1", argv)
	}

	p = &proc{execErr: errors.New("exec failed")}
	if code, _, errOut := sshelf(t, p, fixture(t), "-no-tmux c h web1"); code != 1 || !strings.Contains(errOut, "exec failed") {
		t.Fatalf("exec error: code=%d stderr=%q", code, errOut)
	}
}

// A parent group connects the hosts of its subgroups with the settings they
// inherit: top has no hosts of its own, and leaf takes its user.
func TestRunConnectNestedGroup(t *testing.T) {
	p := &proc{}
	code, _, errOut := sshelf(t, p, fixture(t), "-no-tmux -config $D/nest/config.toml connect group top")
	if want := [][]string{{"ssh", "--", "admin@l1"}}; code != 0 || !reflect.DeepEqual(p.execed, want) {
		t.Fatalf("code=%d exec=%q stderr=%q, want exec %q", code, p.execed, errOut, want)
	}
}

func TestRunTUI(t *testing.T) {
	p := &proc{}
	dir := fixture(t)
	if code, _, _ := sshelf(t, p, dir, kh+"-popup -no-tmux"); code != 0 || p.tuiOpts == nil {
		t.Fatalf("code=%d tui called=%v", code, p.tuiOpts != nil)
	}
	o := p.tuiOpts
	want := ui.Options{
		ConfigPath: filepath.Join(dir, "sshelf/config.toml"), InventoryPath: filepath.Join(dir, "sshelf/hosts.toml"),
		KnownHosts: []string{filepath.Join(dir, "known_hosts")}, Hosts: []string{"kh1", "kh2", "kh3"},
		Popup: true, NoTmux: true,
	}
	got := ui.Options{ConfigPath: o.ConfigPath, InventoryPath: o.InventoryPath, KnownHosts: o.KnownHosts, Hosts: o.Hosts, Popup: o.Popup, NoTmux: o.NoTmux}
	if !reflect.DeepEqual(got, want) || len(o.Inventory.Groups) != 2 || !o.Config.Defaults.LoadKnownHosts {
		t.Fatalf("tui options:\n%+v\nwant:\n%+v", got, want)
	}

	// With load_known_hosts off, the TUI gets inventory hosts and no paths.
	// Without -popup and -no-tmux, both are off.
	p = &proc{}
	sshelf(t, p, dir, kh+"-config $D/inv/config.toml")
	if p.tuiOpts.KnownHosts != nil || !reflect.DeepEqual(p.tuiOpts.Hosts, []string{"inv1", "inv2"}) {
		t.Fatalf("inventory hosts: KnownHosts=%q Hosts=%q", p.tuiOpts.KnownHosts, p.tuiOpts.Hosts)
	}
	if p.tuiOpts.Popup || p.tuiOpts.NoTmux {
		t.Fatalf("flag defaults: Popup=%v NoTmux=%v, want false", p.tuiOpts.Popup, p.tuiOpts.NoTmux)
	}
}

// Inside tmux a group opens as panes of one window: sshelf reports it and
// execs nothing. A fake tmux on PATH stands in for the server.
func TestRunConnectGroupInTmux(t *testing.T) {
	dir := fixture(t)
	bin := filepath.Join(dir, "bin")
	fake := "#!/bin/sh\ncase \"$1\" in new-window) echo '@1 %1';; split-window) echo '%2';; esac\n"
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte(fake), 0o700); err != nil { // #nosec G306 -- test script must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TMUX", "/tmp/fake,1,0")

	p := &proc{}
	code, _, errOut := sshelf(t, p, dir, "connect group web")
	if code != 0 || errOut != "opened 2 in one window\n" || len(p.execed) != 0 {
		t.Fatalf("code=%d stderr=%q exec=%q", code, errOut, p.execed)
	}
}

func TestRunTUIResult(t *testing.T) {
	tc := []struct {
		name   string
		tuiErr error
		code   int
		exec   [][]string
	}{
		{"exec request", &ui.ExecRequest{Cmd: []string{"ssh", "h"}}, 0, [][]string{{"ssh", "h"}}},
		{"quit", ui.ErrQuit, 0, nil},
		{"error", errors.New("tui broke"), 1, nil},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			p := &proc{tuiErr: c.tuiErr}
			code, _, _ := sshelf(t, p, fixture(t), "")
			if code != c.code || !reflect.DeepEqual(p.execed, c.exec) {
				t.Fatalf("code=%d exec=%q, want %d %q", code, p.execed, c.code, c.exec)
			}
		})
	}
}

func TestExecReplaceErrors(t *testing.T) {
	if err := execReplace(nil); err == nil || err.Error() != "empty exec command" {
		t.Fatalf("empty: err=%v", err)
	}
	if err := execReplace([]string{"sshelf-no-such-binary"}); err == nil {
		t.Fatal("missing binary: want error")
	}
}
