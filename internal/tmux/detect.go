package tmux

import "os"

func InTmux() bool {
	return os.Getenv("TMUX") != ""
}
