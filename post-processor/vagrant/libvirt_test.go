package vagrant

import (
	"fmt"
	"os"
	"testing"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/tmp"
	"gotest.tools/assert"
)

func assertSizeInMegabytes(t *testing.T, size string, expected uint64) {
	actual := sizeInMegabytes(size)
	if actual != expected {
		t.Fatalf("the size `%s` was converted to `%d` but expected `%d`", size, actual, expected)
	}
}

func Test_sizeInMegabytes_WithInvalidUnitMustPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected a panic but got none")
		}
	}()

	sizeInMegabytes("1234x")
}

func Test_sizeInMegabytes_WithoutUnitMustDefaultToMegabytes(t *testing.T) {
	assertSizeInMegabytes(t, "1234", 1234)
}

func Test_sizeInMegabytes_WithBytesUnit(t *testing.T) {
	assertSizeInMegabytes(t, fmt.Sprintf("%db", 1234*1024*1024), 1234)
	assertSizeInMegabytes(t, fmt.Sprintf("%dB", 1234*1024*1024), 1234)
	assertSizeInMegabytes(t, "1B", 0)
}

func Test_sizeInMegabytes_WithKiloBytesUnit(t *testing.T) {
	assertSizeInMegabytes(t, fmt.Sprintf("%dk", 1234*1024), 1234)
	assertSizeInMegabytes(t, fmt.Sprintf("%dK", 1234*1024), 1234)
	assertSizeInMegabytes(t, "1K", 0)
}

func Test_sizeInMegabytes_WithMegabytesUnit(t *testing.T) {
	assertSizeInMegabytes(t, "1234m", 1234)
	assertSizeInMegabytes(t, "1234M", 1234)
	assertSizeInMegabytes(t, "1M", 1)
}

func Test_sizeInMegabytes_WithGigabytesUnit(t *testing.T) {
	assertSizeInMegabytes(t, "1234g", 1234*1024)
	assertSizeInMegabytes(t, "1234G", 1234*1024)
	assertSizeInMegabytes(t, "1G", 1*1024)
}

func Test_sizeInMegabytes_WithTerabytesUnit(t *testing.T) {
	assertSizeInMegabytes(t, "1234t", 1234*1024*1024)
	assertSizeInMegabytes(t, "1234T", 1234*1024*1024)
	assertSizeInMegabytes(t, "1T", 1*1024*1024)
}

func Test_sizeInMegabytes_WithPetabytesUnit(t *testing.T) {
	assertSizeInMegabytes(t, "1234p", 1234*1024*1024*1024)
	assertSizeInMegabytes(t, "1234P", 1234*1024*1024*1024)
	assertSizeInMegabytes(t, "1P", 1*1024*1024*1024)
}

func Test_sizeInMegabytes_WithExabytesUnit(t *testing.T) {
	assertSizeInMegabytes(t, "1234e", 1234*1024*1024*1024*1024)
	assertSizeInMegabytes(t, "1234E", 1234*1024*1024*1024*1024)
	assertSizeInMegabytes(t, "1E", 1*1024*1024*1024*1024)
}

func Test_ManyFilesInArtifact(t *testing.T) {
	p := new(LibVirtProvider)
	ui := testUi()
	type testCases struct {
		Files         []string
		Format        string
		FilesExpected []string
	}
	testcases := []testCases{
		{
			[]string{},
			"qcow2",
			[]string{},
		},
		{
			[]string{"test-0"},
			"vmdk",
			[]string{"box_0.img"},
		},
		{
			[]string{"test-0", "/test-1", "test-2"},
			"qcow2",
			[]string{"box_0.img", "box_1.img", "box_2.img"},
		},
	}
	for _, tc := range testcases {
		dir, _ := tmp.Dir("pkr")
		defer os.RemoveAll(dir)

		actifactFiles := []string{}
		for _, file := range tc.Files {
			fullFilePath := fmt.Sprintf("%s/%s", dir, file)
			actifactFiles = append(actifactFiles, fullFilePath)
			_, err := os.Create(fullFilePath)
			if err != nil {
				t.Fatalf("Can't create %s : %s", fullFilePath, err)
			}
		}

		artifact := &packersdk.MockArtifact{
			FilesValue: actifactFiles,
			StateValues: map[string]interface{}{
				"diskType":   tc.Format,
				"diskSize":   "1234M",
				"domainType": "kvm",
			},
		}

		_, metadata, err := p.Process(ui, artifact, "foo")

		if err != nil {
			t.Fatalf("should not have error: %s", err)
		}
		metaDisks := metadata["disks"].([]map[string]string)
		assert.Equal(t, len(tc.FilesExpected), len(metaDisks),
			fmt.Sprintf("%s. Expected %#v", "Disk files count must same", len(tc.FilesExpected)))

		for i, disk := range metaDisks {
			assert.Equal(t, tc.FilesExpected[i], disk["path"],
				fmt.Sprintf("%s. Expected %#v", "Disk files order must be respected", tc.FilesExpected[i]))
			assert.Equal(t, tc.Format, disk["format"],
				fmt.Sprintf("%s. Expected %#v", "Disk files format must be present", tc.Format))
		}
	}

}
