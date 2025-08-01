
The Vagrant plugin integrates Packer with HashiCorp [Vagrant](https://www.vagrantup.com/), allowing you to use Packer to create development boxes.

### Installation
To install this plugin add this code into your Packer configuration and run [packer init](/packer/docs/commands/init)

```hcl
packer {
  required_plugins {
    vagrant = {
      version = "~> 1"
      source = "github.com/hashicorp/vagrant"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
packer plugins install github.com/hashicorp/vagrant
```

**Note: Update to Packer Plugin Installation**

With the new Packer release starting from version 1.14.0, the `packer init` command will automatically install official plugins from the [HashiCorp release site.](https://releases.hashicorp.com/)

Going forward, to use newer versions of official Packer plugins, you'll need to upgrade to Packer version 1.14.0 or later. If you're using an older version, you can still install plugins, but as a workaround, you'll need to [manually install them using the CLI.](https://developer.hashicorp.com/packer/docs/plugins/install#manually-install-plugins-using-the-cli)

There is no change to the syntax or commands for installing plugins.

### Components

#### Builders
- [vagrant](/packer/integrations/hashicorp/vagrant/latest/components/builder/vagrant) - The Vagrant builder is intended for building new boxes from already-existing boxes.

#### Post-Processor
- [vagrant](/packer/integrations/hashicorp/vagrant/latest/components/post-processor/vagrant) - The Packer Vagrant post-processor takes a build and converts the artifact into a valid Vagrant box.
- [vagrant-cloud](/packer/integrations/hashicorp/vagrant/latest/components/post-processor/vagrant-cloud) - The Vagrant Cloud post-processor enables the upload of Vagrant boxes to Vagrant Cloud.
- [vagrant-registry](/packer/integrations/hashicorp/vagrant/latest/components/post-processor/vagrant-registry) - The Vagrant Registry post-processor enables the upload of Vagrant boxes to HCP Vagrant Box Registry. 
