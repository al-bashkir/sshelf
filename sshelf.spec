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
* Thu Oct 29 2026 Al Bashkir <41126916+al-bashkir@users.noreply.github.com> - 0.1.0-1
- First release
