.PHONY: dev build release changelog checksums

dev:
	nix develop -c wails3 dev

APP_NAME := gowebwrap
DIST_DIR := dist

# A release is always named after a tag — there's no "snapshot" concept
# here. Defaults to the tag on HEAD; set TAG (env var or `make build
# TAG=vX.Y.Z`) to override without needing an actual tag on HEAD. Both
# `build` and `release` refuse to run without one so the archive/checksums/
# release names below are never guessed at.
TAG ?=
VERSION := $(if $(TAG),$(TAG),$(shell git describe --tags --exact-match 2>/dev/null))

HOST_ARCH := $(shell uname -m)
ifeq ($(HOST_ARCH),x86_64)
  NATIVE_LINUX_ARCH := amd64
else ifeq ($(HOST_ARCH),aarch64)
  NATIVE_LINUX_ARCH := arm64
else
  NATIVE_LINUX_ARCH := $(HOST_ARCH)
endif

# linux/$(NATIVE_LINUX_ARCH) only: Wails' GTK3/webkitgtk backend needs cgo,
# and nix has no ready-made cross GTK3/webkitgtk sysroot to cross-compile the
# other Linux arch from here (see flake.nix) — so running this on an x86_64
# machine ships linux/amd64 but not linux/arm64, and vice versa on arm64.
# windows/amd64 and windows/arm64 always cross-compile cleanly from either
# host, since Wails' WebView2 backend is pure Go (no cgo, unlike linux).
TARGETS := linux/$(NATIVE_LINUX_ARCH) windows/amd64 windows/arm64

# Builds and packages every target in $(TARGETS) into $(DIST_DIR): one
# APPNAME_GOOS_GOARCH.tar.gz per target (containing just the binary —
# renamed to $(APP_NAME), with a .exe suffix added on windows — plus the
# README and LICENSE), and one APPNAME_VERSION_checksums.txt covering all of
# them.
build:
	@test -n "$(VERSION)" || { echo "no tag on HEAD — run 'git tag vX.Y.Z' first, or set TAG=vX.Y.Z"; exit 1; }
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	nix develop -c bash -c ' \
		set -eu; \
		unset GOFLAGS; \
		for target in $(TARGETS); do \
			goos=$${target%%/*}; goarch=$${target##*/}; \
			echo "==> building $$goos/$$goarch"; \
			case "$$goos" in \
				linux) cgo=1; tags=desktop,production,gtk3; ldflags="-s -w" ;; \
				windows) cgo=0; tags=desktop,production; ldflags="-s -w -H=windowsgui" ;; \
				*) echo "unknown GOOS $$goos" >&2; exit 1 ;; \
			esac; \
			ext=""; [ "$$goos" = windows ] && ext=.exe; \
			workdir=$$(mktemp -d); \
			CGO_ENABLED=$$cgo GOOS=$$goos GOARCH=$$goarch \
				go build -tags "$$tags" -ldflags "$$ldflags" -o "$$workdir/$(APP_NAME)$$ext" .; \
			cp README.md LICENSE "$$workdir/"; \
			tar -C "$$workdir" -czf "$(DIST_DIR)/$(APP_NAME)_$${goos}_$${goarch}.tar.gz" \
				"$(APP_NAME)$$ext" README.md LICENSE; \
			rm -rf "$$workdir"; \
		done \
	'
	$(MAKE) checksums

# Extracted out of `build` so a release job assembling $(DIST_DIR) from
# several runners' worth of `make build TARGETS=...` output (a single
# machine can't produce every arch, see the TARGETS comment above) can
# regenerate one checksums file covering everything, via the same
# tag/naming logic instead of a second copy of it elsewhere.
checksums:
	@test -n "$(VERSION)" || { echo "no tag on HEAD — run 'git tag vX.Y.Z' first, or set TAG=vX.Y.Z"; exit 1; }
	cd $(DIST_DIR) && sha256sum $(APP_NAME)_*.tar.gz > $(APP_NAME)_$(VERSION)_checksums.txt
	@echo "built $(DIST_DIR)/$(APP_NAME)_$(VERSION)_checksums.txt covering:"
	@cd $(DIST_DIR) && cat $(APP_NAME)_$(VERSION)_checksums.txt

# Plain commit-log changelog in $(DIST_DIR)/changelog.txt, scoped to commits
# since the previous tag (or full history if this is the first tag) — no
# nix needed, this only touches git.
changelog:
	@test -n "$(VERSION)" || { echo "no tag on HEAD — run 'git tag vX.Y.Z' first, or set TAG=vX.Y.Z"; exit 1; }
	@mkdir -p $(DIST_DIR)
	@prev=$$(git describe --tags --abbrev=0 "$(VERSION)^" 2>/dev/null || true); \
	{ \
		echo "Changelog"; \
		echo; \
		if [ -n "$$prev" ]; then \
			git log --oneline --no-decorate "$$prev..$(VERSION)"; \
		else \
			git log --oneline --no-decorate "$(VERSION)"; \
		fi | sed 's/^/    /'; \
	} > $(DIST_DIR)/changelog.txt
	@cat $(DIST_DIR)/changelog.txt

# Same as `build`, plus publishing $(DIST_DIR) to a GitHub Release named
# after the tag on HEAD (via `gh`, so `gh auth login` must already be done).
# Only covers this machine's own targets — see the TARGETS comment above —
# so a full multi-arch release still needs one `make release` per Linux arch.
release: build changelog
	nix develop -c gh release create "$(VERSION)" \
		--title "$(VERSION)" \
		--notes-file "$(DIST_DIR)/changelog.txt" \
		$(DIST_DIR)/$(APP_NAME)_*.tar.gz \
		$(DIST_DIR)/$(APP_NAME)_$(VERSION)_checksums.txt
