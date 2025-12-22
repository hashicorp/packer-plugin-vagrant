// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package hcpvagrantregistry

import (
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestArtifact_ImplementsArtifact(t *testing.T) {
	var raw interface{}
	raw = &Artifact{}
	if _, ok := raw.(packer.Artifact); !ok {
		t.Fatalf("Artifact should be a Artifact")
	}
}
