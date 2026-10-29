package hosts

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
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
	if err != nil || home == "" {
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

	out := make([]string, 0, len(set))
	for h := range set {
		out = append(out, h)
	}
	sort.Strings(out)

	return LoadResult{Hosts: out, SkippedLines: skipped}, errs
}

func ParseKnownHosts(r io.Reader) ([]string, int, error) {
	s := bufio.NewScanner(r)
	// known_hosts lines can be long because of key material; bump buffer.
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	set := make(map[string]struct{})
	skipped := 0

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			skipped++
			continue
		}
		if strings.HasPrefix(line, "@") {
			skipped++
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			skipped++
			continue
		}
		first := fields[0]
		if strings.HasPrefix(first, "|1|") {
			skipped++
			continue
		}

		for _, raw := range strings.Split(first, ",") {
			h := strings.TrimSpace(raw)
			if h == "" {
				continue
			}
			// Hashed hostnames are not displayable in MVP.
			if strings.HasPrefix(h, "|1|") {
				continue
			}
			set[h] = struct{}{}
		}
	}

	if err := s.Err(); err != nil {
		return nil, skipped, err
	}

	hosts := make([]string, 0, len(set))
	for h := range set {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)

	return hosts, skipped, nil
}
