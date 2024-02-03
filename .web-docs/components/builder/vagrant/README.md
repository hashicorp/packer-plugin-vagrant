Type: `vagrant`
Artifact BuilderId: `vagrant`

The Vagrant builder is intended for building new boxes from already-existing
boxes. Your source should be a URL or path to a .box file or a Vagrant Cloud
box name such as `hashicorp/precise64`.

Packer will not install vagrant, nor will it install the underlying
virtualization platforms or extra providers; We expect when you run this
builder that you have already installed what you need.

By default, this builder will initialize a new Vagrant workspace, launch your
box from that workspace, provision it, call `vagrant package` to package it
into a new box, and then destroy the original box. Please note that vagrant
will _not_ remove the box file from your system (we don't call
`vagrant box remove`).

You can change the behavior so that the builder doesn't destroy the box by
setting the `teardown_method` option. You can change the behavior so the builder
doesn't package it (not all provisioners support the `vagrant package` command)
by setting the `skip package` option. You can also change the behavior so that
rather than initializing a new Vagrant workspace, you use an already defined
one, by using `global_id` instead of `source_box`.

Please note that if you are using the Vagrant builder, then the Vagrant
post-processor is unnecessary because the output of the Vagrant builder is
already a Vagrant box; using that post-processor with the Vagrant builder will
cause your build to fail. Similarly, since Vagrant boxes are already compressed,
the Compress post-processor will not work with this builder.

## Configuration Reference

### Required

- `source_path` (string) - URL of the vagrant box to use, or the name of the
  vagrant box. `hashicorp/precise64`, `./mylocalbox.box` and
  `https://example.com/my-box.box` are all valid source boxes. If your
  source is a .box file, whether locally or from a URL like the latter example
  above, you will also need to provide a `box_name`. This option is required,
  unless you set `global_id`. You may only set one or the other, not both.

  or

- `global_id` (string) - the global id of a Vagrant box already added to Vagrant
  on your system. You can find the global id of your Vagrant boxes using the
  command `vagrant global-status`; your global_id will be a 7-digit number and
  letter combination that you'll find in the leftmost column of the
  global-status output. If you choose to use `global_id` instead of
  `source_box`, Packer will skip the Vagrant initialize and add steps, and
  simply launch the box directly using the global id.

### Optional

<!-- Code generated from the comments of the Config struct in builder/vagrant/builder.go; DO NOT EDIT MANUALLY -->

- `output_dir` (string) - The directory to create that will contain your output box. We always
  create this directory and run from inside of it to prevent Vagrant init
  collisions. If unset, it will be set to packer- plus your buildname.

- `checksum` (string) - The checksum for the .box file. The type of the checksum is specified
  within the checksum field as a prefix, ex: "md5:{$checksum}". The type
  of the checksum can also be omitted and Packer will try to infer it
  based on string length. Valid values are "none", "{$checksum}",
  "md5:{$checksum}", "sha1:{$checksum}", "sha256:{$checksum}",
  "sha512:{$checksum}" or "file:{$path}". Here is a list of valid checksum
  values:
   * md5:090992ba9fd140077b0661cb75f7ce13
   * 090992ba9fd140077b0661cb75f7ce13
   * sha1:ebfb681885ddf1234c18094a45bbeafd91467911
   * ebfb681885ddf1234c18094a45bbeafd91467911
   * sha256:ed363350696a726b7932db864dda019bd2017365c9e299627830f06954643f93
   * ed363350696a726b7932db864dda019bd2017365c9e299627830f06954643f93
   * file:http://releases.ubuntu.com/20.04/SHA256SUMS
   * file:file://./local/path/file.sum
   * file:./local/path/file.sum
   * none
  Although the checksum will not be verified when it is set to "none",
  this is not recommended since these files can be very large and
  corruption does happen from time to time.

- `box_name` (string) - if your source_box is a boxfile that we need to add to Vagrant, this is
  the name to give it. If left blank, will default to "packer_" plus your
  buildname.

- `insert_key` (bool) - If true, Vagrant will automatically insert a keypair to use for SSH,
  replacing Vagrant's default insecure key inside the machine if detected.
  By default, Packer sets this to false.

- `provider` (string) - The vagrant provider.
  This parameter is required when source_path have more than one provider,
  or when using vagrant-cloud post-processor. Defaults to unset.

- `vagrantfile_template` (string) - What vagrantfile to use

- `teardown_method` (string) - Whether to halt, suspend, or destroy the box when the build has
  completed. Defaults to "halt"

- `box_version` (string) - What box version to use when initializing Vagrant.

- `template` (string) - a path to a golang template for a vagrantfile. Our default template can
  be found [here](https://github.com/hashicorp/packer-plugin-vagrant/blob/main/builder/vagrant/step_create_vagrantfile.go#L39-L54). The template variables available to you are
  `{{ .BoxName }}`, `{{ .SyncedFolder }}`, and `{{.InsertKey}}`, which
  correspond to the Packer options box_name, synced_folder, and insert_key.
  Alternatively, the template variable `{{.DefaultTemplate}}` is available for
  use if you wish to extend the default generated template.

- `synced_folder` (string) - Path to the folder to be synced to the guest. The path can be absolute
  or relative to the directory Packer is being run from.

- `skip_add` (bool) - Don't call "vagrant add" to add the box to your local environment; this
  is necessary if you want to launch a box that is already added to your
  vagrant environment.

- `add_cacert` (string) - Equivalent to setting the
  --cacert
  option in vagrant add; defaults to unset.

- `add_capath` (string) - Equivalent to setting the
  --capath option
  in vagrant add; defaults to unset.

- `add_cert` (string) - Equivalent to setting the
  --cert option in
  vagrant add; defaults to unset.

- `add_clean` (bool) - Equivalent to setting the
  --clean flag in
  vagrant add; defaults to unset.

- `add_force` (bool) - Equivalent to setting the
  --force flag in
  vagrant add; defaults to unset.

- `add_insecure` (bool) - Equivalent to setting the
  --insecure flag in
  vagrant add; defaults to unset.

- `skip_package` (bool) - if true, Packer will not call vagrant package to
  package your base box into its own standalone .box file.

- `output_vagrantfile` (string) - Output Vagrantfile

- `package_include` ([]string) - Equivalent to setting the
  [`--include`](https://developer.hashicorp.com/vagrant/docs/cli/package#include-x-y-z) option
  in `vagrant package`; defaults to unset

<!-- End of code generated from the comments of the Config struct in builder/vagrant/builder.go; -->


## Example

Sample for `hashicorp/precise64` with virtualbox provider.

**JSON**

```json
{
  "builders": [
    {
      "communicator": "ssh",
      "source_path": "hashicorp/precise64",
      "provider": "virtualbox",
      "add_force": true,
      "type": "vagrant"
    }
  ]
}
```

**HCL2**

```hcl
source "vagrant" "example" {
  communicator = "ssh"
  source_path = "hashicorp/precise64"
  provider = "virtualbox"
  add_force = true
}

build {
  sources = ["source.vagrant.example"]
}
```


## Regarding output directory and new box

After Packer completes building and provisioning a new Vagrant Box file, it is worth
noting that the new box file will need to be added to Vagrant. For a beginner to Packer
and Vagrant, it may seem as if a simple 'vagrant up' in the output directory will run the
the newly created Box. This is not the case.

Rather, create a new directory (to avoid Vagarant init collisions), add the new
package.box to Vagrant and init. Then run vagrant up to bring up the new box created
by Packer. You will now be able to connect to the new box with provisioned changes.

```
'mkdir output2'
'cp package.box ./output2'
'vagrant box add new-box name-of-the-packer-box.box'
'vagrant init new-box'
'vagrant up'
```

## A note on SSH connections

Currently this builder only works for SSH connections, and automatically fills
in all information needed for the SSH communicator using vagrant's ssh-config.

If you would like to connect via a different username or authentication method
than is produced when you call `vagrant ssh-config`, then you must provide the

`ssh_username` and all other relevant authentication information (e.g.
`ssh_password` or `ssh_private_key_file`)

By providing the `ssh_username`, you're telling Packer not to use the vagrant
ssh config, except for determining the host and port for the virtual machine to
connect to.
