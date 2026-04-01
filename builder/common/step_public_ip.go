package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepPublicIp optionally creates and associates a temporary public IP with the build VM.
type StepPublicIp struct {
	AssociatePublicIpAddress bool
	Comm                     *communicator.Config
	Debug                    bool

	publicIpId string
	doCleanup  bool
}

// Run executes the step to create and associate a temporary public IP.
func (s *StepPublicIp) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
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

	if !s.AssociatePublicIpAddress {
		return multistep.ActionContinue
	}

	ui.Say("Creating temporary PublicIp for instance ")

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		state.Put("error", fmt.Errorf("error getting client: %w", err))
		return multistep.ActionHalt
	}

	spaceUuid, err := parseSpaceID(spaceId)
	if err != nil {
		state.Put("error", fmt.Errorf("error parsing space_id: %w", err))
		return multistep.ActionHalt
	}

	resp, err := apiClient.CreatePublicIpWithResponse(ctx, spaceUuid)
	if err != nil {
		state.Put("error", fmt.Errorf("error creating temporary PublicIp: %w", err))
		return multistep.ActionHalt
	}

	if resp.JSON201 == nil || resp.JSON201.Id == nil {
		state.Put("error", errNoPublicIPIDReturned)
		return multistep.ActionHalt
	}

	s.doCleanup = true
	s.publicIpId = *resp.JSON201.Id
	state.Put("publicip_id", *resp.JSON201.Id)

	return multistep.ActionContinue
}

// Cleanup deletes the temporary public IP created during the build.
func (s *StepPublicIp) Cleanup(state multistep.StateBag) {
	if !s.doCleanup {
		return
	}

	client, ok := state.Get("client").(*numspot.NumspotClient)
	if !ok {
		return
	}
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		return
	}
	spaceId, ok := state.Get("space_id").(string)
	if !ok {
		return
	}

	ui.Say("Deleting temporary PublicIp...")

	ctx := context.Background()
	apiClient, err := client.GetClient(ctx)
	if err != nil {
		ui.Error(fmt.Sprintf("Error getting client: %s", err))
		return
	}

	spaceUuid, err := parseSpaceID(spaceId)
	if err != nil {
		ui.Error(fmt.Sprintf("Error parsing space_id: %s", err))
		return
	}

	_, err = apiClient.DeletePublicIpWithResponse(ctx, spaceUuid, s.publicIpId)
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error cleaning up PublicIp. Please delete the PublicIp manually: %s",
			s.publicIpId,
		))
	}
}
