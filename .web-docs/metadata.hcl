# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "Vagrant"
  description = "The Vagrant multi-component plugin can be used with HashiCorp Packer to create custom images."
  identifier = "packer/hashicorp/vagrant"
  component {
    type = "builder"
    name = "Vagrant"
    slug = "vagrant"
  }
  component {
    type = "post-processor"
    name = "Vagrant"
    slug = "vagrant"
  }
  component {
    type = "post-processor"
    name = "Vagrant Cloud"
    slug = "vagrant-cloud"
  }
  component {
    type = "post-processor"
    name = "Vagrant Registry"
    slug = "vagrant-registry"
  }
}
