package cmd

import (
	"github.com/spf13/cobra"

	"github.com/atcharles/glibs/boot"
)

var drop = false

var stopCmd = &cobra.Command{
	Use: "stop",
	Run: func(*cobra.Command, []string) {
		killProcess()
		if drop {
			boot.Drop()
		}
	},
}
