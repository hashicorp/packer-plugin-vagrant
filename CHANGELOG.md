## 1.1.6 (August 5, 2025)

### IMPROVEMENTS:

* docs: fixed formatting and diffs from actual implementation

* core: bump packer-plugin-sdk to v0.6.1

* Updated plugin release process: Plugin binaries are now published on the HashiCorp official [release site](https://releases.hashicorp.com/packer-plugin-vagrant), ensuring a secure and standardized delivery pipeline.

### NOTES:
* **Binary Distribution Update**: To streamline our release process and align with other HashiCorp tools, all release binaries will now be published exclusively to the official HashiCorp [release](https://releases.hashicorp.com/packer-plugin-vagrant) site. We will no longer attach release assets to GitHub Releases. Any scripts or automation that rely on the old location will need to be updated. For more information, see our post [here](https://discuss.hashicorp.com/t/important-update-official-packer-plugin-distribution-moving-to-releases-hashicorp-com/75972).

## 1.0.1 (December 22, 2021)

* post-procesor/vagrant-cloud: Add `box_checksum` argument to allow for setting  
  checksums on uploaded box. [GH-32]
  
## 1.0.0 (June 15, 2021)

* Update to v0.2.3 of packer-plugin-sdk. [GH-18]

## 0.0.3 (April 21, 2021)

* Refactor docs and fix goreleaser configuration.

## 0.0.2 (April 13, 2021)

* Update docs generation with the packer-sdc command
* No-op tag; debugging release workflow.

## 0.0.1 (April 13, 2021)

* Extract Vagrant builders and post-processors to packer-plugin-vagrant
