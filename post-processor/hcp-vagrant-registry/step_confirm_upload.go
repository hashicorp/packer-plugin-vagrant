// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcpvagrantregistry

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/client/registry_service"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

type stepConfirmUpload struct{}

func (s *stepConfirmUpload) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*registry_service.Client)
	config := state.Get("config").(*Config)

	if config.NoDirectUpload {
		return multistep.ActionContinue
	}

	providerName := state.Get("providerName").(string)
	archName := state.Get("architecture").(string)
	object := state.Get("upload-object").(string)

	_, err := client.CompleteDirectUploadBox(
		&registry_service.CompleteDirectUploadBoxParams{
			Context:      ctx,
			Registry:     config.registry,
			Box:          config.box,
			Version:      config.Version,
			Provider:     providerName,
			Architecture: archName,
			Object:       object,
		}, nil,
	)

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if errMsg, ok := errorResponseMsg(err); ok {
		state.Put("error", fmt.Errorf("Failure confirming upload: %s", errMsg))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepConfirmUpload) Cleanup(state multistep.StateBag) {}
