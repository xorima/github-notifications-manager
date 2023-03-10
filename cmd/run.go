/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/xorima/github-notifications-manager/config"
	"github.com/xorima/github-notifications-manager/manager"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		h := manager.NewManager(config.AppConfig)
		h.Handle()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&config.AppConfig.OrgName, "org-name", "o", "*", "The name of the organisation to mark as read, defaults to all (*)")
	runCmd.Flags().StringVarP(&config.AppConfig.State, "state", "s", "closed,merged", "The states to mark as read (csv), defaults to closed,merged")
	runCmd.Flags().BoolVarP(&config.AppConfig.DryRun, "dry-run", "d", false, "Dry run, don't actually mark anything as read")
}
