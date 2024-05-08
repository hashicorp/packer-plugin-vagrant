// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcpvagrantregistry

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/client/registry_service"
	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/models"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateVersion struct{}

func (s *stepCreateVersion) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*registry_service.Client)
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(*Config)

	resp, err := client.ReadVersion(
		&registry_service.ReadVersionParams{
			Context:  ctx,
			Registry: config.registry,
			Box:      config.box,
			Version:  config.Version,
		},
	)

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if resp, ok := errorResponse(err); ok {
		// Any code outside of not found should error
		if !resp.IsCode(404) {
			state.Put("error", fmt.Errorf("Failure retrieving version: %s", resp.GetPayload().Message))
			return multistep.ActionHalt
		}
	}

	// If the request was successful, nothing to do
	if resp != nil && resp.IsSuccess() {
		if resp.Payload == nil || resp.Payload.Version == nil {
			state.Put("error", fmt.Errorf("Invalid response body for version read"))
			return multistep.ActionHalt
		}

		state.Put("version", resp.Payload.Version)
		ui.Message("Version exists, skipping creation")
		return multistep.ActionContinue
	}

	vresp, err := client.CreateVersion(
		&registry_service.CreateVersionParams{
			Context:  ctx,
			Registry: config.registry,
			Box:      config.box,
			Data: &models.HashicorpCloudVagrant20220930Version{
				Name:        config.Version,
				Description: config.VersionDescription,
			},
		}, nil,
	)

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if errMsg, ok := errorResponseMsg(err); ok {
		state.Put("error", fmt.Errorf("Failure creating new version: %s", errMsg))
		return multistep.ActionHalt
	}

	if vresp.Payload == nil || vresp.Payload.Version == nil {
		state.Put("error", fmt.Errorf("Invalid response body for version create"))
		return multistep.ActionHalt
	}

	state.Put("version", vresp.Payload.Version)
	ui.Say(fmt.Sprintf("Created new version: %s", config.Version))

	return multistep.ActionContinue
}

func (s *stepCreateVersion) Cleanup(state multistep.StateBag) {}
