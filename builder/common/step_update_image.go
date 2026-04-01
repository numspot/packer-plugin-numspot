package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepUpdateImageAttributes updates image permissions and account sharing settings.
type StepUpdateImageAttributes struct {
	AccountIDs         []string
	SnapshotAccountIDs []string
	RawRegion          string
	GlobalPermission   bool
	Ctx                interpolate.Context
}

// Run executes the step to update image attributes.
func (s *StepUpdateImageAttributes) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		return multistep.ActionContinue
	}

	images, ok := state.Get("images").(map[string]string)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
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

	valid := s.GlobalPermission
	if !valid {
		return multistep.ActionContinue
	}

	s.Ctx.Data = extractBuildInfo(s.RawRegion, state)

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	spaceUuid, err := parseSpaceID(spaceID)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	for _, imageID := range images {
		ui.Say(fmt.Sprintf("Updating attributes on Image (%s)...", imageID))
		ui.Message(fmt.Sprintf("Updating: %s", imageID))

		updateReq := numspot.UpdateImage{
			AccessCreation: numspot.AccessCreation{
				Additions: &numspot.Access{
					IsPublic: &s.GlobalPermission,
				},
			},
		}

		_, err := apiClient.UpdateImageWithResponse(ctx, spaceUuid, imageID, updateReq)
		if err != nil {
			err := fmt.Errorf("error updating Image: %w", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

// Cleanup is a no-op for this step.
func (s *StepUpdateImageAttributes) Cleanup(_ multistep.StateBag) {}
