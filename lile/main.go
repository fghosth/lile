package main

import (
	"fmt"
	"os"

	"github.com/fghosth/lile/lile/cmd"
)
var (
	Version   string
	BuildTime string
	GoVersion string
)
func main() {
	cmd.GoVersion = GoVersion
	cmd.BuildTime = BuildTime
	cmd.Version = Version
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
