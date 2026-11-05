package hosts

import (
	"bufio"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type LoadResult struct {
	Hosts        []string
	SkippedLines int
}

type PathError struct {
	Path string
	Err  error
}

func (e PathError) Error() string {
	if e.Path == "" {
		return e.Err.Error()
	}
	return e.Path + ": " + e.Err.Error()
}

func DefaultKnownHostsPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil { // UserHomeDir fails when $HOME is empty
		return nil
	}
	return []string{filepath.Join(home, ".ssh", "known_hosts")}
}

func LoadKnownHosts(paths []string) (LoadResult, []PathError) {
	if len(paths) == 0 {
		paths = DefaultKnownHostsPaths()
	}

	set := make(map[string]struct{})
	var skipped int
	var errs []PathError

	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}

		// #nosec G304 -- path is user-provided by design (CLI flag / config).
		f, err := os.Open(p)
		if err != nil {
			errs = append(errs, PathError{Path: p, Err: err})
			continue
		}

		hosts, sk, perr := ParseKnownHosts(f)
		_ = f.Close()
		skipped += sk
		if perr != nil {
			errs = append(errs, PathError{Path: p, Err: perr})
			continue
		}

		for _, h := range hosts {
			set[h] = struct{}{}
		}
	}

	return LoadResult{Hosts: slices.Sorted(maps.Keys(set)), SkippedLines: skipped}, errs
}

func ParseKnownHosts(r io.Reader) ([]string, int, error) {
	s := bufio.NewScanner(r)
	// known_hosts lines can be long because of key material; bump buffer.
	s.Buffer(nil, 1024*1024)

	set := make(map[string]struct{})
	skipped := 0

	for s.Scan() {
		names, ok := lineHosts(s.Text())
		if !ok {
			skipped++
			continue
		}
		for _, h := range names {
			set[h] = struct{}{}
		}
	}

	if err := s.Err(); err != nil {
		return nil, skipped, err
	}
	return slices.Sorted(maps.Keys(set)), skipped, nil
}

// lineHosts returns the host names of one known_hosts line. ok is false for
// a line to skip: blank, comment, @marker, or a hashed first host.
func lineHosts(line string) (names []string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "@") {
		return nil, false
	}
	first := strings.Fields(line)[0] // line is trimmed and non-empty
	if strings.HasPrefix(first, "|1|") {
		return nil, false
	}
	for _, raw := range strings.Split(first, ",") {
		h := strings.TrimSpace(raw)
		// Hashed hostnames are not displayable in MVP.
		if h != "" && !strings.HasPrefix(h, "|1|") {
			names = append(names, h)
		}
	}
	return names, true
}
