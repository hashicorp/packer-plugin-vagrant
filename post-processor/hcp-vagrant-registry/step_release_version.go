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

type stepReleaseVersion struct{}

func (s *stepReleaseVersion) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*registry_service.Client)
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(*Config)
	version := state.Get("version").(*models.HashicorpCloudVagrant20220930Version)

	if config.NoRelease {
		ui.Message("Not releasing version due to configuration")
		return multistep.ActionContinue
	}

	if version.State != nil && *version.State != models.HashicorpCloudVagrant20220930VersionStateUNRELEASED {
		ui.Message("Version not in unreleased state, skipping release")
		return multistep.ActionContinue
	}

	ui.Say(fmt.Sprintf("Releasing version: %s", config.Version))

	_, err := client.ReleaseVersion(
		&registry_service.ReleaseVersionParams{
			Context:  ctx,
			Registry: config.registry,
			Box:      config.box,
			Version:  config.Version,
		}, nil,
	)

	if isErrorUnexpected(err, state) {
		return multistep.ActionHalt
	}

	if errMsg, ok := errorResponseMsg(err); ok {
		state.Put("error", fmt.Errorf("Failure releasing version: %s", errMsg))
		return multistep.ActionHalt
	}

	ui.Message("Version successfully released and available")
	return multistep.ActionContinue
}

func (s *stepReleaseVersion) Cleanup(state multistep.StateBag) {}
