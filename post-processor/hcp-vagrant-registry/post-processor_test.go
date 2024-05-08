// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcpvagrantregistry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/stretchr/testify/require"
)

type stubResponse struct {
	Path       string
	Method     string
	Response   string
	StatusCode int
}

type tarFiles []struct {
	Name, Body string
}

func TestPostProcessor(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()

	t.Run("Configure", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			testCases := []struct {
				desc  string
				setup func(map[string]interface{})
			}{
				{
					desc: "minimal",
				},
				{
					desc: "with checksum",
					setup: func(c map[string]interface{}) {
						c["box_checksum"] = "sha256:testchecksumvalue"
					},
				},
				{
					desc: "with custom address and no credentials",
					setup: func(c map[string]interface{}) {
						delete(c, "client_id")
						delete(c, "client_secret")
						c["hcp_api_address"] = "localhost"
					},
				},
				{
					desc: "with custom address and skip tls verify",
					setup: func(c map[string]interface{}) {
						c["hcp_api_address"] = "localhost"
						c["insecure_skip_tls_verify"] = true
					},
				},
			}

			for _, tc := range testCases {
				t.Run(tc.desc, func(t *testing.T) {
					var p PostProcessor
					config := testMinimalConfig()
					if tc.setup != nil {
						tc.setup(config)
					}

					require.NoError(p.Configure(config))
				})
			}
		})

		t.Run("invalid", func(t *testing.T) {
			testCases := []struct {
				desc    string
				setup   func(map[string]interface{})
				wantErr string
			}{
				{
					desc: "bad tag",
					setup: func(c map[string]interface{}) {
						c["box_tag"] = "box-name"
					},
					wantErr: "box_tag must include registry and box name",
				},
				{
					desc: "missing tag",
					setup: func(c map[string]interface{}) {
						delete(c, "box_tag")
					},
					wantErr: "box_tag must be set",
				},
				{
					desc: "missing version",
					setup: func(c map[string]interface{}) {
						delete(c, "version")
					},
					wantErr: "version must be set",
				},
				{
					desc: "missing client id",
					setup: func(c map[string]interface{}) {
						delete(c, "client_id")
					},
					wantErr: "client_id must be set",
				},
				{
					desc: "missing client secret",
					setup: func(c map[string]interface{}) {
						delete(c, "client_secret")
					},
					wantErr: "client_secret must be set",
				},
				{
					desc: "checksum format",
					setup: func(c map[string]interface{}) {
						c["box_checksum"] = "testchecksumvalue"
					},
					wantErr: "box_checksum format invalid",
				},
				{
					desc: "skip tls verify",
					setup: func(c map[string]interface{}) {
						c["insecure_skip_tls_verify"] = true
					},
					wantErr: "insecure_skip_tls_verify cannot be enabled for HCP",
				},
			}

			for _, tc := range testCases {
				t.Run(tc.desc, func(t *testing.T) {
					var p PostProcessor
					config := testMinimalConfig()
					if tc.setup != nil {
						tc.setup(config)
					}

					require.ErrorContains(p.Configure(config), tc.wantErr)
				})
			}
		})
	})

	t.Run("PostProcess", func(t *testing.T) {
		uploadServer := newDummyServer()
		uploadURL := fmt.Sprintf("http://%s/do-upload", uploadServer.Listener.Addr())
		uploadObject := "TEST_OBJECT_ID"
		uploadCallbackURL := fmt.Sprintf("http://%s/do-upload-callback", uploadServer.Listener.Addr())

		testCases := []struct {
			desc    string
			files   tarFiles
			stack   []stubResponse
			setup   func(config map[string]interface{}, artifact *packer.MockArtifact)
			wantErr string
		}{
			{
				desc: "Invalid - missing architecture",
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox"}`},
				},
				wantErr: "not determine architecture",
			},
			{
				desc: "OK - architecture config only",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						Response:   `{"version": {"state": "UNRELEASED"}}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/upload",
						Response:   fmt.Sprintf(`{"url": "%s"}`, uploadURL),
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/release",
						Response:   `{}`,
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox"}`},
				},
				setup: func(c map[string]interface{}, _ *packer.MockArtifact) {
					c["architecture"] = "amd64"
				},
			},
			{
				desc: "OK - architecture metadata",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						Response:   `{"version": {"state": "UNRELEASED"}}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/upload",
						Response:   fmt.Sprintf(`{"url": "%s"}`, uploadURL),
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/release",
						Response:   `{}`,
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox", "architecture": "amd64"}`},
				},
			},
			{
				desc: "OK - architecture metadata and config",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						Response:   `{"version": {"state": "UNRELEASED"}}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/upload",
						Response:   fmt.Sprintf(`{"url": "%s"}`, uploadURL),
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/release",
						Response:   `{}`,
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox", "architecture": "arm"}`},
				},
				setup: func(c map[string]interface{}, _ *packer.MockArtifact) {
					c["architecture"] = "amd64"
				},
			},
			{
				desc: "Invalid - bad version read response",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						Response:   `{}`,
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox", "architecture": "amd64"}`},
				},
				wantErr: "Invalid response body",
			},
			{
				desc: "OK - creates box when missing",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						StatusCode: 404,
					},
					{
						Method:     "POST",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/boxes",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						StatusCode: 404,
					},
					{
						Method:     "POST",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/versions",
						Response:   `{"version": {"state": "UNRELEASED"}}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/upload",
						Response:   fmt.Sprintf(`{"url": "%s"}`, uploadURL),
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/release",
						Response:   `{}`,
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox", "architecture": "amd64"}`},
				},
			},
			{
				desc: "OK - creates version when missing",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						StatusCode: 404,
					},
					{
						Method:     "POST",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/versions",
						Response:   `{"version": {"state": "UNRELEASED"}}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/upload",
						Response:   fmt.Sprintf(`{"url": "%s"}`, uploadURL),
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/release",
						Response:   `{}`,
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox", "architecture": "amd64"}`},
				},
			},
			{
				desc: "OK - creates provider when missing",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						Response:   `{"version": {"state": "UNRELEASED"}}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox",
						StatusCode: 404,
					},
					{
						Method:     "POST",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/providers",
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/upload",
						Response:   fmt.Sprintf(`{"url": "%s"}`, uploadURL),
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/release",
						Response:   `{}`,
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox", "architecture": "amd64"}`},
				},
			},
			{
				desc: "OK - creates architecture when missing",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						Response:   `{"version": {"state": "UNRELEASED"}}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						StatusCode: 404,
					},
					{
						Method:     "POST",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architectures",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/upload",
						Response:   fmt.Sprintf(`{"url": "%s"}`, uploadURL),
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/release",
						Response:   `{}`,
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox", "architecture": "amd64"}`},
				},
			},
			{
				desc: "OK - does not release version when already released",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						Response:   `{"version": {"state": "RELEASED"}}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						StatusCode: 404,
					},
					{
						Method:     "POST",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architectures",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/upload",
						Response:   fmt.Sprintf(`{"url": "%s"}`, uploadURL),
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox", "architecture": "amd64"}`},
				},
			},
			{
				desc: "OK - direct upload",
				stack: []stubResponse{
					{
						Method:     "POST",
						Path:       "/oauth2/token",
						Response:   `{"access_token": "TEST_TOKEN", "expiry": 0}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5",
						Response:   `{"version": {"state": "RELEASED"}}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64",
						Response:   `{}`,
						StatusCode: 200,
					},
					{
						Method:     "GET",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/direct/upload",
						Response:   fmt.Sprintf(`{"url": "%s", "callback": "%s", "object": "%s"}`, uploadURL, uploadCallbackURL, uploadObject),
						StatusCode: 200,
					},
					{
						Method:     "PUT",
						Path:       "/vagrant/2022-09-30/registry/hashicorp/box/precise64/version/0.5/provider/virtualbox/architecture/amd64/direct/complete/TEST_OBJECT_ID",
						Response:   `{}`,
						StatusCode: 200,
					},
				},
				files: tarFiles{
					{"foo.txt", "This is a foo file"},
					{"bar.txt", "This is a bar file"},
					{"metadata.json", `{"provider": "virtualbox", "architecture": "amd64"}`},
				},
				setup: func(c map[string]interface{}, _ *packer.MockArtifact) {
					c["no_direct_upload"] = false
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				boxfile, err := createBox(tc.files)
				require.NoError(err, "failed to create test box")
				defer os.Remove(boxfile.Name())

				artifact := &packer.MockArtifact{
					BuilderIdValue: "mitchellh.post-processor.vagrant",
					FilesValue:     []string{boxfile.Name()},
					IdValue:        "virtualbox",
				}
				server := newStackServer(tc.stack)
				defer server.Close()

				config := testMinimalConfigAddr(server.Listener.Addr().String())
				config["no_direct_upload"] = true

				if tc.setup != nil {
					tc.setup(config, artifact)
				}

				var p PostProcessor
				require.NoError(p.Configure(config), "failed to configure post processor")

				_, _, _, err = p.PostProcess(ctx, testUi(), artifact)
				if tc.wantErr != "" {
					require.Error(err)
					require.ErrorContains(err, tc.wantErr)
				} else {
					require.NoError(err)
				}
			})
		}
	})
}

func newBoxFile() (boxfile *os.File, err error) {
	boxfile, err = os.CreateTemp(os.TempDir(), "test*.box")
	if err != nil {
		return boxfile, fmt.Errorf("Error creating test box file: %s", err)
	}
	return boxfile, nil
}

func createBox(files tarFiles) (boxfile *os.File, err error) {
	boxfile, err = newBoxFile()
	if err != nil {
		return boxfile, err
	}

	// Box files are gzipped tar archives
	aw := gzip.NewWriter(boxfile)
	tw := tar.NewWriter(aw)

	// Add each file to the box
	for _, file := range files {
		// Create and write the tar file header
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0644,
			Size: int64(len(file.Body)),
		}
		err = tw.WriteHeader(hdr)
		if err != nil {
			return boxfile, fmt.Errorf("Error writing box tar file header: %s", err)
		}
		// Write the file contents
		_, err = tw.Write([]byte(file.Body))
		if err != nil {
			return boxfile, fmt.Errorf("Error writing box tar file contents: %s", err)
		}
	}
	// Flush and close each writer
	err = tw.Close()
	if err != nil {
		return boxfile, fmt.Errorf("Error flushing tar file contents: %s", err)
	}
	err = aw.Close()
	if err != nil {
		return boxfile, fmt.Errorf("Error flushing gzip file contents: %s", err)
	}

	return boxfile, nil
}

func testUi() *packer.BasicUi {
	return &packer.BasicUi{
		Reader: new(bytes.Buffer),
		Writer: new(bytes.Buffer),
	}
}

func testMinimalConfig() map[string]interface{} {
	return map[string]interface{}{
		"box_tag":       "hashicorp/precise64",
		"version":       "0.5",
		"client_id":     "TEST-CLIENT-ID",
		"client_secret": "TEST-CLIENT-SECRET",
	}
}

func testMinimalConfigAddr(addr string) map[string]interface{} {
	c := testMinimalConfig()
	c["hcp_api_address"] = addr
	c["hcp_auth_url"] = fmt.Sprintf("https://%s", addr)
	c["insecure_skip_tls_verify"] = true

	return c
}

func newStackServer(stack []stubResponse) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if len(stack) < 1 {
			fmt.Printf("Request stack is empty - unhandled %s %s\n", req.Method, req.URL.Path)
			rw.Header().Add("Error", fmt.Sprintf("Request stack is empty - Method: %s Path: %s", req.Method, req.URL.Path))
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		match := stack[0]
		stack = stack[1:]
		if match.Method != "" && req.Method != match.Method {
			fmt.Printf("Request not matched - request[%q %s] != match[%q %s]\n", req.URL.Path, req.Method, match.Path, match.Method)
			rw.Header().Add("Error", fmt.Sprintf("Request %s != %s", match.Method, req.Method))
			http.Error(rw, fmt.Sprintf("Request %s != %s", match.Method, req.Method), http.StatusInternalServerError)
			return
		}
		if match.Path != "" && match.Path != req.URL.Path {
			fmt.Printf("Request not matched - request[%q %s] != match[%q %s]\n", req.URL.Path, req.Method, match.Path, match.Method)
			rw.Header().Add("Error", fmt.Sprintf("Request %s != %s", match.Path, req.URL.Path))
			http.Error(rw, fmt.Sprintf("Request %s != %s", match.Path, req.URL.Path), http.StatusInternalServerError)
			return
		}
		rw.Header().Add("Complete", fmt.Sprintf("Method: %s Path: %s", match.Method, match.Path))
		rw.Header().Add("Content-Type", "application/json")
		rw.WriteHeader(match.StatusCode)
		if match.Response != "" {
			_, err := rw.Write([]byte(match.Response))
			if err != nil {
				panic("failed to write response: " + err.Error())
			}
		}
	}))
}

func newDummyServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("[Dummy Server] %s %s\n", req.Method, req.URL.Path)

		rw.WriteHeader(200)
	}))
}
