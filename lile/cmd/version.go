package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	Version   string
	BuildTime string
	GoVersion string
)
var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "show version",
	Example: "./peep version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Version:"+Version)
		fmt.Println("BuildTime:"+BuildTime)
		fmt.Println("GoVersion:"+GoVersion)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
