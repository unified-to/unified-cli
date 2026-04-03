package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Set by GoReleaser via ldflags
var Version = "1.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
