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

# Both Linux arches are built via `nix build` (see the `linux)` case in
# `build` below), not a raw `go build` — Wails' GTK3/webkitgtk backend needs
# cgo, and a plain cross `go build` has no matching arm64 GTK3/webkitgtk
# sysroot to link against. `nix build .#packages.<system>.default` sidesteps
# that: aarch64-linux is a first-class nixpkgs platform, so its GTK3/
# webkitgtk come prebuilt from cache.nixos.org regardless of host arch —
# only our own small Go package actually needs building for the target arch,
# via QEMU user-mode emulation (`boot.binfmt.emulatedSystems` on NixOS; must
# be registered on whichever machine runs `make build`).
# windows/amd64 and windows/arm64 cross-compile cleanly with a plain `go
# build` from any host, since Wails' WebView2 backend is pure Go (no cgo).
TARGETS := linux/amd64 linux/arm64 windows/amd64 windows/arm64

# Builds and packages every target in $(TARGETS) into $(DIST_DIR): one
# APPNAME_GOOS_GOARCH.tar.gz per target (containing just the binary —
# renamed to $(APP_NAME), with a .exe suffix added on windows — plus the
# README and LICENSE), and one APPNAME_VERSION_checksums.txt covering all of
# them.
build:
	@test -n "$(VERSION)" || { echo "no tag on HEAD — run 'git tag vX.Y.Z' first, or set TAG=vX.Y.Z"; exit 1; }
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	for target in $(TARGETS); do \
		goos=$${target%%/*}; goarch=$${target##*/}; \
		echo "==> building $$goos/$$goarch"; \
		ext=""; [ "$$goos" = windows ] && ext=.exe; \
		workdir=$$(mktemp -d); \
		case "$$goos" in \
			linux) \
				case "$$goarch" in \
					amd64) nixsystem=x86_64-linux ;; \
					arm64) nixsystem=aarch64-linux ;; \
					*) echo "unknown linux GOARCH $$goarch" >&2; exit 1 ;; \
				esac; \
				nix build ".#packages.$$nixsystem.default" -o "$$workdir/result"; \
				cp "$$workdir/result/bin/.$(APP_NAME)-wrapped" "$$workdir/$(APP_NAME)$$ext"; \
				;; \
			windows) \
				nix develop -c bash -c " \
					unset GOFLAGS; \
					CGO_ENABLED=0 GOOS=windows GOARCH=$$goarch \
						go build -tags desktop,production -ldflags '-s -w -H=windowsgui' \
						-o '$$workdir/$(APP_NAME)$$ext' ."; \
				;; \
			*) echo "unknown GOOS $$goos" >&2; exit 1 ;; \
		esac; \
		cp README.md LICENSE "$$workdir/"; \
		tar -C "$$workdir" -czf "$(DIST_DIR)/$(APP_NAME)_$${goos}_$${goarch}.tar.gz" \
			"$(APP_NAME)$$ext" README.md LICENSE; \
		rm -rf "$$workdir"; \
	done
	$(MAKE) checksums

# Extracted out of `build` so it can be rerun on its own (e.g. after hand-
# editing $(DIST_DIR)) without re-running every build, via the same
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
# Builds every target in $(TARGETS) on this one machine — see the TARGETS
# comment above for what that requires (QEMU emulation registered for the
# linux/arm64 leg).
release: build changelog
	nix develop -c gh release create "$(VERSION)" \
		--title "$(VERSION)" \
		--notes-file "$(DIST_DIR)/changelog.txt" \
		$(DIST_DIR)/$(APP_NAME)_*.tar.gz \
		$(DIST_DIR)/$(APP_NAME)_$(VERSION)_checksums.txt
