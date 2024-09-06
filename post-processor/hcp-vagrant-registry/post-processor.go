// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package hcpvagrantregistry

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcp-sdk-go/clients/cloud-operation/stable/2020-05-05/client/operation_service"
	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/client"
	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/client/registry_service"
	hcpconfig "github.com/hashicorp/hcp-sdk-go/config"
	"github.com/hashicorp/hcp-sdk-go/httpclient"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

var builtins = map[string]string{
	"mitchellh.post-processor.vagrant": "vagrant",
	"packer.post-processor.artifice":   "artifice",
	"vagrant":                          "vagrant",
}

const HCP_API_ADDRESS = "api.cloud.hashicorp.com:443"

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	Tag                 string `mapstructure:"box_tag"`
	BoxDescription      string `mapstructure:"box_description"`
	BoxPrivate          bool   `mapstructure:"box_private"`
	Version             string `mapstructure:"version"`
	VersionDescription  string `mapstructure:"version_description"`
	NoRelease           bool   `mapstructure:"no_release"`
	Architecture        string `mapstructure:"architecture"`
	DefaultArchitecture string `mapstructure:"default_architecture"`

	ClientID       string `mapstructure:"client_id"`
	ClientSecret   string `mapstructure:"client_secret"`
	BoxDownloadUrl string `mapstructure:"box_download_url"`
	NoDirectUpload bool   `mapstructure:"no_direct_upload"`
	BoxChecksum    string `mapstructure:"box_checksum"`

	// NOTE: These are used for development
	HcpApiAddress         string `mapstructure:"hcp_api_address"`
	HcpAuthUrl            string `mapstructure:"hcp_auth_url"`
	InsecureSkipTLSVerify bool   `mapstructure:"insecure_skip_tls_verify"`

	registry     string
	box          string
	checksum     string
	checksumType string
	ctx          interpolate.Context
}

type PostProcessor struct {
	config                Config
	client                *client.CloudVagrantBoxRegistry
	runner                multistep.Runner
	insecureSkipTLSVerify bool
}

func (p *PostProcessor) ConfigSpec() hcldec.ObjectSpec {
	return p.config.FlatMapstructure().HCL2Spec()
}

func (p *PostProcessor) Configure(raws ...interface{}) error {
	err := config.Decode(&p.config, &config.DecodeOpts{
		PluginType:         BuilderId,
		Interpolate:        true,
		InterpolateContext: &p.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"box_download_url",
			},
		},
	}, raws...)
	if err != nil {
		return err
	}

	errs := new(packer.MultiError)

	// Set API address if none provided
	if p.config.HcpApiAddress == "" {
		p.config.HcpApiAddress = os.Getenv("HCP_API_ADDRESS")
	}

	if p.config.HcpApiAddress == "" {
		p.config.HcpApiAddress = HCP_API_ADDRESS
	}

	// Allow disabling tls verification if not connecting to hcp
	if p.config.InsecureSkipTLSVerify {
		if p.config.HcpApiAddress == HCP_API_ADDRESS {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("insecure_skip_tls_verify cannot be enabled for HCP"),
			)
		} else {
			p.insecureSkipTLSVerify = true
		}
	}

	// Attempt to get credentials from environment variables if unset
	if p.config.ClientID == "" {
		p.config.ClientID = os.Getenv("HCP_CLIENT_ID")
	}

	if p.config.ClientSecret == "" {
		p.config.ClientSecret = os.Getenv("HCP_CLIENT_SECRET")
	}

	// Required configuration
	templates := map[string]*string{
		"box_tag": &p.config.Tag,
		"version": &p.config.Version,
	}

	for key, ptr := range templates {
		if *ptr == "" {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("%s must be set", key),
			)
		}
	}

	// Validate format of the box tag provided
	parts := strings.SplitN(p.config.Tag, "/", 2)
	if len(parts) != 2 {
		errs = packer.MultiErrorAppend(errs,
			fmt.Errorf("box_tag must include registry and box name"),
		)
	} else {
		p.config.registry = parts[0]
		p.config.box = parts[1]
	}

	// If checksum is provided, validate the format
	if p.config.BoxChecksum != "" {
		parts = strings.SplitN(p.config.BoxChecksum, ":", 2)
		if len(parts) != 2 {
			errs = packer.MultiErrorAppend(errs,
				fmt.Errorf("box_checksum format invalid (format: CHECKSUM_TYPE:CHECKSUM_VALUE)"),
			)
		} else {
			p.config.checksumType = parts[0]
			p.config.checksum = parts[1]
		}
	}

	// Required configuration when api address is not custom
	if p.config.HcpApiAddress == HCP_API_ADDRESS {
		templates = map[string]*string{
			"client_id":     &p.config.ClientID,
			"client_secret": &p.config.ClientSecret,
		}

		for key, ptr := range templates {
			if *ptr == "" {
				errs = packer.MultiErrorAppend(
					errs, fmt.Errorf("%s must be set", key),
				)
			}
		}
	}

	opts := []hcpconfig.HCPConfigOption{
		hcpconfig.FromEnv(),
		hcpconfig.WithClientCredentials(p.config.ClientID, p.config.ClientSecret),
		hcpconfig.WithoutBrowserLogin(),
	}

	if p.config.HcpApiAddress != HCP_API_ADDRESS {
		opts = append(opts, hcpconfig.WithAPI(p.config.HcpApiAddress, &tls.Config{
			InsecureSkipVerify: p.insecureSkipTLSVerify,
		}))
	}

	if p.config.HcpAuthUrl != "" {
		opts = append(opts, hcpconfig.WithAuth(p.config.HcpAuthUrl, &tls.Config{
			InsecureSkipVerify: p.insecureSkipTLSVerify,
		}))
	}

	// Do all the hcp setup
	hcpConfig, err := hcpconfig.NewHCPConfig(opts...)

	if err != nil {
		return packer.MultiErrorAppend(errs, err)
	}

	sdkClient, err := httpclient.New(httpclient.Config{
		HCPConfig: hcpConfig,
	})

	if err != nil {
		return packer.MultiErrorAppend(errs, err)
	}

	// Create the base client
	p.client = client.New(sdkClient, nil)

	if len(errs.Errors) > 0 {
		return errs
	}

	return nil
}

func (p *PostProcessor) PostProcess(ctx context.Context, ui packer.Ui, artifact packer.Artifact) (packer.Artifact, bool, bool, error) {
	if _, ok := builtins[artifact.BuilderId()]; !ok {
		return nil, false, false, fmt.Errorf(
			"Unknown artifact type: this post-processor requires an input artifact from the artifice post-processor, vagrant post-processor, or vagrant builder: %s", artifact.BuilderId())
	}

	if len(artifact.Files()) == 0 {
		return nil, false, false, fmt.Errorf("No files provided in artifact for upload")
	} else if !strings.HasSuffix(artifact.Files()[0], ".box") {
		return nil, false, false, fmt.Errorf(
			"Unknown file in artifact, Vagrant box with .box suffix is required as first artifact file: %s", artifact.Files())
	}

	var boxMetadata map[string]interface{}
	var err error

	// Get the architecture
	archName := p.config.Architecture
	if archName == "" {
		if boxMetadata, err = metadataFromVagrantBox(artifact.Files()[0]); err != nil {
			return nil, false, false, err
		}
		if archName, err = getArchitecture(boxMetadata); err != nil {
			return nil, false, false, err
		}
	}

	providerName, err := getProvider(artifact.Id(), artifact.Files()[0], builtins[artifact.BuilderId()], boxMetadata)
	if err != nil {
		return nil, false, false, fmt.Errorf("error getting provider name: %s", err)
	}

	var generatedData map[interface{}]interface{}
	stateData := artifact.State("generated_data")
	if stateData != nil {
		// Make sure it's not a nil map so we can assign to it later.
		generatedData = stateData.(map[interface{}]interface{})
	}
	// If stateData has a nil map generatedData will be nil
	// and we need to make sure it's not
	if generatedData == nil {
		generatedData = make(map[interface{}]interface{})
	}
	generatedData["ArtifactId"] = artifact.Id()
	generatedData["Provider"] = providerName
	generatedData["Architecture"] = archName
	p.config.ctx.Data = generatedData

	boxDownloadUrl, err := interpolate.Render(p.config.BoxDownloadUrl, &p.config.ctx)
	if err != nil {
		return nil, false, false, fmt.Errorf("Failed processing box_download_url: %s", err)
	}

	if p.config.BoxChecksum != "" {
		if checksumParts := strings.SplitN(p.config.BoxChecksum, ":", 2); len(checksumParts) != 2 {
			return nil, false, false, fmt.Errorf("box checksum must be specified as `$type:$digest`")
		}
	}

	// Set up the state
	state := new(multistep.BasicStateBag)
	state.Put("config", &p.config)
	state.Put("client", registry_service.New(p.client.Transport, nil))
	state.Put("operation-client", operation_service.New(p.client.Transport, nil))
	state.Put("artifact", artifact)
	state.Put("artifactFilePath", artifact.Files()[0])
	state.Put("ui", ui)
	state.Put("providerName", providerName)
	state.Put("downloadUrl", boxDownloadUrl)
	state.Put("architecture", archName)

	// Build the steps
	steps := []multistep.Step{
		new(stepCreateBox),
		new(stepCreateVersion),
		new(stepCreateProvider),
		new(stepCreateArchitecture),
	}
	if p.config.BoxDownloadUrl == "" {
		steps = append(steps,
			new(stepPrepareUpload),
			new(stepUpload),
			new(stepConfirmUpload))
	}
	steps = append(steps, new(stepReleaseVersion))

	// Run the steps
	p.runner = commonsteps.NewRunner(steps, p.config.PackerConfig, ui)
	p.runner.Run(ctx, state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, false, false, rawErr.(error)
	}

	return NewArtifact(providerName, p.config.Tag), true, false, nil
}

func getArchitecture(metadata map[string]interface{}) (architectureName string, err error) {
	if arch, ok := metadata["architecture"]; ok {
		if architectureName, ok = arch.(string); ok && architectureName != "" {
			return
		}
	}

	return "", fmt.Errorf("Error: Could not determine architecture from box metadata.json file")
}

func getProvider(builderName, boxfile, builderId string, metadata map[string]interface{}) (string, error) {
	if builderId == "artifice" {
		// The artifice post processor cannot embed any data in the
		// supplied artifact so the provider information must be extracted
		// from the box file directly
		return providerFromVagrantBox(boxfile, metadata)
	}
	// For the Vagrant builder and Vagrant post processor the provider can
	// be determined from information embedded in the artifact
	return providerFromBuilderName(builderName), nil
}

// Converts a packer builder name to the corresponding vagrant provider
func providerFromBuilderName(name string) string {
	switch name {
	case "aws":
		return "aws"
	case "scaleway":
		return "scaleway"
	case "digitalocean":
		return "digitalocean"
	case "virtualbox":
		return "virtualbox"
	case "vmware":
		return "vmware_desktop"
	case "parallels":
		return "parallels"
	default:
		return name
	}
}

// Returns the Vagrant provider the box is intended for use with by
// reading the metadata file packaged inside the box
func providerFromVagrantBox(boxfile string, metadata map[string]interface{}) (providerName string, err error) {
	if len(metadata) == 0 {
		if metadata, err = metadataFromVagrantBox(boxfile); err != nil {
			return
		}
	}

	if prov, ok := metadata["provider"]; ok {
		if providerName, ok = prov.(string); ok && providerName != "" {
			return
		}
	}

	return "", fmt.Errorf("Could not determine provider from box metadata.json file")
}

// Returns the metadata found within the metadata file
// packaged inside the box
func metadataFromVagrantBox(boxfile string) (metadata map[string]interface{}, err error) {
	log.Printf("Attempting to extract metadata in box file. This may take some time...")

	f, err := os.Open(boxfile)
	if err != nil {
		return nil, fmt.Errorf("Failed to open box file: %w", err)
	}
	defer f.Close()

	// Vagrant boxes are gzipped tar archives
	ar, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("Failed unpacking box archive: %w", err)
	}
	tr := tar.NewReader(ar)

	for {
		var hdr *tar.Header
		hdr, err = tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("metadata.json file not found in box: %s", boxfile)
		}

		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("Failed reading header info from box tar archive: %w", err)
		}

		if hdr.Name == "metadata.json" {
			var contents []byte
			contents, err = io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("Failed reading contents of metadata.json file from box file: %w", err)
			}
			err = json.Unmarshal(contents, &metadata)
			if err != nil {
				return nil, fmt.Errorf("Failed parsing metadata.json file: %w", err)
			}

			return
		}
	}
}
