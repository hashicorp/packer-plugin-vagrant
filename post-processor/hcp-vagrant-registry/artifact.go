// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package hcpvagrantregistry

import (
	"fmt"
)

const BuilderId = "hashicorp.post-processor.vagrant-registry"

type Artifact struct {
	Tag      string
	Provider string
}

func NewArtifact(provider, tag string) *Artifact {
	return &Artifact{
		Tag:      tag,
		Provider: provider,
	}
}

func (*Artifact) BuilderId() string {
	return BuilderId
}

func (*Artifact) Files() []string {
	return nil
}

func (*Artifact) Id() string {
	return ""
}

func (a *Artifact) String() string {
	return fmt.Sprintf("'%s': %s", a.Provider, a.Tag)
}

func (*Artifact) State(name string) interface{} {
	return nil
}

func (*Artifact) Destroy() error {
	return nil
}
