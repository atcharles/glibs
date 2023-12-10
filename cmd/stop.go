package cmd

import (
	"github.com/spf13/cobra"

	"github.com/atcharles/glibs/boot"
)

var BeforeStopFunc func()
var DropFunc = boot.Drop
var drop = false
var dropCmd = &cobra.Command{
	Use: "drop",
	Run: func(*cobra.Command, []string) {
		DropFunc()
	},
}
var stopCmd = &cobra.Command{
	Use: "stop",
	Run: func(*cobra.Command, []string) {
		if BeforeStopFunc != nil {
			BeforeStopFunc()
		}
		killProcess()
		if drop {
			DropFunc()
		}
	},
}
