package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/atcharles/glibs/util"
)

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

var rootCmd = &cobra.Command{
	Use: util.AppName,
}

var (
	daemon bool
	isInit bool
)

func init() {
	rootCmd.PersistentFlags().BoolP("x", "x", false, "set daemon")
	rootCmd.PersistentFlags().BoolVarP(&isInit, "init", "i", false, "init database")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run as daemon")
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().BoolVar(&drop, "drop", false, "drop database")
}

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(`%s
Version: %s
CompileMod: %s
BuildTime: %s
GoVersion: %s
GitBranch: %s
GitHash: %s
`,
			util.AppName,
			util.Version,
			util.CompileMod,
			util.BuildTime,
			util.GoVersion,
			util.GitBranch,
			util.GitHash,
		)
	},
}
