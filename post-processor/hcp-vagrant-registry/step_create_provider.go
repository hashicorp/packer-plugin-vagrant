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

type stepCreateProvider struct{}

func (s *stepCreateProvider) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*registry_service.Client)
	ui := state.Get("ui").(packer.Ui)
	providerName := state.Get("providerName").(string)
	config := state.Get("config").(*Config)

	resp, err := client.ReadProvider(
		&registry_service.ReadProviderParams{
			Context:  ctx,
			Registry: config.registry,
			Box:      config.box,
			Version:  config.Version,
			Provider: providerName,
		},
	)

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if resp, ok := errorResponse(err); ok {
		// Any code outside of not found should be an error
		if !resp.IsCode(404) {
			state.Put("error", fmt.Errorf("Failure retrieving provider: %s", resp.GetPayload().Message))
			return multistep.ActionHalt
		}
	}

	if resp != nil && resp.IsSuccess() {
		ui.Message("Provider exists, skipping creation")
		return multistep.ActionContinue
	}

	_, err = client.CreateProvider(
		&registry_service.CreateProviderParams{
			Context:  ctx,
			Registry: config.registry,
			Box:      config.box,
			Version:  config.Version,
			Data: &models.HashicorpCloudVagrant20220930Provider{
				Name: providerName,
			},
		}, nil,
	)

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if errMsg, ok := errorResponseMsg(err); ok {
		state.Put("error", fmt.Errorf("Failure creating new provider: %s", errMsg))
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Created new provider: %s", providerName))

	return multistep.ActionContinue
}

func (s *stepCreateProvider) Cleanup(state multistep.StateBag) {}
