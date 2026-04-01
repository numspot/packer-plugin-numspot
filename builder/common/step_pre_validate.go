package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepPreValidate checks that the destination image name does not already exist.
type StepPreValidate struct {
	DestImageName   string
	ForceDeregister bool
}

// Run executes the step to retrieve the Windows admin password.
//
//nolint:gocyclo // Run is our orchestration function
func (s *StepPreValidate) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if s.ForceDeregister {
		ui.Say("Force Deregister flag found, skipping prevalidating Image Name")
		return multistep.ActionContinue
	}

	client, ok := state.Get("client").(*numspot.NumspotClient)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}
	spaceID, ok := state.Get("space_id").(string)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Prevalidating Image Name: %s", s.DestImageName))

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	spaceUUID, err := parseSpaceID(spaceID)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	imageNames := []string{s.DestImageName}
	resp, err := apiClient.ReadImagesWithResponse(ctx, spaceUUID, &numspot.ReadImagesParams{
		ImageNames: &imageNames,
	})
	if err != nil {
		err := fmt.Errorf("error querying Image: %w", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if resp.JSON200 != nil && resp.JSON200.Items != nil && len(*resp.JSON200.Items) > 0 {
		err = fmt.Errorf("%w: %s", errImageNameConflict, s.DestImageName)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup is a no-op for this step.
func (s *StepPreValidate) Cleanup(_ multistep.StateBag) {}
