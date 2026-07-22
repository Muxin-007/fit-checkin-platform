package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"platform/common/version"
	"platform/starter"
)

var (
	configPath                                                          string
	showVersion                                                         bool
	moduleName, moduleVersion, branch, revision, goVersion, compileTime string
)

func init() {
	pflag.StringVarP(&configPath, "config", "f", "/etc/fitness/platform.yaml", "Specify server config file location")
	pflag.BoolVarP(&showVersion, "version", "v", false, "Print version information")
}

func main() {
	pflag.Parse()
	if showVersion {
		versionInfo := version.NewVersionInfo(
			moduleName,
			moduleVersion,
			branch,
			revision,
			goVersion,
			compileTime,
		)
		fmt.Println(versionInfo.String())
		os.Exit(0)
	}

	// Start server
	starter.Start(configPath)
}
