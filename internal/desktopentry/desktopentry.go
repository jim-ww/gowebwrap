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
	"os/exec"
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
	exe, err := invokedPath()
	if err != nil {
		return "", fmt.Errorf("finding this binary's own path: %w", err)
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

// invokedPath returns the path the current process was actually invoked as
// (os.Args[0], resolved via PATH if it was a bare name), NOT os.Executable().
// On Nix, the installed `gowebwrap` is a wrapper script (from wrapProgram)
// that sets GIO_EXTRA_MODULES/SSL_CERT_FILE/etc. before exec-ing into the
// real, hidden binary — so os.Executable() (which reads /proc/self/exe)
// resolves past that wrapper to the unwrapped binary once we're running.
// Baking that unwrapped path into the .desktop's Exec= would skip all of
// the wrapper's env setup on every future launch (symptom: webview requests
// fail with "TLS support is not available" — the missing GIO_EXTRA_MODULES
// glib-networking module). os.Args[0] instead reflects the wrapper itself,
// since that's what the shell/nix actually invoked.
func invokedPath() (string, error) {
	arg0 := os.Args[0]
	if strings.ContainsRune(arg0, os.PathSeparator) {
		return filepath.Abs(arg0)
	}
	return exec.LookPath(arg0)
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
