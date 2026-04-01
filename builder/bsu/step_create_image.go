package bsu

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

type stepCreateImage struct {
	image        *numspot.Image
	RawRegion    string
	ProductCodes []string
	BootModes    []string
}

func parseSpaceID(spaceID string) (openapi_types.UUID, error) {
	var uuid openapi_types.UUID
	if err := uuid.UnmarshalText([]byte(spaceID)); err != nil {
		return uuid, fmt.Errorf("parsing space ID %q: %w", spaceID, err)
	}
	return uuid, nil
}

var errStateTypeCastFailed = errors.New("state type cast failed")

// Run is the sequential function that creates the image playing the steps
//
//nolint:gocognit,gocyclo // sequential image creation flow with necessary error handling at each step
func (s *stepCreateImage) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	config, ok := state.Get("config").(*Config)
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
	spaceID, ok := state.Get("space_id").(string)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}

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

	imageName := config.ImageName

	ui.Say(fmt.Sprintf("Creating Image %s from vm %s", imageName, *vm.Id))
	blockDeviceMapping := config.BuildNumspotImageDevices()
	createOpts := numspot.CreateImage{
		Name: &imageName,
	}
	if len(blockDeviceMapping) == 0 {
		createOpts.VmId = vm.Id
	} else {
		createOpts.BlockDeviceMappings = &blockDeviceMapping
		if rootDName := config.RootDeviceName; rootDName != "" {
			createOpts.RootDeviceName = &rootDName
		} else {
			err = errMissingRootDeviceName
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}
	if len(s.ProductCodes) > 0 {
		createOpts.ProductCodes = &s.ProductCodes
	}

	if description := config.ImageDescription; description != "" {
		createOpts.Description = &description
	}

	resp, err := apiClient.CreateImageWithResponse(ctx, spaceUUID, createOpts)
	if err != nil {
		err = fmt.Errorf("error creating Image: %w", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if resp.JSON201 == nil || resp.JSON201.Id == nil {
		err = errImageNoIDReturned
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	image := resp.JSON201

	ui.Message(fmt.Sprintf("Image: %s", *image.Id))
	images := make(map[string]string)
	images[s.RawRegion] = *image.Id
	state.Put("images", images)

	ui.Say("Waiting for Image to become ready...")
	if err := numspot.WaitUntilImageAvailable(ctx, apiClient, spaceID, *image.Id); err != nil {
		log.Printf("Error waiting for Image: %s", err)
		readResp, readErr := apiClient.ReadImagesWithResponse(
			ctx,
			spaceUUID,
			&numspot.ReadImagesParams{
				Ids: &[]string{*image.Id},
			},
		)
		if readErr != nil {
			log.Printf("Unable to determine reason waiting for Image failed: %s", readErr)
			err = errUnknownWaitingForImage
		} else if readResp.JSON200 != nil && readResp.JSON200.Items != nil && len(*readResp.JSON200.Items) > 0 {
			img := (*readResp.JSON200.Items)[0]
			if img.StateComment != nil && img.StateComment.StateMessage != nil {
				err = fmt.Errorf(
					"%w: %s",
					errImageWaitingForReason,
					*img.StateComment.StateMessage,
				)
			}
		}

		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	readResp, err := apiClient.ReadImagesWithResponse(ctx, spaceUUID, &numspot.ReadImagesParams{
		Ids: &[]string{*image.Id},
	})
	if err != nil {
		err := fmt.Errorf("error searching for Image: %w", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	if readResp.JSON200 == nil || readResp.JSON200.Items == nil ||
		len(*readResp.JSON200.Items) == 0 {
		err = errImageNoImageReturned
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.image = &(*readResp.JSON200.Items)[0]
	if s.image == nil {
		err = errImageEmptyID
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	snapshots := make(map[string][]string)
	if s.image.BlockDeviceMappings != nil {
		for _, blockDeviceMapping := range *s.image.BlockDeviceMappings {
			if blockDeviceMapping.Bsu != nil && blockDeviceMapping.Bsu.SnapshotId != nil {
				snapshots[s.RawRegion] = append(
					snapshots[s.RawRegion],
					*blockDeviceMapping.Bsu.SnapshotId,
				)
			}
		}
	}
	state.Put("snapshots", snapshots)

	return multistep.ActionContinue
}

// Cleanup deregisters the created image if the build was cancelled or halted.
func (s *stepCreateImage) Cleanup(state multistep.StateBag) {
	if s.image == nil {
		return
	}

	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if !cancelled && !halted {
		return
	}

	client, ok := state.Get("client").(*numspot.NumspotClient)
	if !ok {
		return
	}
	spaceID, ok := state.Get("space_id").(string)
	if !ok {
		return
	}
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		return
	}

	ui.Say("Deregistering the Image because cancellation or error...")

	apiClient, err := client.GetClient(context.Background())
	if err != nil {
		ui.Error(fmt.Sprintf("error getting client: %v", err))
		return
	}

	spaceUUID, err := parseSpaceID(spaceID)
	if err != nil {
		ui.Error(fmt.Sprintf("error parsing space_id: %v", err))
		return
	}

	_, err = apiClient.DeleteImageWithResponse(context.Background(), spaceUUID, *s.image.Id)
	if err != nil {
		ui.Error(fmt.Sprintf("error deleting Image, may still be around: %v", err))
		return
	}
}
