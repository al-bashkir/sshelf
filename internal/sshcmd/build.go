package sshcmd

import (
	"strconv"
	"strings"

	"github.com/al-bashkir/sshelf/internal/config"
)

func shellQuotePOSIX(s string) string {
	if s == "" {
		return "''"
	}
	// Close/open quotes around escaped single quote:  ' -> '\''
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

type Settings struct {
	User          string
	Port          int
	IdentityFile  string
	ExtraArgs     []string
	RemoteCommand string
}

func FromDefaults(defaults config.Defaults) Settings {
	return Settings{
		User:         defaults.User,
		Port:         defaults.Port,
		IdentityFile: defaults.IdentityFile,
		ExtraArgs:    defaults.ExtraArgs,
	}
}

// ApplyGroup overrides the same SSH fields as ApplyHost, plus the remote command.
func ApplyGroup(base Settings, group config.Group) Settings {
	s := ApplyHost(base, config.Host{User: group.User, Port: group.Port, IdentityFile: group.IdentityFile, ExtraArgs: group.ExtraArgs})
	if strings.TrimSpace(group.RemoteCommand) != "" {
		s.RemoteCommand = group.RemoteCommand
	}
	return s
}

func ApplyHost(base Settings, host config.Host) Settings {
	s := base
	if strings.TrimSpace(host.User) != "" {
		s.User = host.User
	}
	if host.Port != 0 {
		s.Port = host.Port
	}
	if strings.TrimSpace(host.IdentityFile) != "" {
		s.IdentityFile = host.IdentityFile
	}
	if len(host.ExtraArgs) != 0 {
		s.ExtraArgs = host.ExtraArgs
	}
	return s
}

// BuildCommand returns the full command slice starting with "ssh".
func BuildCommand(host string, s Settings) []string {
	baseHost := strings.TrimSpace(host)
	if baseHost == "" {
		return []string{"ssh"}
	}

	sshPort := s.Port
	if h, p, ok := parseBracketHost(baseHost); ok {
		baseHost = h
		sshPort = p
	}

	target := baseHost
	if s.User != "" {
		target = s.User + "@" + baseHost
	}

	cmd := []string{"ssh"}
	if s.IdentityFile != "" {
		cmd = append(cmd, "-i", s.IdentityFile)
	}
	if sshPort != 0 && sshPort != 22 {
		cmd = append(cmd, "-p", strconv.Itoa(sshPort))
	}
	if len(s.ExtraArgs) != 0 {
		cmd = append(cmd, s.ExtraArgs...)
	}
	// "--" keeps a host such as "-oProxyCommand=..." from being read as an option.
	cmd = append(cmd, "--", target)

	if rc := strings.TrimSpace(s.RemoteCommand); rc != "" {
		// ssh executes the remote command through a shell. If we pass args like
		// ["sh","-c","ls -lah"], ssh will serialize them into a string without
		// preserving argv boundaries, and the remote shell will split the script.
		// To keep the script intact, pass a single shell command string.
		cmd = append(cmd, "sh -c "+shellQuotePOSIX(rc))
	}

	return cmd
}

func parseBracketHost(s string) (host string, port int, ok bool) {
	// known_hosts uses "[host]:port".
	if !strings.HasPrefix(s, "[") {
		return "", 0, false
	}
	idx := strings.LastIndex(s, "]:")
	if idx < 0 {
		return "", 0, false
	}
	h := s[1:idx]
	if h == "" {
		return "", 0, false
	}
	ps := s[idx+2:]
	p, err := strconv.Atoi(ps)
	if err != nil || p <= 0 {
		return "", 0, false
	}
	return h, p, true
}
