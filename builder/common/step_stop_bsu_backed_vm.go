package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepStopBSUBackedVM stops the BSU-backed VM before creating the image.
type StepStopBSUBackedVM struct {
	Skip          bool
	DisableStopVM bool
}

// Run executes the step to stop the source VM.
func (s *StepStopBSUBackedVM) Run( //nolint:gocyclo // state reads and shutdown-behavior switch each need independent error handling
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	client, ok := state.Get("client").(*numspot.NumspotClient)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}
	vm, ok := state.Get("vm").(numspot.Vm)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}
	spaceId, ok := state.Get("space_id").(string)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if s.Skip {
		return multistep.ActionContinue
	}

	if vm.Id == nil {
		err := errVMIDIsNil
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	spaceUuid, err := parseSpaceID(spaceId)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if !s.DisableStopVM {
		ui.Say("Stopping the source vm...")

		_, err = apiClient.StopVmWithResponse(ctx, spaceUuid, *vm.Id, numspot.StopVm{})
		if err != nil {
			err := fmt.Errorf("error stopping vm: %w", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	} else {
		ui.Say("Automatic vm stop disabled. Please stop vm manually.")
	}

	ui.Say("Waiting for the vm to stop...")

	var shutdownBehavior string
	if vm.InitiatedShutdownBehavior != nil {
		shutdownBehavior = *vm.InitiatedShutdownBehavior
	} else {
		shutdownBehavior = stopShutdownBehavior
	}

	switch shutdownBehavior {
	case stopShutdownBehavior:
		err = numspot.WaitUntilVMStopped(ctx, apiClient, spaceId, *vm.Id)
	case terminateShutdownBehavior:
		err = numspot.WaitUntilVMDeleted(ctx, apiClient, spaceId, *vm.Id)
	default:
		err = fmt.Errorf("%w: %s", errWrongShutdownBehavior, shutdownBehavior)
	}

	if err != nil {
		err := fmt.Errorf("error waiting for vm to stop: %w", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup is a no-op for this step.
func (s *StepStopBSUBackedVM) Cleanup(_ multistep.StateBag) {}
