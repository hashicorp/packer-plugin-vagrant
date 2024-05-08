// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcpvagrantregistry

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/client/registry_service"
	"github.com/hashicorp/hcp-sdk-go/clients/cloud-vagrant-box-registry/stable/2022-09-30/models"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

type stepCreateArchitecture struct{}

func (s *stepCreateArchitecture) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*registry_service.Client)
	providerName := state.Get("providerName").(string)
	downloadUrl := state.Get("downloadUrl").(string)
	config := state.Get("config").(*Config)
	archName := state.Get("architecture").(string)

	resp, err := client.ReadArchitecture(
		&registry_service.ReadArchitectureParams{
			Context:      ctx,
			Registry:     config.registry,
			Box:          config.box,
			Version:      config.Version,
			Provider:     providerName,
			Architecture: archName,
		},
	)

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if resp, ok := errorResponse(err); ok {
		// Any code outside of not found should be an error
		if !resp.IsCode(404) {
			state.Put("error", fmt.Errorf("Failure retrieving architecture: %s", resp.GetPayload().Message))
			return multistep.ActionHalt
		}
	}

	data := &models.HashicorpCloudVagrant20220930BoxData{}

	if downloadUrl != "" {
		data.DownloadURL = downloadUrl
	}

	if config.BoxChecksum != "" {
		data.Checksum = config.checksum
		data.ChecksumType = models.NewHashicorpCloudVagrant20220930ChecksumType(
			models.HashicorpCloudVagrant20220930ChecksumType(config.checksumType),
		)
	} else {
		data.ChecksumType = models.HashicorpCloudVagrant20220930ChecksumTypeNONE.Pointer()
	}

	// If the architecture already exists, update it
	if resp != nil && resp.IsSuccess() {
		_, err := client.UpdateArchitecture(
			&registry_service.UpdateArchitectureParams{
				Context:      ctx,
				Registry:     config.registry,
				Box:          config.box,
				Version:      config.Version,
				Provider:     providerName,
				Architecture: archName,
				Data: &models.HashicorpCloudVagrant20220930Architecture{
					BoxData: data,
				},
			}, nil,
		)

		if isErrorUnexpected(err, state) {
			return multistep.ActionHalt
		}

		if errMsg, ok := errorResponseMsg(err); ok {
			state.Put("error", fmt.Errorf("Failure updating existing architecture: %s", errMsg))
			return multistep.ActionHalt
		}

		return multistep.ActionContinue
	}

	_, err = client.CreateArchitecture(
		&registry_service.CreateArchitectureParams{
			Context:  ctx,
			Registry: config.registry,
			Box:      config.box,
			Version:  config.Version,
			Provider: providerName,
			Data: &models.HashicorpCloudVagrant20220930Architecture{
				ArchitectureType: archName,
				Default:          archName == config.DefaultArchitecture,
				BoxData:          data,
			},
		}, nil,
	)

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if errMsg, ok := errorResponseMsg(err); ok {
		state.Put("error", fmt.Errorf("Failure creating new architecture: %s", errMsg))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepCreateArchitecture) Cleanup(state multistep.StateBag) {}
