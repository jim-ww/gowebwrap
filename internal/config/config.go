// Package config defines the settings that describe a wrapped site: window
// geometry, identity (name/description/icon), and webview behavior. The same
// struct is populated from a TOML file, CLI flags, or a mix of both (flags
// win) — see cmd/root.go for the merge order.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultUserAgent is a real Chrome-on-Linux UA. Wails' Linux/GTK3 backend
// hardcodes a UA that appends a non-standard "wails.io" token to WebKit's
// default string, which gets some sites flat-out rejected
// by bot-detection before any JS runs.
const DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// Config is the full set of options for one wrapped site.
type Config struct {
	Name            string `mapstructure:"name" toml:"name"`
	Description     string `mapstructure:"description" toml:"description"`
	URL             string `mapstructure:"url" toml:"url"`
	Icon            string `mapstructure:"icon" toml:"icon,omitempty"`
	Width           int    `mapstructure:"window-width" toml:"window-width"`
	Height          int    `mapstructure:"window-height" toml:"window-height"`
	MaxWidth        int    `mapstructure:"window-max-width" toml:"window-max-width,omitempty"`
	MaxHeight       int    `mapstructure:"window-max-height" toml:"window-max-height,omitempty"`
	MinWidth        int    `mapstructure:"window-min-width" toml:"window-min-width,omitempty"`
	MinHeight       int    `mapstructure:"window-min-height" toml:"window-min-height,omitempty"`
	AlwaysOnTop     bool   `mapstructure:"always-on-top" toml:"always-on-top"`
	DisableResize   bool   `mapstructure:"disable-resize" toml:"disable-resize"`
	Frameless       bool   `mapstructure:"frameless" toml:"frameless"`
	BackgroundColor string `mapstructure:"background-color" toml:"background-color"`
	DarkTheme       bool   `mapstructure:"dark-theme" toml:"dark-theme"`
	GPUAcceleration bool   `mapstructure:"gpu-acceleration" toml:"gpu-acceleration"`
	UserAgent       string `mapstructure:"user-agent" toml:"user-agent"`
}

// Validate checks the fields that have no sane default (namely URL) and
// fills in the ones that fall back to another field's value.
func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required (set --url or \"url\" in the config file)")
	}
	if c.Name == "" {
		c.Name = c.URL
	}
	return nil
}

// ParseBackgroundColor parses BackgroundColor ("#rrggbb") into RGB bytes.
func (c Config) ParseBackgroundColor() (r, g, b uint8, err error) {
	s := strings.TrimPrefix(c.BackgroundColor, "#")
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("background-color must be in #rrggbb form, got %q", c.BackgroundColor)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid background-color %q: %w", c.BackgroundColor, err)
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), nil
}
