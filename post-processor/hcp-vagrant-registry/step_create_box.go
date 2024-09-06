// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcpvagrantregistry

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcp-sdk-go/clients/cloud-operation/stable/2020-05-05/client/operation_service"
	shared_models "github.com/hashicorp/hcp-sdk-go/clients/cloud-shared/v1/models"
	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/client/registry_service"
	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/models"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateBox struct{}

var BOX_CREATE_TIMEOUT = "60s"

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
	cresp, err := client.CreateBox(
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
		state.Put("error", fmt.Errorf("Failure creating new box: %s - Please try again.", errMsg))
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Created new box: %s", config.Tag))
	if cresp.Payload == nil || cresp.Payload.Operation == nil {
		state.Put("error", fmt.Errorf("Unable to wait for box to become available - Please check HCP Vagrant for box status, and try again."))
		return multistep.ActionHalt
	}

	ui.Say("Waiting for box to become available...")
	op := cresp.Payload.Operation

	operationClient := state.Get("operation-client").(operation_service.ClientService)
	waitReq := &operation_service.WaitParams{
		ID:                     op.ID,
		LocationOrganizationID: op.Location.OrganizationID,
		LocationProjectID:      op.Location.ProjectID,
		Timeout:                &BOX_CREATE_TIMEOUT,
		Context:                ctx,
	}

	wresp, err := operationClient.Wait(waitReq, nil)
	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if errMsg, ok := errorResponseMsg(err); ok {
		state.Put("error", fmt.Errorf("Unexpected failure waiting for box to become available: %s - Please try again.", errMsg))
		return multistep.ActionHalt
	}

	if wresp.Payload == nil || wresp.Payload.Operation == nil {
		state.Put("error", fmt.Errorf("Unable to check box creation operation status - Please check HCP Vagrant for box status, and try again."))
		return multistep.ActionHalt
	}

	operation := wresp.Payload.Operation
	if operation.Error != nil {
		state.Put("error", fmt.Errorf("Box creation operation reported a failure: %s - Please try again.", operation.Error.Message))
		return multistep.ActionHalt
	}

	if operation.State == nil || *operation.State != shared_models.HashicorpCloudOperationOperationStateDONE {
		state.Put("error", fmt.Errorf("Timeout exceeded waiting for box to become available - Please verify box creation in HCP Vagrant and try again."))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepCreateBox) Cleanup(state multistep.StateBag) {}
