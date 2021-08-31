package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/plugin"

	vagrantB "github.com/hashicorp/packer-plugin-vagrant/builder/vagrant"
	vagrantPP "github.com/hashicorp/packer-plugin-vagrant/post-processor/vagrant"
	vagrantCloudPP "github.com/hashicorp/packer-plugin-vagrant/post-processor/vagrant-cloud"
	"github.com/hashicorp/packer-plugin-vagrant/version"
)

func main() {
	pps := plugin.NewSet()
	pps.RegisterBuilder(plugin.DEFAULT_NAME, new(vagrantB.Builder))
	pps.RegisterPostProcessor(plugin.DEFAULT_NAME, new(vagrantPP.PostProcessor))
	pps.RegisterPostProcessor("cloud", new(vagrantCloudPP.PostProcessor))
	pps.SetVersion(version.PluginVersion)
	err := pps.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
