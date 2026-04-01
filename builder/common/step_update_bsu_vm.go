package common

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

// StepUpdateBSUBackedVM is a multistep.Step implementation used by Packer
// to update a BSU-backed VM with specific image capabilities.
type StepUpdateBSUBackedVM struct {
	EnableImageENASupport      *bool
	EnableImageSriovNetSupport bool
}

// Run executes the step.
func (s *StepUpdateBSUBackedVM) Run(
	_ context.Context,
	_ multistep.StateBag,
) multistep.StepAction {
	return multistep.ActionContinue
}

// Cleanup performs post-step cleanup.
func (s *StepUpdateBSUBackedVM) Cleanup(_ multistep.StateBag) {}
