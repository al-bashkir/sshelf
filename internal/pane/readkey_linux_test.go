package pane

import (
	"fmt"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/charmbracelet/x/term"
)

func ioctl(f *os.File, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

// openPTY opens a new pseudo-terminal pair, not the user's terminal. The test
// types on the master side; the slave is a real terminal for readKey.
func openPTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	t.Cleanup(func() { _ = master.Close() })
	var unlock int32
	var n uint32
	if err := ioctl(master, syscall.TIOCSPTLCK, unsafe.Pointer(&unlock)); err != nil {
		t.Skipf("no pty: %v", err)
	}
	if err := ioctl(master, syscall.TIOCGPTN, unsafe.Pointer(&n)); err != nil {
		t.Skipf("no pty: %v", err)
	}
	slave, err = os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	t.Cleanup(func() { _ = slave.Close() })
	return master, slave
}

func TestReadKeyOnTerminalNeedsNoEnterAndRestoresMode(t *testing.T) {
	master, slave := openPTY(t)
	before, err := term.GetState(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	setStdin(t, slave)

	got := make(chan byte, 1)
	go func() { k, _ := readKey(); got <- k }()
	if _, err := master.Write([]byte("r")); err != nil {
		t.Fatal(err)
	}
	select {
	case k := <-got:
		if k != 'r' {
			t.Fatalf("readKey=%q want 'r'", k)
		}
	case <-time.After(3 * time.Second):
		_, _ = master.Write([]byte("\n")) // let the line-mode read return
		<-got
		t.Fatal("readKey waited for Enter: the terminal was not in raw mode")
	}

	after, err := term.GetState(slave.Fd())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatal("readKey left the terminal mode changed")
	}
}
