Type: `vagrant-registry`
Artifact BuilderId: `hashicorp.post-processor.vagrant-registry`

[HCP Vagrant Box Registry](https://portal.cloud.hashicorp.com/vagrant/discover)
hosts and serves boxes to Vagrant, allowing you to version and distribute boxes
to an organization or the public in a simple way.

The Vagrant Registry post-processor enables the upload of Vagrant boxes to HCP
Vagrant Box Registry. Currently, the Vagrant Registry post-processor will accept
and upload boxes supplied to it from the [Vagrant](/docs/post-processor/vagrant.mdx) or
[Artifice](https://developer.hashicorp.com/packer/docs/post-processor/artifice) post-processors and the
[Vagrant](/docs/builder/vagrant.mdx) builder.

## Workflow

It's important to understand the workflow that using this post-processor
enforces in order to take full advantage of Vagrant and HCP Vagrant Box Registry.

The use of this processor assume that you currently distribute, or plan to
distribute, boxes via HCP Vagrant Box Registry. It also assumes you create
Vagrant Boxes and deliver them to your team in some fashion.

Here is an example workflow:

1. You use Packer to build a Vagrant Box for the `virtualbox` provider
1. The `vagrant-registry` post-processor is configured to point to the box
    `hashicorp/foobar` on HCP Vagrant Box Registry via the `box_tag` configuration
1. The post-processor receives the box from the `vagrant` post-processor
1. It then creates the box name, or verifies the existence of it, on HCP
    Vagrant Box Registry
1. It then creates the configured version, or verifies the existence of it
1. It then creates the configured provider, or verifies the existence of it
1. It then creates the configured architecture, or verifies the existence of it
1. The box artifact is uploaded to HCP Vagrant Box Registry
1. The upload is verified
1. The version is released and available to users of the box

## Configuration

The configuration allows you to specify the target box that you have access to
on HCP Vagrant Box Registry, as well as authentication and version information.

### Required

- `box_tag` (string) - The shorthand tag for your box that maps to the box registry
  name and the box name. For example, `hashicorp/precise64` is the shorthand tag
  for the `precise64` box in the `hashicorp` box registry.

- `version` (string) - The version number, typically incrementing a previous
  version. The version string is validated based on [Semantic
  Versioning](http://semver.org/). The string must match a pattern that could
  be semver, and doesn't validate that the version comes after your previous
  versions.

- `client_id` (string) - The service principal client ID for the HCP API. This 
  value can be omitted if the `HCP_CLIENT_ID` environment variable is set. See 
  the [HCP documentation](https://developer.hashicorp.com/hcp/docs/hcp/admin/iam/service-principals)
  for creating a service principal.

- `client_secret` (string) - The service principal client secret for the HCP API. This 
  value can be omitted if the `HCP_CLIENT_SECRET` environment variable is set. See 
  the [HCP documentation](https://developer.hashicorp.com/hcp/docs/hcp/admin/iam/service-principals)
  for creating a service principal.

### Optional
- `architecture` (string) - The architecture of the Vagrant box. This will be
  detected from the box if possible by default. Supported values: amd64, i386,
  arm, arm64, ppc64le, ppc64, mips64le, mips64, mipsle, mips, and s390x.

- `default_architecture` (string) - The architecture that should be flagged as
  the default architecture for this provider. 

- `no_release` (boolean) - If set to true, does not release the version on
  HCP Vagrant Box Registry, making it active. You can manually release the version via
  the API or Web UI. Defaults to `false`.

- `keep_input_artifact` (boolean) - When true, preserve the local box
  after uploading to HCP Vagrant Box Registry. Defaults to `true`.

- `version_description` (string) - Optional Markdown text used as a
  full-length and in-depth description of the version, typically for denoting
  changes introduced

- `box_download_url` (string) - Optional URL for a self-hosted box.
  If this is set the box will not be uploaded to HCP Vagrant Box Registry.
  This is a [template engine](https://developer.hashicorp.com/packer/docs/templates/legacy_json_templates/engine).
  Therefore, you may use user variables and template functions in this field.
  The following extra variables are also available in this engine:

  - `Architecture`: The architecture of the Vagrant box
  - `Provider`: The Vagrant provider the box is for
  - `ArtifactId`: The ID of the input artifact.

- `box_checksum` (string) - Optional checksum for the provider .box file.
  The type of the checksum is specified within the checksum field as a prefix,
  ex: "md5:{$checksum}". Valid values are:
  - null or ""
  - "md5:{$checksum}"
  - "sha1:{$checksum}"
  - "sha256:{$checksum}"
  - "sha512:{$checksum}"

- `no_direct_upload` (boolean) - When `true`, upload the box artifact through
  HCP Vagrant Box Registry instead of directly to the backend storage.

## Use with the Vagrant Post-Processor

An example configuration is shown below. Note the use of the [post-processors](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/blocks/build/post-processors)
block that wraps both the Vagrant and Vagrant Registry [post-processor](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/blocks/build/post-processor) blocks within the post-processor section. Chaining
the post-processors together in this way tells Packer that the artifact
produced by the Vagrant post-processor should be passed directly to the Vagrant
Registry Post-Processor. It also sets the order in which the post-processors
should run.

Failure to chain the post-processors together in this way will result in the
wrong artifact being supplied to the Vagrant Registry post-processor. This will
likely cause the Vagrant Registry post-processor to error and fail.

**JSON**

```json
{
  "variables": {
    "hcp_client_id": "{{ env `HCP_CLIENT_ID` }}",
    "hcp_client_secret": "{{ env `HCP_CLIENT_SECRET` }}"
    "version": "1.0.{{timestamp}}"
    "architecture": "amd64",
  },
  "post-processors": [
    {
      "type": "shell-local",
      "inline": ["echo Doing stuff..."]
    },
    [
      {
        "type": "vagrant",
        "include": ["image.iso"],
        "vagrantfile_template": "vagrantfile.tpl",
        "output": "proxycore_{{.Provider}}_{{.Architecture}}_.box"
      },
      {
        "type": "vagrant-registry",
        "box_tag": "hashicorp/precise64",
        "client_id": "{{user `hcp_client_id`}}",
        "client_secret": "{{user `hcp_client_secret`}}",
        "version": "{{user `version`}}",
        "architecture": "{{user `architecture`}}"
      }
    ]
  ]
}
```

**HCL2**

```hcl
variable "hcp_client_id" {
  type    = string
  default = "${env("HCP_CLIENT_ID")}"
}

variable "hcp_client_secret" {
  type    = string
  default = "${env("HCP_CLIENT_SECRET")}"
}

build {
  sources = ["source.null.autogenerated_1"]

  post-processor "shell-local" {
    inline = ["echo Doing stuff..."]
  }
  post-processors {
    post-processor "vagrant" {
      include              = ["image.iso"]
      output               = "proxycore_{{.Provider}}_{{.Architecture}}_.box"
      vagrantfile_template = "vagrantfile.tpl"
    }
    post-processor "vagrant-registry" {
      client_id     = "${var.hcp_client_id}"
      client_secret = "${var.hcp_client_secret}"
      box_tag       = "hashicorp/precise64"
      version       = "${local.version}"
      architecture  = "${local.architecture}"
    }
  }
}
```

## Use with the Artifice Post-Processor

An example configuration is shown below. Note the use of the nested array that
wraps both the Artifice and Vagrant Registry post-processors within the
post-processor section. Chaining the post-processors together in this way tells
Packer that the artifact produced by the Artifice post-processor should be
passed directly to the Vagrant Registry Post-Processor. It also sets the order in
which the post-processors should run.

Failure to chain the post-processors together in this way will result in the
wrong artifact being supplied to the Vagrant Registry post-processor. This will
likely cause the Vagrant Registry post-processor to error and fail.

Note that the Vagrant box specified in the Artifice post-processor `files` array
must end in the `.box` extension. It must also be the first file in the array.
Additional files bundled by the Artifice post-processor will be ignored.

**JSON**

```json
{
  "variables": {
    "hcp_client_id": "{{ env `HCP_CLIENT_ID` }}",
    "hcp_client_secret": "{{ env `HCP_CLIENT_SECRET` }}"
  },

  "builders": [
    {
      "type": "null",
      "communicator": "none"
    }
  ],

  "post-processors": [
    {
      "type": "shell-local",
      "inline": ["echo Doing stuff..."]
    },
    [
      {
        "type": "artifice",
        "files": ["./path/to/my.box"]
      },
      {
        "type": "vagrant-registry",
        "box_tag": "myorganisation/mybox",
        "client_id": "{{user `hcp_client_id`}}",
        "client_secret": "{{user `hcp_client_secret`}}",
        "version": "0.1.0",
        "architecture": "amd64"
      }
    ]
  ]
}
```

**HCL2**

```hcl
variable "hcp_client_id" {
  type    = string
  default = "${env("HCP_CLIENT_ID")}"
}

variable "hcp_client_secret" {
  type    = string
  default = "${env("HCP_CLIENT_SECRET")}"
}

source "null" "autogenerated_1" {
  communicator = "none"
}

build {
  sources = ["source.null.autogenerated_1"]

  post-processor "shell-local" {
    inline = ["echo Doing stuff..."]
  }
  post-processors {
    post-processor "artifice" {
      files = ["./path/to/my.box"]
    }
    post-processor "vagrant-registry" {
      client_id     = "${var.hcp_client_id}"
      client_secret = "${var.hcp_client_secret}"
      box_tag      = "myorganisation/mybox"
      version      = "0.1.0"
      architecture = "amd64"
    }
  }
}
```
