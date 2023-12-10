package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/atcharles/glibs/util"
)

func init() {
	RootCmd.PersistentFlags().BoolP("x", "x", false, "set daemon")
	RootCmd.PersistentFlags().BoolVarP(&isInit, "init", "i", false, "init database")
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(startCmd)
	startCmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run as daemon")
	RootCmd.AddCommand(stopCmd)
	stopCmd.Flags().BoolVar(&drop, "drop", false, "drop database")
	RootCmd.AddCommand(dropCmd)
}

var RootCmd = &cobra.Command{
	Use: util.AppName,
}
var (
	daemon bool
	isInit bool
)
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

func Execute() {
	cobra.CheckErr(RootCmd.Execute())
}
