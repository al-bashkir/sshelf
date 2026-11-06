%global app_version 0.0.0

Name:           sshelf
Version:        %{app_version}
Release:        1%{?dist}
Summary:        Terminal UI for managing SSH connections

# Disable the empty debuginfo subpackage that Go stripped binaries produce.
%global debug_package %{nil}

License:        MIT
URL:            https://github.com/al-bashkir/sshelf

# Source tarball is prepared by .copr/Makefile, which runs go mod vendor
# before creating the archive, so the vendor/ directory is already present.
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.26.10
BuildRequires:  git-core

# sshelf shells out to the system ssh binary
Requires:       openssh-clients
# tmux is required only for multi-host sessions
Recommends:     tmux

%description
sshelf is a terminal UI for browsing and connecting to SSH hosts.
It reads your SSH config and known_hosts files and presents an
interactive host picker built with Bubble Tea. Single-host connections
exec ssh directly; multi-host sessions are managed through tmux.

%prep
%autosetup

%build
export CGO_ENABLED=0
go build \
    -trimpath \
    -mod=vendor \
    -ldflags "-s -w" \
    -o %{name} \
    ./cmd/sshelf

%install
install -Dpm 0755 %{name} %{buildroot}%{_bindir}/%{name}

# Shell completion scripts
./%{name} completion bash > %{name}.bash
install -Dpm 0644 %{name}.bash %{buildroot}%{_datadir}/bash-completion/completions/%{name}

./%{name} completion zsh > _sshelf
install -Dpm 0644 _sshelf %{buildroot}%{_datadir}/zsh/site-functions/_sshelf

%post
# Regenerate completion files so they always match the installed binary.
%{_bindir}/%{name} completion bash > %{_datadir}/bash-completion/completions/%{name} 2>/dev/null || :
%{_bindir}/%{name} completion zsh  > %{_datadir}/zsh/site-functions/_sshelf         2>/dev/null || :

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_datadir}/bash-completion/completions/%{name}
%{_datadir}/zsh/site-functions/_sshelf

%changelog
* Fri Nov 06 2026 Al Bashkir <41126916+al-bashkir@users.noreply.github.com> - 0.2.0-1
- Let a group sit inside another group; know a group by its path, such as prod/web
- Connect, list and complete groups by path on the command line
- Connect a parent group together with the hosts of its subgroups
- Show groups as a tree, open them like folders, and list subgroups above hosts
- Pick a parent in the group form; rename, move or delete a group together with its subgroups
- Add typed hosts to a group with Ctrl+g on the Connect popup
- Preview the colorscheme and accent in Settings
- Save forms with Ctrl+S from insert mode
- Do not ask before connecting when the threshold is 0
- Reset terminal modes before the reconnect prompt
- Show the "removed N" toast after removing hosts from a group
- Keep the full list height in the border-format picker after adding a format
- Scroll a form picker that has more options than rows
- Fit the help overlay, modal frames, long modal titles and the tab box into the window
- Keep the search cursor in view
- Fit the longest Settings labels in the label column
- Mark matched letters correctly in names with multi-byte or wide letters and in cut rows
- Restore every theme color when the colorscheme goes back to default
- Wrap whole host names in the group picker's "adding" lines

* Thu Oct 29 2026 Al Bashkir <41126916+al-bashkir@users.noreply.github.com> - 0.1.0-1
- First release
