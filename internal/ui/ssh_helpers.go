package ui

import "strings"

func ensureSSHForceTTY(extraArgs []string) []string {
	out := append([]string(nil), extraArgs...)
	for _, a := range out {
		if a == "-t" || a == "-tt" {
			return out
		}
	}
	return append(out, "-t")
}

func keepSessionOpenRemoteCmd(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ""
	}
	// Avoid doubling when the caller already includes it.
	if strings.Contains(cmd, "exec ${SHELL") || strings.Contains(cmd, "exec $SHELL") {
		return cmd
	}
	return cmd + "; exec ${SHELL:-sh}"
}
