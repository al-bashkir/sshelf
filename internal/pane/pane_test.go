package pane

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDecide(t *testing.T) {
	tc := []struct {
		key  byte
		want Action
	}{
		{'r', Reconnect},
		{'R', Reconnect},
		{'\r', Close},
		{'\n', Close},
		{'q', Close},
		{0x03, Close}, // Ctrl-C
		{0x04, Close}, // Ctrl-D
		{'x', Ignore},
		{' ', Ignore},
	}
	for _, tt := range tc {
		if got := Decide(tt.key); got != tt.want {
			t.Errorf("Decide(%q)=%v want=%v", tt.key, got, tt.want)
		}
	}
}

func TestArgvRoundTrip(t *testing.T) {
	ssh := []string{"ssh", "-p", "2222", "--", "web01"}
	argv := Argv("/usr/bin/sshelf", "web01", ssh)

	if want := []string{"/usr/bin/sshelf", "__pane", "web01", "--", "ssh", "-p", "2222", "--", "web01"}; !reflect.DeepEqual(argv, want) {
		t.Fatalf("Argv=%v want=%v", argv, want)
	}
	host, cmd, err := parseArgs(argv[2:])
	if err != nil || host != "web01" || !reflect.DeepEqual(cmd, ssh) {
		t.Fatalf("parseArgs=%q %v %v", host, cmd, err)
	}
}

func TestParseArgsRejectsBadInput(t *testing.T) {
	for _, args := range [][]string{nil, {"web01"}, {"web01", "--"}, {"web01", "ssh", "web01"}} {
		if _, _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%q) want error", args)
		}
	}
}

// fake records what the session did and feeds it scripted exits and keys.
type fake struct {
	exits []int
	keys  string
	runs  int
	syncs []bool
	out   bytes.Buffer
}

func (f *fake) session() session {
	return session{
		host: "web01",
		ssh: func() int {
			code := f.exits[f.runs]
			f.runs++
			return code
		},
		key: func() (byte, error) {
			if f.keys == "" {
				return 0, io.EOF
			}
			k := f.keys[0]
			f.keys = f.keys[1:]
			return k, nil
		},
		isolate: func(on bool) { f.syncs = append(f.syncs, on) },
		out:     &f.out,
	}
}

func TestRunUserExitClosesAtOnce(t *testing.T) {
	f := &fake{exits: []int{1}}
	if got := f.session().run(); got != 1 {
		t.Fatalf("exit=%d want=1", got)
	}
	if f.runs != 1 || f.out.Len() != 0 || f.syncs != nil {
		t.Fatalf("runs=%d out=%q syncs=%v, want one silent run", f.runs, f.out.String(), f.syncs)
	}
}

func TestRunLostThenReconnect(t *testing.T) {
	f := &fake{exits: []int{255, 0}, keys: "xr"}
	if got := f.session().run(); got != 0 {
		t.Fatalf("exit=%d want=0", got)
	}
	if f.runs != 2 {
		t.Fatalf("runs=%d want=2", f.runs)
	}
	if want := []bool{true, false}; !reflect.DeepEqual(f.syncs, want) {
		t.Fatalf("syncs=%v want=%v", f.syncs, want)
	}
	if !strings.Contains(f.out.String(), "Connection to web01 lost") {
		t.Fatalf("out=%q, want lost prompt", f.out.String())
	}
	// The dead session may leave alt screen and mouse tracking on: no history, no scroll.
	if !strings.HasPrefix(f.out.String(), "\033[?1049l\033[?1000l") {
		t.Fatalf("out=%q, want terminal modes reset before the prompt", f.out.String())
	}
}

func TestRunLostThenClose(t *testing.T) {
	for _, keys := range []string{"\r", ""} { // Enter, or the input ends
		f := &fake{exits: []int{255}, keys: keys}
		if got := f.session().run(); got != 255 {
			t.Fatalf("keys=%q exit=%d want=255", keys, got)
		}
		if f.runs != 1 || !reflect.DeepEqual(f.syncs, []bool{true}) {
			t.Fatalf("keys=%q runs=%d syncs=%v", keys, f.runs, f.syncs)
		}
	}
}

func TestRunSSHReturnsExitCode(t *testing.T) {
	tc := []struct {
		argv []string
		want int
	}{
		{[]string{"true"}, 0},
		{[]string{"sh", "-c", "exit 255"}, 255},
		{[]string{"/nonexistent/ssh"}, 127},
	}
	for _, tt := range tc {
		if got := runSSH(tt.argv); got != tt.want {
			t.Errorf("runSSH(%v)=%d want=%d", tt.argv, got, tt.want)
		}
	}
}

func TestMainPassesUserExitThrough(t *testing.T) {
	if got := Main([]string{"web01", "--", "sh", "-c", "exit 3"}); got != 3 {
		t.Fatalf("Main=%d want=3", got)
	}
	if got := Main([]string{"web01"}); got != 2 {
		t.Fatalf("Main(bad args)=%d want=2", got)
	}
}

// setStdin points os.Stdin at f for one test, so readKey never reads the real terminal.
func setStdin(t *testing.T, f *os.File) {
	t.Helper()
	old := os.Stdin
	os.Stdin = f
	t.Cleanup(func() { os.Stdin = old })
}

// pipeStdin makes os.Stdin a pipe that holds input and then ends.
func pipeStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if _, err := w.WriteString(input); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	setStdin(t, r)
}

func TestReadKeyFromPipe(t *testing.T) {
	pipeStdin(t, "r") // not a terminal: no raw mode, read anyway
	if k, err := readKey(); k != 'r' || err != nil {
		t.Fatalf("readKey=%q,%v want 'r',nil", k, err)
	}
	if k, err := readKey(); k != 0 || err != io.EOF {
		t.Fatalf("readKey at end=%q,%v want 0,EOF", k, err)
	}
}

// fakeTmux puts a tmux stand-in first in PATH. It records each call's argv
// (one arg per line, then "--call--") in the returned file.
func fakeTmux(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" --call-- >> '" + log + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte(script), 0o700); err != nil { // #nosec G306 -- test stub must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func TestMainLostConnectionTakesPaneOutOfSync(t *testing.T) {
	off := "set-option\n-p\n-t\n%7\nsynchronize-panes\noff\n--call--\n"
	back := "set-option\n-p\n-u\n-t\n%7\nsynchronize-panes\n--call--\n"
	for _, tc := range []struct{ pane, want string }{
		{"%7", off + back + off},
		{"", ""}, // not in a tmux pane: tmux is never run
	} {
		t.Run("pane="+tc.pane, func(t *testing.T) {
			log := fakeTmux(t)
			t.Setenv("TMUX_PANE", tc.pane)
			pipeStdin(t, "r") // reconnect once, then the input ends: close
			out, err := os.Create(filepath.Join(t.TempDir(), "out"))
			if err != nil {
				t.Fatal(err)
			}
			old := os.Stdout
			os.Stdout = out
			t.Cleanup(func() { os.Stdout = old; _ = out.Close() })

			if got := Main([]string{"web01", "--", "sh", "-c", "exit 255"}); got != 255 {
				t.Fatalf("Main=%d want=255", got)
			}
			calls, _ := os.ReadFile(log) // no file: no calls
			if string(calls) != tc.want {
				t.Fatalf("tmux calls=%q want=%q", calls, tc.want)
			}
			printed, _ := os.ReadFile(out.Name())
			if n := strings.Count(string(printed), "Connection to web01 lost"); n != 2 {
				t.Fatalf("prompt shown %d times, want 2: %q", n, printed)
			}
		})
	}
}

func TestParseArgsMinimal(t *testing.T) {
	host, cmd, err := parseArgs([]string{"web01", "--", "ssh"})
	if err != nil || host != "web01" || !reflect.DeepEqual(cmd, []string{"ssh"}) {
		t.Fatalf("parseArgs=%q %v %v", host, cmd, err)
	}
}
