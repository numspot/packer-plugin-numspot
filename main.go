// Package main is the entry point for the numspot-plugin-packer.
package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/plugin"
	"github.com/hashicorp/packer-plugin-sdk/version"

	"github.com/numspot/numspot-plugin-packer/builder/bsu"
	"github.com/numspot/numspot-plugin-packer/datasource/image"
)

func main() {
	pps := plugin.NewSet()
	pps.SetVersion(pluginVersion)
	pps.RegisterBuilder("bsu", new(bsu.Builder))
	pps.RegisterDatasource("image", new(image.Datasource))
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

var (
	pluginVersion = version.NewPluginVersion(semver, prerelease, "")
	semver        = "0.1.1"
	prerelease    = ""
)
