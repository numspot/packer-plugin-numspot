package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepCreateTags creates tags on the built image and its snapshots.
type StepCreateTags struct {
	Tags         TagMap
	SnapshotTags TagMap
	Ctx          interpolate.Context
}

// Run executes the step to create tags on the image and snapshots.
//
//nolint:gocognit,gocyclo // tags must be applied to multiple resource types with independent error handling
func (s *StepCreateTags) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
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
	spaceId, ok := state.Get("space_id").(string)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if !s.Tags.IsSet() && !s.SnapshotTags.IsSet() {
		return multistep.ActionContinue
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

	for _, imageID := range images {
		ui.Say(fmt.Sprintf("Adding tags to Image (%s)...", imageID))

		resourceIDs := []string{imageID}
		imageResp, err := apiClient.ReadImagesWithResponse(
			ctx,
			spaceUuid,
			&numspot.ReadImagesParams{
				Ids: &resourceIDs,
			},
		)
		if err != nil {
			err = fmt.Errorf("error retrieving details for Image (%s): %w", imageID, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		if imageResp.JSON200 == nil || imageResp.JSON200.Items == nil ||
			len(*imageResp.JSON200.Items) == 0 {
			err = fmt.Errorf("%w: %s", errImageRetrievalNoImages, imageID)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		image := (*imageResp.JSON200.Items)[0]
		snapshotIDs := []string{}

		if image.BlockDeviceMappings != nil {
			for _, device := range *image.BlockDeviceMappings {
				if device.Bsu != nil && device.Bsu.SnapshotId != nil {
					ui.Say(fmt.Sprintf("Tagging snapshot: %s", *device.Bsu.SnapshotId))
					resourceIDs = append(resourceIDs, *device.Bsu.SnapshotId)
					snapshotIDs = append(snapshotIDs, *device.Bsu.SnapshotId)
				}
			}
		}

		ui.Say("Creating Image tags")
		imageTags, err := s.Tags.ToNumspotTags(&s.Ctx, state)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		s.Tags.Report(ui)

		ui.Say("Creating snapshot tags")
		snapshotTags, err := s.SnapshotTags.ToNumspotTags(&s.Ctx, state)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		s.SnapshotTags.Report(ui)

		if len(imageTags) > 0 {
			resourceTags := make([]numspot.ResourceTag, len(imageTags))
			for i, t := range imageTags {
				resourceTags[i] = numspot.ResourceTag{
					Key:   *t.Key,
					Value: *t.Value,
				}
			}
			_, err = apiClient.CreateTagsWithResponse(ctx, spaceUuid, numspot.CreateTags{
				ResourceIds: resourceIDs,
				Tags:        resourceTags,
			})
			if err != nil {
				err = fmt.Errorf("error adding tags to Resources (%#v): %w", resourceIDs, err)
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}
		}

		if len(snapshotTags) > 0 && len(snapshotIDs) > 0 {
			snapTags := make([]numspot.ResourceTag, len(snapshotTags))
			for i, t := range snapshotTags {
				snapTags[i] = numspot.ResourceTag{
					Key:   *t.Key,
					Value: *t.Value,
				}
			}
			_, err = apiClient.CreateTagsWithResponse(ctx, spaceUuid, numspot.CreateTags{
				ResourceIds: snapshotIDs,
				Tags:        snapTags,
			})
			if err != nil {
				err = fmt.Errorf("error adding tags to snapshots (%#v): %w", snapshotIDs, err)
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}
		}
	}

	return multistep.ActionContinue
}

// Cleanup is a no-op for this step.
func (s *StepCreateTags) Cleanup(_ multistep.StateBag) {}
