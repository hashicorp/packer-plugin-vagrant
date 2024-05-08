// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcpvagrantregistry

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/net"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/retry"
)

type stepUpload struct{}

func (s *stepUpload) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	url := state.Get("upload-url").(string)
	artifactFilePath := state.Get("artifactFilePath").(string)

	client := net.HttpClientWithEnvironmentProxy()

	// Stash the http client we built so it can
	// be used in the confrim step if needed.
	state.Put("http-client", client)

	ui.Say(fmt.Sprintf("Uploading box: %s", artifactFilePath))
	ui.Message(
		"Depending on your internet connection and the size of the box,\n" +
			"this may take some time")

	err := retry.Config{
		Tries:      3,
		RetryDelay: (&retry.Backoff{InitialBackoff: 10 * time.Second, MaxBackoff: 10 * time.Second, Multiplier: 2}).Linear,
	}.Run(ctx, func(ctx context.Context) (err error) {
		ui.Message("Uploading box")

		defer func() {
			if err != nil {
				ui.Message(fmt.Sprintf(
					"Error uploading box! Will retry in 10 seconds. Error: %s", err))
			}
		}()

		file, err := os.Open(artifactFilePath)
		if err != nil {
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			return
		}

		request, err := http.NewRequest("PUT", url, file)
		if err != nil {
			return
		}

		request.ContentLength = info.Size()
		resp, err := client.Do(request)
		if err != nil {
			return
		}

		if resp.StatusCode != 200 {
			err = fmt.Errorf("bad HTTP status: %d", resp.StatusCode)
			return
		}

		return
	})

	if err != nil {
		state.Put("error", fmt.Errorf("Failed to upload box asset: %s", err))
		return multistep.ActionHalt
	}

	ui.Message("Box successfully uploaded")

	return multistep.ActionContinue
}

func (s *stepUpload) Cleanup(state multistep.StateBag) {}
