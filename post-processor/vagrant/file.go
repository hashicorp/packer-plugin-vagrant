// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vagrant

import (
	"fmt"
	"path/filepath"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type FileProvider struct{}

func (p *FileProvider) KeepInputArtifact() bool {
	return false
}

func (p *FileProvider) Process(ui packersdk.Ui, artifact packersdk.Artifact, dir string) (vagrantfile string, metadata map[string]interface{}, err error) {
	// Create the metadata
	metadata = map[string]interface{}{"provider": "file"}

	// Copy all of the original contents into the temporary directory
	for _, path := range artifact.Files() {
		ui.Message(fmt.Sprintf("Copying: %s", path))

		dstPath := filepath.Join(dir, filepath.Base(path))
		if err = CopyContents(dstPath, path); err != nil {
			return
		}
	}

	return
}
