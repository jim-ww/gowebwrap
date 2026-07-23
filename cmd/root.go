// Package cmd is the CLI routing layer: flag/config parsing and command
// wiring only. Business logic lives in internal/wailsrun and
// internal/desktopentry, which know nothing about Cobra or Viper.
package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/jim-ww/gowebwrap/internal/config"
	"github.com/jim-ww/gowebwrap/internal/wailsrun"
)

func Execute() error {
	return NewRootCmd().Execute()
}

func NewRootCmd() *cobra.Command {
	v := viper.New()

	root := &cobra.Command{
		Use:           "gowebwrap",
		Short:         "Wrap any URL as a lightweight desktop app",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(v, cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := resolveConfig(v)
			if err != nil {
				return err
			}
			return wailsrun.Run(cfg)
		},
	}

	registerAppFlags(root.PersistentFlags())
	root.PersistentFlags().String("config", "", "path to a gowebwrap.toml config file; flags override its values (default: ./gowebwrap.toml if present)")

	root.AddCommand(newInstallCmd(v))
	return root
}

// registerAppFlags defines every setting a site config carries, so both the
// root command (run the webview) and `install` (write a launcher for it)
// accept the same flags.
func registerAppFlags(flags *pflag.FlagSet) {
	flags.String("name", "", "app name (defaults to the URL)")
	flags.String("description", "", "app description")
	flags.String("url", "", "URL to wrap (required)")
	flags.String("icon", "", "path to a PNG icon file, read at startup")
	flags.Int("window-width", 1024, "window width")
	flags.Int("window-height", 768, "window height")
	flags.Int("window-max-width", 0, "window max width (0 = unset)")
	flags.Int("window-max-height", 0, "window max height (0 = unset)")
	flags.Int("window-min-width", 0, "window min width (0 = unset)")
	flags.Int("window-min-height", 0, "window min height (0 = unset)")
	flags.Bool("always-on-top", false, "keep window always on top")
	flags.Bool("disable-resize", false, "disable window resizing")
	flags.Bool("frameless", false, "hide the window frame/titlebar")
	flags.String("background-color", "#06070f", "window background color as #rrggbb, shown before the page loads")
	flags.Bool("dark-theme", true, "force dark theme for web content (Linux)")
	flags.Bool("gpu-acceleration", false, "enable GPU-accelerated webview compositing (can render blank/white on some Linux setups, e.g. nested/sandboxed Wayland sessions)")
	flags.String("user-agent", config.DefaultUserAgent, "User-Agent sent by the webview (some sites reject Wails' default UA as bot traffic)")
}

func initConfig(v *viper.Viper, cmd *cobra.Command) error {
	if cfgFile, _ := cmd.Flags().GetString("config"); cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.AddConfigPath(".")
		v.SetConfigType("toml")
		v.SetConfigName("gowebwrap")
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("reading config: %w", err)
		}
	}

	return v.BindPFlags(cmd.Flags())
}

func resolveConfig(v *viper.Viper) (config.Config, error) {
	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("decoding config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}
