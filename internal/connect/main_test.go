package connect

import (
	"os"
	"testing"
)

// TestMain cuts the tests off from the user's tmux server. A bug or a mutant
// that reaches tmux must fail, not open real panes: those panes would run this
// test binary again (os.Executable) and spawn more panes without end.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "tmuxtest")
	if err != nil {
		panic(err)
	}
	_ = os.Unsetenv("TMUX")
	_ = os.Unsetenv("TMUX_PANE")
	_ = os.Setenv("TMUX_TMPDIR", dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
