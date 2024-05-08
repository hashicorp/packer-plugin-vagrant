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

type stepCreateBox struct{}

func (s *stepCreateBox) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*registry_service.Client)
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(*Config)

	resp, err := client.ReadBox(&registry_service.ReadBoxParams{
		Context:  ctx,
		Box:      config.box,
		Registry: config.registry,
	})

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if resp, ok := errorResponse(err); ok {
		// Any code outside of not found should be an error
		if !resp.IsCode(404) {
			state.Put("error", fmt.Errorf("Failure retrieving box: %s", resp.GetPayload().Message))
			return multistep.ActionHalt
		}
	}

	// If the request was successful, nothing to do
	if resp != nil && resp.IsSuccess() {
		ui.Say(fmt.Sprintf("Found box and verified accessible: %s", config.Tag))
		return multistep.ActionContinue
	}

	// Create the box
	_, err = client.CreateBox(
		&registry_service.CreateBoxParams{
			Context:  ctx,
			Registry: config.registry,
			Data: &models.HashicorpCloudVagrant20220930Box{
				Name:        config.box,
				Description: config.BoxDescription,
				IsPrivate:   config.BoxPrivate,
			},
		}, nil,
	)

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if errMsg, ok := errorResponseMsg(err); ok {
		state.Put("error", fmt.Errorf("Failure creating new box: %s", errMsg))
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Created new box: %s", config.Tag))

	return multistep.ActionContinue
}

func (s *stepCreateBox) Cleanup(state multistep.StateBag) {}
