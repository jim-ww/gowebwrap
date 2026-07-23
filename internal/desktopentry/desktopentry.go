// Package desktopentry gives non-Nix users the same "acts like a real app"
// result Nix users get from the flake's lib.mkApp: a .desktop launcher with
// the site's own name/icon, installed into the user's XDG directories.
//
// Unlike lib.mkApp, nothing is baked at build time — install writes a
// canonical TOML config (so the launcher has a stable file to point at) and
// a .desktop entry whose Exec line just re-invokes this same generic binary
// with --config pointing at that file.
package desktopentry

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/jim-ww/gowebwrap/internal/config"
)

var slugInvalid = regexp.MustCompile(`[^a-z0-9-]+`)

func slugify(name string) string {
	s := slugInvalid.ReplaceAllString(strings.ToLower(name), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "app"
	}
	return s
}

func xdgDir(envVar, fallbackRelToHome string) (string, error) {
	if v := os.Getenv(envVar); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, fallbackRelToHome), nil
}

// Install writes a canonical config file (merging cfg's current values into
// configFile if given, or a fresh file under the user's config dir
// otherwise) and a .desktop launcher that points at it. It returns the path
// of the installed .desktop file.
func Install(cfg config.Config, configFile string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding this binary's own path: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	slug := slugify(cfg.Name)

	if configFile == "" {
		configDir, err := xdgDir("XDG_CONFIG_HOME", ".config")
		if err != nil {
			return "", err
		}
		configFile = filepath.Join(configDir, "gowebwrap", slug+".toml")
	}
	configFile, err = filepath.Abs(configFile)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return "", fmt.Errorf("encoding config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(configFile, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("writing config %q: %w", configFile, err)
	}

	var iconPath string
	if cfg.Icon != "" {
		iconPath, err = filepath.Abs(cfg.Icon)
		if err != nil {
			return "", err
		}
	}

	dataDir, err := xdgDir("XDG_DATA_HOME", ".local/share")
	if err != nil {
		return "", err
	}
	desktopDir := filepath.Join(dataDir, "applications")
	if err := os.MkdirAll(desktopDir, 0o755); err != nil {
		return "", err
	}
	desktopFile := filepath.Join(desktopDir, "gowebwrap-"+slug+".desktop")

	var entry strings.Builder
	fmt.Fprintf(&entry, "[Desktop Entry]\n")
	fmt.Fprintf(&entry, "Type=Application\n")
	fmt.Fprintf(&entry, "Name=%s\n", cfg.Name)
	if cfg.Description != "" {
		fmt.Fprintf(&entry, "Comment=%s\n", cfg.Description)
	}
	fmt.Fprintf(&entry, "Exec=%s --config %s\n", quoteDesktopExec(exe), quoteDesktopExec(configFile))
	if iconPath != "" {
		fmt.Fprintf(&entry, "Icon=%s\n", iconPath)
	}
	fmt.Fprintf(&entry, "Terminal=false\n")
	fmt.Fprintf(&entry, "Categories=Network;\n")

	if err := os.WriteFile(desktopFile, []byte(entry.String()), 0o644); err != nil {
		return "", fmt.Errorf("writing desktop entry %q: %w", desktopFile, err)
	}

	return desktopFile, nil
}

// quoteDesktopExec applies the quoting the Exec key of the freedesktop.org
// desktop entry spec requires for arguments containing spaces or quotes.
func quoteDesktopExec(s string) string {
	if !strings.ContainsAny(s, " \t\"'\\$`") {
		return s
	}
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`).Replace(s)
	return `"` + escaped + `"`
}
