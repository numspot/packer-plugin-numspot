package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepSourceImageInfo retrieves information about the source image.
type StepSourceImageInfo struct {
	SourceImage  string
	ImageFilters ImageFilterOptions
	AccessConfig *AccessConfig
}

// Run executes the step to get source image information.
func (s *StepSourceImageInfo) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	client, ok := state.Get("client").(*numspot.NumspotClient)
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
	spaceID, ok := state.Get("space_id").(string)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}

	var image *numspot.Image
	var err error

	if s.SourceImage != "" {
		image, err = s.getImageByID(ctx, client, spaceID, s.SourceImage)
	} else {
		image, err = s.ImageFilters.GetFilteredImage(ctx, client, spaceID)
	}

	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if image == nil || image.Id == nil {
		err = errNoImageFoundMatchingFilters
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Message(fmt.Sprintf("Found Image Id: %s", *image.Id))

	state.Put("source_image", *image)
	return multistep.ActionContinue
}

// Cleanup is a no-op for this step.
func (s *StepSourceImageInfo) Cleanup(_ multistep.StateBag) {}

func (s *StepSourceImageInfo) getImageByID(
	ctx context.Context,
	client *numspot.NumspotClient,
	spaceID, imageID string,
) (*numspot.Image, error) {
	apiClient, err := client.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting client: %w", err)
	}

	spaceUUID, err := parseSpaceID(spaceID)
	if err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	ids := []string{imageID}
	resp, err := apiClient.ReadImagesWithResponse(ctx, spaceUUID, &numspot.ReadImagesParams{
		Ids: &ids,
	})
	if err != nil {
		return nil, fmt.Errorf("error querying Image: %w", err)
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil || len(*resp.JSON200.Items) == 0 {
		return nil, fmt.Errorf("%w: %s", errImageNotFoundByID, imageID)
	}

	return &(*resp.JSON200.Items)[0], nil
}
