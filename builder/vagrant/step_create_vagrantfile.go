// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package vagrant

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCreateVagrantfile struct {
	Template               string
	OutputDir              string
	SyncedFolder           string
	GlobalID               string
	SourceBox              string
	BoxName                string
	InsertKey              bool
	defaultTemplateContent string
}

type VagrantfileOptions struct {
	SyncedFolder    string
	SourceBox       string
	BoxName         string
	InsertKey       bool
	DefaultTemplate string
}

const DEFAULT_TEMPLATE = `Vagrant.configure("2") do |config|
  config.vm.define "source", autostart: false do |source|
	source.vm.box = "{{.SourceBox}}"
	config.ssh.insert_key = {{.InsertKey}}
  end
  config.vm.define "output" do |output|
	output.vm.box = "{{.BoxName}}"
	output.vm.box_url = "file://package.box"
	config.ssh.insert_key = {{.InsertKey}}
  end
  {{ if ne .SyncedFolder "" -}}
  		config.vm.synced_folder "{{.SyncedFolder}}", "/vagrant"
  {{- else -}}
  		config.vm.synced_folder ".", "/vagrant", disabled: true
  {{- end}}
end`

var defaultTemplate = template.Must(template.New("VagrantTpl").Parse(DEFAULT_TEMPLATE))

func (s *StepCreateVagrantfile) createVagrantfile() (tplPath string, err error) {
	tplPath, err = filepath.Abs(filepath.Join(s.OutputDir, "Vagrantfile"))
	if err != nil {
		return
	}

	templateFile, err := os.Create(tplPath)
	if err != nil {
		err = fmt.Errorf("Error creating vagrantfile %s", err.Error())
		return
	}

	if s.defaultTemplateContent, err = s.renderDefaultTemplate(); err != nil {
		return
	}

	if s.Template == "" {
		// Generate vagrantfile template based on our default template
		_, err = templateFile.WriteString(s.defaultTemplateContent)
	} else {
		// Otherwise, read in the template from provided file.
		var tpl *template.Template
		tpl, err = template.ParseFiles(s.Template)
		if err == nil {
			err = s.executeTemplate(tpl, templateFile)
		}
	}
	return
}

func (s *StepCreateVagrantfile) executeTemplate(tpl *template.Template, file io.Writer) error {
	opts := &VagrantfileOptions{
		SyncedFolder:    s.SyncedFolder,
		BoxName:         s.BoxName,
		SourceBox:       s.SourceBox,
		InsertKey:       s.InsertKey,
		DefaultTemplate: s.defaultTemplateContent,
	}
	return tpl.Execute(file, opts)
}

func (s *StepCreateVagrantfile) renderDefaultTemplate() (string, error) {
	buf := new(strings.Builder)
	if err := s.executeTemplate(defaultTemplate, buf); err != nil {
		return "", fmt.Errorf("Error rendering default template %w", err)
	}
	return buf.String(), nil
}

func (s *StepCreateVagrantfile) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)

	// Skip the initialize step if we're trying to launch from a global ID.
	if s.GlobalID != "" {
		ui.Say("Using a global-id; skipping Vagrant init in this directory...")
		return multistep.ActionContinue
	}

	ui.Say("Creating a Vagrantfile in the build directory...")
	vagrantfilePath, err := s.createVagrantfile()
	if err != nil {
		state.Put("error", err)
		return multistep.ActionHalt
	}
	log.Printf("Created vagrantfile at %s", vagrantfilePath)

	return multistep.ActionContinue
}

func (s *StepCreateVagrantfile) Cleanup(state multistep.StateBag) {
}
