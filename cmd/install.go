package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jim-ww/gowebwrap/internal/desktopentry"
)

func newInstallCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install a desktop launcher for this site, so it shows up like a normal app",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := resolveConfig(v)
			if err != nil {
				return err
			}
			configFile, _ := cmd.Flags().GetString("config")
			path, err := desktopentry.Install(cfg, configFile)
			if err != nil {
				return err
			}
			cmd.Printf("Installed launcher: %s\n", path)
			return nil
		},
	}
}
