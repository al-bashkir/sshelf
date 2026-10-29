package main

import (
	"fmt"

	"github.com/al-bashkir/sshelf/internal/config"
)

func runCompletion(args []string) {
	if len(args) == 0 {
		fatal(fmt.Errorf("completion requires a shell: bash or zsh\nUsage: sshelf completion bash|zsh"))
	}
	switch args[0] {
	case "bash":
		fmt.Print(bashCompletionScript)
	case "zsh":
		fmt.Print(zshCompletionScript)
	default:
		fatal(fmt.Errorf("unknown shell %q: use bash or zsh", args[0]))
	}
}

// runInternalComplete is called by shell completion scripts to get dynamic candidates.
// It prints one entry per line and is intentionally silent on errors.
func runInternalComplete(args []string, inv config.Inventory, knownHosts []string) {
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "groups":
		for _, g := range inv.Groups {
			fmt.Println(g.Name)
		}
	case "hosts":
		for _, h := range knownHosts {
			fmt.Println(h)
		}
	}
	// unknown token → print nothing (graceful for completion scripts)
}

const bashCompletionScript = `# bash completion for sshelf
# Usage:
#   source <(sshelf completion bash)
# Or add to ~/.bashrc:
#   eval "$(sshelf completion bash)"

_sshelf() {
  COMPREPLY=()
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local cmd="${COMP_WORDS[1]}"
  local subcmd="${COMP_WORDS[2]}"

  # Complete flags when the current word starts with -
  if [[ "$cur" == -* ]]; then
    local flags="-config -hosts -known-hosts -no-tmux -popup"
    if [[ "$cmd" == list || "$cmd" == l ]]; then
      flags="$flags -json"
    fi
    COMPREPLY=($(compgen -W "$flags" -- "$cur"))
    return
  fi

  case $COMP_CWORD in
    1)
      COMPREPLY=($(compgen -W "connect c list l completion" -- "$cur"))
      ;;
    2)
      case $cmd in
        connect|c)
          COMPREPLY=($(compgen -W "group g host h" -- "$cur"))
          ;;
        list|l)
          COMPREPLY=($(compgen -W "groups g hosts h" -- "$cur"))
          ;;
        completion)
          COMPREPLY=($(compgen -W "bash zsh" -- "$cur"))
          ;;
      esac
      ;;
    3)
      case $cmd in
        connect|c)
          case $subcmd in
            group|g)
              COMPREPLY=($(compgen -W "$(sshelf __complete groups 2>/dev/null)" -- "$cur"))
              ;;
            host|h)
              COMPREPLY=($(compgen -W "$(sshelf __complete hosts 2>/dev/null)" -- "$cur"))
              ;;
          esac
          ;;
      esac
      ;;
  esac
}

complete -F _sshelf sshelf
`

const zshCompletionScript = `#compdef sshelf
# zsh completion for sshelf
# Usage (one-time setup):
#   mkdir -p ~/.zfunc
#   sshelf completion zsh > ~/.zfunc/_sshelf
# Then add to ~/.zshrc (before compinit):
#   fpath=(~/.zfunc $fpath)
#   autoload -Uz compinit && compinit

_sshelf() {
  local context state line
  typeset -A opt_args

  # _arguments -C drives flag and subcommand completion.
  # Flags use the '[description]:message:action' format understood by
  # _arguments; _describe (used below for commands) uses 'item:description'.
  _arguments -C -s \
    '-config[path to config.toml]:file:_files' \
    '-hosts[path to hosts.toml]:file:_files' \
    '*-known-hosts[known_hosts path]:file:_files' \
    '-no-tmux[disable tmux integration]' \
    '-popup[quit after connecting (for tmux popup)]' \
    '1: :->cmd' \
    '*:: :->args' && return

  case $state in
    cmd)
      local -a cmds
      cmds=(
        'connect:connect to a host or group'
        'c:alias for connect'
        'list:list groups or hosts'
        'l:alias for list'
        'completion:output shell completion script'
      )
      _describe 'command' cmds
      ;;
    args)
      # In this state $words[1] is the subcommand and $CURRENT is the
      # index within the subcommand context.
      case $words[1] in
        connect|c)
          case $CURRENT in
            2)
              local -a sub
              sub=(
                'group:connect to all hosts in a group'
                'g:alias for group'
                'host:connect to a specific host'
                'h:alias for host'
              )
              _describe 'subcommand' sub
              ;;
            3)
              case $words[2] in
                group|g)
                  local -a groups
                  groups=(${(f)"$(sshelf __complete groups 2>/dev/null)"})
                  _describe 'group' groups
                  ;;
                host|h)
                  local -a hosts
                  hosts=(${(f)"$(sshelf __complete hosts 2>/dev/null)"})
                  _describe 'host' hosts
                  ;;
              esac
              ;;
          esac
          ;;
        list|l)
          _arguments \
            '-json[output as JSON]' \
            '1: :(groups g hosts h)' && return
          ;;
        completion)
          local -a shells
          shells=('bash:bash completion script' 'zsh:zsh completion script')
          _describe 'shell' shells
          ;;
      esac
      ;;
  esac
}

_sshelf "$@"
`
