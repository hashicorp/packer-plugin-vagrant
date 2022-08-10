// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vagrant

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func TestStepCreateVagrantfile_Impl(t *testing.T) {
	var raw interface{}
	raw = new(StepCreateVagrantfile)
	if _, ok := raw.(multistep.Step); !ok {
		t.Fatalf("initialize should be a step")
	}
}

func TestCreateFile(t *testing.T) {
	testy := StepCreateVagrantfile{
		OutputDir: "./",
		SourceBox: "apples",
		BoxName:   "bananas",
	}
	templatePath, err := testy.createVagrantfile()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(templatePath)
	contents, err := ioutil.ReadFile(templatePath)
	if err != nil {
		t.Fatalf(err.Error())
	}
	actual := string(contents)
	expected := `Vagrant.configure("2") do |config|
  config.vm.define "source", autostart: false do |source|
	source.vm.box = "apples"
	config.ssh.insert_key = false
  end
  config.vm.define "output" do |output|
	output.vm.box = "bananas"
	output.vm.box_url = "file://package.box"
	config.ssh.insert_key = false
  end
  config.vm.synced_folder ".", "/vagrant", disabled: true
end`
	if ok := strings.Compare(actual, expected); ok != 0 {
		t.Fatalf("EXPECTED: \n%s\n\n RECEIVED: \n%s\n\n", expected, actual)
	}
}

func TestCreateFile_customSync(t *testing.T) {
	testy := StepCreateVagrantfile{
		OutputDir:    "./",
		SyncedFolder: "myfolder/foldertimes",
	}
	templatePath, err := testy.createVagrantfile()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(templatePath)
	contents, err := ioutil.ReadFile(templatePath)
	if err != nil {
		t.Fatalf(err.Error())
	}
	actual := string(contents)
	expected := `Vagrant.configure("2") do |config|
  config.vm.define "source", autostart: false do |source|
	source.vm.box = ""
	config.ssh.insert_key = false
  end
  config.vm.define "output" do |output|
	output.vm.box = ""
	output.vm.box_url = "file://package.box"
	config.ssh.insert_key = false
  end
  config.vm.synced_folder "myfolder/foldertimes", "/vagrant"
end`
	if ok := strings.Compare(actual, expected); ok != 0 {
		t.Fatalf("EXPECTED: \n%s\n\n RECEIVED: \n%s\n\n", expected, actual)
	}
}

func TestCreateFile_InsertKeyTrue(t *testing.T) {
	testy := StepCreateVagrantfile{
		OutputDir: "./",
		InsertKey: true,
	}
	templatePath, err := testy.createVagrantfile()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(templatePath)
	contents, err := ioutil.ReadFile(templatePath)
	if err != nil {
		t.Fatalf(err.Error())
	}
	actual := string(contents)
	expected := `Vagrant.configure("2") do |config|
  config.vm.define "source", autostart: false do |source|
	source.vm.box = ""
	config.ssh.insert_key = true
  end
  config.vm.define "output" do |output|
	output.vm.box = ""
	output.vm.box_url = "file://package.box"
	config.ssh.insert_key = true
  end
  config.vm.synced_folder ".", "/vagrant", disabled: true
end`
	if ok := strings.Compare(actual, expected); ok != 0 {
		t.Fatalf("EXPECTED: \n%s\n\n RECEIVED: \n%s\n\n", expected, actual)
	}
}

func TestCreateFile_customTemplate(t *testing.T) {
	workdir := t.TempDir()
	vagrantfileTemplatePath := filepath.Join(workdir, "Vagrantfile.tpl")
	f, err := os.Create(vagrantfileTemplatePath)
	if err != nil {
		t.Fatal(err.Error())
	}
	defer f.Close()

	var TEMPLATE = `{{ .DefaultTemplate }}
Vagrant.configure("2") do |config|
  config.vm.provider "virtualbox" do |vb|
    vb.customize ['modifyvm', :id, '--nested-hw-virt', 'on']
  end
end`

	_, err = f.WriteString(TEMPLATE)
	if err != nil {
		t.Fatal(err.Error())
	}

	testy := StepCreateVagrantfile{
		OutputDir: "./",
		SourceBox: "apples",
		BoxName:   "bananas",
		Template:  vagrantfileTemplatePath,
	}
	templatePath, err := testy.createVagrantfile()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer os.Remove(templatePath)
	contents, err := ioutil.ReadFile(templatePath)
	if err != nil {
		t.Fatalf(err.Error())
	}
	actual := string(contents)
	expected := `Vagrant.configure("2") do |config|
  config.vm.define "source", autostart: false do |source|
	source.vm.box = "apples"
	config.ssh.insert_key = false
  end
  config.vm.define "output" do |output|
	output.vm.box = "bananas"
	output.vm.box_url = "file://package.box"
	config.ssh.insert_key = false
  end
  config.vm.synced_folder ".", "/vagrant", disabled: true
end
Vagrant.configure("2") do |config|
  config.vm.provider "virtualbox" do |vb|
    vb.customize ['modifyvm', :id, '--nested-hw-virt', 'on']
  end
end`
	if ok := strings.Compare(actual, expected); ok != 0 {
		t.Fatalf("EXPECTED: \n%s\n\n RECEIVED: \n%s\n\n", expected, actual)
	}
}
