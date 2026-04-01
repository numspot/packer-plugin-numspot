package common

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepDeregisterImage deregisters an existing image.
type StepDeregisterImage struct {
	AccessConfig        *AccessConfig
	ForceDeregister     bool
	ForceDeleteSnapshot bool
	ImageName           string
}

// Run executes the step to deregister the image.
func (s *StepDeregisterImage) Run( //nolint:gocyclo // deregistration requires iterating images and optionally deleting their snapshots
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	if !s.ForceDeregister {
		return multistep.ActionContinue
	}

	ui, ok := state.Get("ui").(packersdk.Ui)
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
	spaceId, ok := state.Get("space_id").(string)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}

	log.Printf("Deregistering image with name: %s", s.ImageName)

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		err := fmt.Errorf("error getting client: %w", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	spaceUuid, err := parseSpaceID(spaceId)
	if err != nil {
		err := fmt.Errorf("error parsing space_id: %w", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	imageNames := []string{s.ImageName}
	resp, err := apiClient.ReadImagesWithResponse(ctx, spaceUuid, &numspot.ReadImagesParams{
		ImageNames: &imageNames,
	})
	if err != nil {
		err := fmt.Errorf("error describing Image: %w", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		return multistep.ActionContinue
	}

	log.Printf("Found %d images", len(*resp.JSON200.Items))

	for _, img := range *resp.JSON200.Items {
		if img.Id == nil {
			continue
		}

		_, err := apiClient.DeleteImageWithResponse(ctx, spaceUuid, *img.Id)
		if err != nil {
			err := fmt.Errorf("error deregistering existing Image: %w", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		ui.Say(fmt.Sprintf("Deregistered Image %s, id: %s", s.ImageName, *img.Id))

		if s.ForceDeleteSnapshot && img.BlockDeviceMappings != nil {
			for _, b := range *img.BlockDeviceMappings {
				if b.Bsu != nil && b.Bsu.SnapshotId != nil {
					_, err := apiClient.DeleteSnapshotWithResponse(
						ctx,
						spaceUuid,
						*b.Bsu.SnapshotId,
					)
					if err != nil {
						err := fmt.Errorf("error deleting existing snapshot: %w", err)
						state.Put("error", err)
						ui.Error(err.Error())
						return multistep.ActionHalt
					}
					ui.Say("Deleted snapshot: " + *b.Bsu.SnapshotId)
				}
			}
		}
	}

	return multistep.ActionContinue
}

// Cleanup is a no-op for this step.
func (s *StepDeregisterImage) Cleanup(_ multistep.StateBag) {}
