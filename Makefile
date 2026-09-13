# Build and packaging for sipexer (engine) and sipx (wrapper CLI).
#
# Local build (host binaries at repo root):
#   make                build ./sipexer and ./sipx for this machine
#   make sipexer        build only the engine
#   make sipx           build only the wrapper
#   make test           run go vet + go test
#   make clean          remove host binaries and staging dirs
#
# Packaging (mac + linux, amd64 + arm64; CGO is not required):
#   make dist           build all four cross-compiled bundles into dist/
#   make dist-one OS=linux ARCH=amd64   one bundle
#
# Each dist/ bundle is a tar.gz laid out for staging-install:
#   bin/sipexer                    the engine
#   bin/sipx                       the wrapper (finds bin/sipexer next to it)
#   share/doc/sipx/SIPX.md         standalone usage doc
#
# Install (system dirs, like a package manager would do):
#   make install        copies host binaries + doc to /usr/local (override: PREFIX=/opt/sipx)
#   sudo make install   ...or run with sudo
#   make uninstall      remove what 'make install' placed
#
#   make help           show all targets

VERSION ?= $(shell sed -n 's/.*sipexerVersion = "\(.*\)".*/\1/p' sipexer.go 2>/dev/null || echo dev)
PREFIX  ?= /usr/local
BINDIR  := $(PREFIX)/bin
DOCDIR  := $(PREFIX)/share/doc/sipx

BIN_SIPEXER := sipexer
BIN_SIPX    := sipx

# CGO is not needed by either binary; keep it off for portable cross builds.
GOBUILD := CGO_ENABLED=0 go build -trimpath

# os/arch pairs to package: mac + linux, amd64 + arm64.
PLATFORMS := darwin-amd64 darwin-arm64 linux-amd64 linux-arm64

.PHONY: all sipexer sipx test vet clean dist dist-one install uninstall help

all: sipexer sipx

sipexer:
	$(GOBUILD) -o $(BIN_SIPEXER) .

sipx:
	$(GOBUILD) -o $(BIN_SIPX) ./cmd/sipx

test: vet
	go test ./...

vet:
	go vet ./...

## packaging -------------------------------------------------------------

# build_bundle <os> <arch> -> dist/sipexer-sipx-<ver>-<os>-<arch>.tar.gz
# Uses a self-contained shell recipe so it works inside the dist for-loop.

dist:
	@mkdir -p dist
	@set -e; \
	for p in $(PLATFORMS); do \
	  os=$${p%-*}; arch=$${p#*-}; \
	  out=dist/sipexer-sipx-$(VERSION)-$$os-$$arch.tar.gz; \
	  rm -rf dist/stage; mkdir -p dist/stage/bin dist/stage/share/doc/sipx; \
	  echo "building $$os/$$arch"; \
	  GOOS=$$os GOARCH=$$arch $(GOBUILD) -o dist/stage/bin/sipexer .; \
	  GOOS=$$os GOARCH=$$arch $(GOBUILD) -o dist/stage/bin/sipx ./cmd/sipx; \
	  cp SIPX.md dist/stage/share/doc/sipx/SIPX.md; \
	  tar -C dist/stage -czf $$out bin share; \
	  rm -rf dist/stage; \
	  echo "  packed $$out"; \
	done
	@rm -rf dist/stage

# dist-one builds a single bundle for the host (override OS/ARCH), e.g.
#   make dist-one OS=linux ARCH=arm64
OS   ?= $(shell uname -s | tr 'A-Z' 'a-z')
ARCH ?= $(shell uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/; s/arm64/arm64/')
dist-one:
	@mkdir -p dist
	@set -e; \
	out=dist/sipexer-sipx-$(VERSION)-$(OS)-$(ARCH).tar.gz; \
	rm -rf dist/stage; mkdir -p dist/stage/bin dist/stage/share/doc/sipx; \
	echo "building $(OS)/$(ARCH)"; \
	GOOS=$(OS) GOARCH=$(ARCH) $(GOBUILD) -o dist/stage/bin/sipexer .; \
	GOOS=$(OS) GOARCH=$(ARCH) $(GOBUILD) -o dist/stage/bin/sipx ./cmd/sipx; \
	cp SIPX.md dist/stage/share/doc/sipx/SIPX.md; \
	tar -C dist/stage -czf $$out bin share; \
	rm -rf dist/stage; \
	echo "  packed $$out"

## install ---------------------------------------------------------------

install: all
	mkdir -p $(DESTDIR)$(BINDIR) $(DESTDIR)$(DOCDIR)
	cp $(BIN_SIPEXER) $(DESTDIR)$(BINDIR)/sipexer
	cp $(BIN_SIPX)    $(DESTDIR)$(BINDIR)/sipx
	cp SIPX.md        $(DESTDIR)$(DOCDIR)/SIPX.md
	@echo "installed: $(BINDIR)/sipexer $(BINDIR)/sipx  (doc: $(DOCDIR)/SIPX.md)"

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/sipexer $(DESTDIR)$(BINDIR)/sipx
	rm -rf $(DESTDIR)$(DOCDIR)
	@echo "removed $(BINDIR)/sip{exer,x} and $(DOCDIR)"

## housekeeping ----------------------------------------------------------

clean:
	rm -f $(BIN_SIPEXER) $(BIN_SIPX)
	rm -rf dist

help:
	@echo "targets:"
	@echo "  make                build host binaries ./sipexer and ./sipx"
	@echo "  make sipexer|sipx   build one of them"
	@echo "  make test           go vet + go test ./..."
	@echo "  make dist           cross-compile bundles for mac/linux x amd64/arm64 into dist/"
	@echo "  make dist-one OS=linux ARCH=arm64   build a single bundle (defaults to host os/arch)"
	@echo "  make install        stage host binaries + SIPX.md under PREFIX (default /usr/local); DESTDIR for staging"
	@echo "  make uninstall      remove what 'make install' placed"
	@echo "  make clean          remove binaries and dist/"
	@echo "  version = $(VERSION)"
