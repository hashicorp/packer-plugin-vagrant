// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vagrant

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestParallelsProvider_impl(t *testing.T) {
	var _ Provider = new(ParallelsProvider)
}

// mockParallelsVMDir creates a fake temp dir for parallels testing
//
// Note: the path to the pvm/macvm dir is returned, the responsibility to remove
// it befalls the caller.
func mockParallelsVMDir() ([]string, error) {
	tmpDir := fmt.Sprintf("%s/%d.pvm", os.TempDir(), rand.Uint32())
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		return nil, err
	}

	file1, err := os.CreateTemp(tmpDir, "")
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}
	file1.Close()

	file2, err := os.CreateTemp(tmpDir, "")
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, err
	}
	file2.Close()

	return []string{
		tmpDir,
		file1.Name(),
		file2.Name(),
	}, nil
}

func TestPostProcessorPostProcessParallels(t *testing.T) {
	var p PostProcessor

	inputVM, err := mockParallelsVMDir()
	if err != nil {
		t.Fatalf("failed to create parallels VM directory")
	}
	dir := inputVM[0]
	defer os.RemoveAll(dir)

	f, err := ioutil.TempFile("", "packer")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.Remove(f.Name())

	c := map[string]interface{}{
		"packer_user_variables": map[string]string{
			"foo": f.Name(),
		},

		"vagrantfile_template": "{{user `foo`}}",
	}
	err = p.Configure(c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	a := &packersdk.MockArtifact{
		BuilderIdValue: "packer.parallels",
		FilesValue:     inputVM,
	}
	a2, _, _, err := p.PostProcess(context.Background(), testUi(), a)
	if a2 != nil {
		for _, fn := range a2.Files() {
			defer os.Remove(fn)
		}
	}
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestPostProcessorPostProcessParallels_NoFileErrorOnCopy(t *testing.T) {
	var p PostProcessor

	c := map[string]interface{}{}
	err := p.Configure(c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	a := &packersdk.MockArtifact{
		BuilderIdValue: "packer.parallels",
	}
	a2, _, _, err := p.PostProcess(context.Background(), testUi(), a)
	if a2 != nil {
		for _, fn := range a2.Files() {
			defer os.Remove(fn)
		}
	}
	if err == nil {
		t.Fatalf("should have failed without a file to copy, succeeded instead")
	}
	t.Logf("failed as expected: %s", err)
}
