package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepCleanupVolumes cleans up any extra volumes created during the build.
type StepCleanupVolumes struct {
	BlockDevices BlockDevices
}

// Run is a no-op for this step; cleanup happens in Cleanup.
func (s *StepCleanupVolumes) Run(_ context.Context, _ multistep.StateBag) multistep.StepAction {
	return multistep.ActionContinue
}

// Cleanup removes any extra volumes that were not explicitly mapped.
func (s *StepCleanupVolumes) Cleanup(state multistep.StateBag) { //nolint:gocognit,gocyclo // multi-pass volume filtering with sequential API calls
	ctx := context.Background()
	client, ok := state.Get("client").(*numspot.NumspotClient)
	if !ok {
		return
	}
	vmRaw := state.Get("vm")
	var vm numspot.Vm
	if vmRaw != nil {
		vm, _ = vmRaw.(numspot.Vm)
	}
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		return
	}
	spaceId, ok := state.Get("space_id").(string)
	if !ok {
		return
	}

	ui.Say("Cleaning up any extra volumes...")

	var vl []string
	volList := make(map[string]string)
	if vm.BlockDeviceMappings != nil {
		for _, bdm := range *vm.BlockDeviceMappings {
			if bdm.Bsu != nil && bdm.Bsu.VolumeId != nil {
				vl = append(vl, *bdm.Bsu.VolumeId)
				if bdm.DeviceName != nil {
					volList[*bdm.Bsu.VolumeId] = *bdm.DeviceName
				}
			}
		}
	}

	if len(vl) == 0 {
		ui.Say("No volumes to clean up, skipping")
		return
	}

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		ui.Say(fmt.Sprintf("Error getting client: %s", err))
		return
	}

	spaceUuid, err := parseSpaceID(spaceId)
	if err != nil {
		ui.Say(fmt.Sprintf("Error parsing space_id: %s", err))
		return
	}

	resp, err := apiClient.ReadVolumesWithResponse(ctx, spaceUuid, &numspot.ReadVolumesParams{
		Ids: &vl,
	})
	if err != nil {
		ui.Say(fmt.Sprintf("Error describing volumes: %s", err))
		return
	}

	if resp.JSON200 == nil || resp.JSON200.Items == nil {
		ui.Say("No volumes to clean up, skipping")
		return
	}

	for _, v := range *resp.JSON200.Items {
		if v.State != nil && *v.State != "available" {
			if v.Id != nil {
				delete(volList, *v.Id)
			}
		}
	}

	if len(*resp.JSON200.Items) == 0 {
		ui.Say("No volumes to clean up, skipping")
		return
	}

	for _, b := range s.BlockDevices.LaunchMappings {
		for volKey, volName := range volList {
			if volName == b.DeviceName {
				delete(volList, volKey)
			}
		}
	}

	for k := range volList {
		ui.Say(fmt.Sprintf("Destroying volume (%s)...", k))
		_, err := apiClient.DeleteVolumeWithResponse(ctx, spaceUuid, k)
		if err != nil {
			ui.Say(fmt.Sprintf("Error deleting volume: %s", err))
		}
	}
}
