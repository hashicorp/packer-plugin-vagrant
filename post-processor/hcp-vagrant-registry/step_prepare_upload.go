// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcpvagrantregistry

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/client/registry_service"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

const HCP_VAGRANT_REGISTRY_DIRECT_UPLOAD_LIMIT = 5368709120 // Upload limit is 5G

type stepPrepareUpload struct{}

func (s *stepPrepareUpload) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*registry_service.Client)
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	providerName := state.Get("providerName").(string)
	archName := state.Get("architecture").(string)
	artifactFilePath := state.Get("artifactFilePath").(string)

	// If direct upload is enabled, the asset size must be <= 5 GB
	if config.NoDirectUpload == false {
		f, err := os.Stat(artifactFilePath)
		if err != nil {
			ui.Error(fmt.Sprintf("failed determining size of upload artifact: %s", artifactFilePath))
		}
		if f.Size() > HCP_VAGRANT_REGISTRY_DIRECT_UPLOAD_LIMIT {
			ui.Say(fmt.Sprintf("Asset %s is larger than the direct upload limit. Setting `NoDirectUpload` to true", artifactFilePath))
			config.NoDirectUpload = true
		}
	}

	ui.Say(fmt.Sprintf("Preparing upload of box: %s", artifactFilePath))

	if config.NoDirectUpload {
		resp, err := client.UploadBox(
			&registry_service.UploadBoxParams{
				Context:      ctx,
				Registry:     config.registry,
				Box:          config.box,
				Version:      config.Version,
				Provider:     providerName,
				Architecture: archName,
			}, nil,
		)

		if isErrorUnexpected(err, state) {
			return multistep.ActionHalt
		}

		if errMsg, ok := errorResponseMsg(err); ok {
			state.Put("error", fmt.Errorf("Failure preparing upload: %s", errMsg))
			return multistep.ActionHalt
		}

		state.Put("upload-url", resp.Payload.URL)
	} else {
		resp, err := client.DirectUploadBox(
			&registry_service.DirectUploadBoxParams{
				Context:      ctx,
				Registry:     config.registry,
				Box:          config.box,
				Version:      config.Version,
				Provider:     providerName,
				Architecture: archName,
			}, nil,
		)

		if isErrorUnexpected(err, state) {
			return multistep.ActionHalt
		}

		if errMsg, ok := errorResponseMsg(err); ok {
			state.Put("error", fmt.Errorf("Failure preparing upload: %s", errMsg))
			return multistep.ActionHalt
		}

		state.Put("upload-url", resp.Payload.URL)
		state.Put("upload-object", resp.Payload.Object)
	}

	return multistep.ActionContinue
}

func (s *stepPrepareUpload) Cleanup(state multistep.StateBag) {}
