# COPR SCM build hook.
# COPR calls: make -f .copr/Makefile srpm outdir=<path>
# This Makefile vendors Go dependencies into the source tarball and produces an SRPM.

outdir ?= .

SPEC      := $(CURDIR)/sshelf.spec
NAME      := sshelf
VERSION   ?= 0.0.0
TARBALL   := $(NAME)-$(VERSION).tar.gz
WORKDIR   := /tmp/copr-$(NAME)-$(VERSION)
SPEC_WORK := $(WORKDIR)/$(NAME).spec

.PHONY: srpm
srpm:
	# Install Go if not present (COPR SCM builder may not have it)
	command -v go >/dev/null 2>&1 || dnf install -y golang

	# Clean any leftover work directory
	rm -rf $(WORKDIR)
	mkdir -p $(WORKDIR)

	# Create source archive from the current git checkout
	git archive --prefix=$(NAME)-$(VERSION)/ HEAD \
		| gzip > $(WORKDIR)/$(TARBALL)

	# Extract, vendor Go dependencies, repack
	tar xf $(WORKDIR)/$(TARBALL) -C $(WORKDIR)/
	cd $(WORKDIR)/$(NAME)-$(VERSION) && go mod vendor
	tar czf $(WORKDIR)/$(TARBALL) -C $(WORKDIR) $(NAME)-$(VERSION)/

	# Stamp the version into a spec copy so it is baked into the SRPM.
	# COPR rebuilds from the spec inside the SRPM without any --define flags,
	# so the version must be a literal value in the spec, not a macro argument.
	sed 's/^%global app_version.*/%global app_version $(VERSION)/' \
		$(SPEC) > $(SPEC_WORK)

	# Build the SRPM
	rpmbuild -bs \
		--define "_sourcedir $(WORKDIR)" \
		--define "_srcrpmdir $(outdir)" \
		$(SPEC_WORK)

	# Clean up
	rm -rf $(WORKDIR)
