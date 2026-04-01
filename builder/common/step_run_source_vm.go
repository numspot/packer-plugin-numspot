package common

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

const (
	// RunSourceVmBSUExpectedRootDevice is the expected root device type for BSU-backed VMs.
	RunSourceVmBSUExpectedRootDevice = "bsu"
)

// StepRunSourceVm launches the source Numspot VM for the build.
type StepRunSourceVm struct {
	BlockDevices                BlockDevices
	Comm                        *communicator.Config
	Ctx                         interpolate.Context
	Debug                       bool
	BsuOptimized                bool
	EnableT2Unlimited           bool
	ExpectedRootDevice          string
	IamVmProfile                string
	VmInitiatedShutdownBehavior string
	VmType                      string
	IsRestricted                bool
	SourceImage                 string
	Tags                        TagMap
	UserData                    string
	UserDataFile                string
	VolumeTags                  TagMap
	RawRegion                   string
	BootMode                    string
	vmId                        string
}

// Run executes the step to launch the source VM.
//
//nolint:gocognit,gocyclo,maintidx // Packer step orchestration is inherently complex
func (s *StepRunSourceVm) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
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
	securityGroupIds, ok := state.Get("securityGroupIds").([]string)
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

	userData := s.UserData
	if s.UserDataFile != "" {
		contents, err := os.ReadFile(s.UserDataFile)
		if err != nil {
			state.Put("error", fmt.Errorf("%w: %w", errReadingUserDataFile, err))
			return multistep.ActionHalt
		}
		userData = string(contents)
	}

	if _, err := base64.StdEncoding.DecodeString(userData); err != nil {
		log.Printf("[DEBUG] base64 encoding user data...")
		userData = base64.StdEncoding.EncodeToString([]byte(userData))
	}

	ui.Say("Launching a source Numspot VM...")
	image, ok := state.Get("source_image").(numspot.Image)
	if !ok {
		state.Put("error", errSourceImageTypeAssertion)
		return multistep.ActionHalt
	}

	if image.Id == nil {
		state.Put("error", errSourceImageIDIsNil)
		return multistep.ActionHalt
	}
	s.SourceImage = *image.Id

	if s.ExpectedRootDevice != "" && image.RootDeviceType != nil &&
		*image.RootDeviceType != s.ExpectedRootDevice {
		state.Put("error", fmt.Errorf(
			"%w.\nExpected '%s', got '%s'",
			errInvalidRootDeviceType, s.ExpectedRootDevice, *image.RootDeviceType))
		return multistep.ActionHalt
	}

	ui.Say("Adding tags to source VM")
	if _, exists := s.Tags["Name"]; !exists {
		s.Tags["Name"] = "Packer Builder"
	}

	tags, err := s.Tags.ToNumspotTags(&s.Ctx, state)
	if err != nil {
		err := fmt.Errorf("%w: %w", errTaggingSourceVM, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	volTags, err := s.VolumeTags.ToNumspotTags(&s.Ctx, state)
	if err != nil {
		err := fmt.Errorf("%w: %w", errTaggingVolumes, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	blockDevice := s.BlockDevices.BuildNumspotLaunchDevices()
	subnetID, ok := state.Get("subnet_id").(string)
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

	spaceUuid := openapi_types.UUID{}
	if err := spaceUuid.UnmarshalText([]byte(spaceId)); err != nil {
		state.Put("error", fmt.Errorf("invalid space_id: %w", err))
		return multistep.ActionHalt
	}

	createVmReq := numspot.CreateVms{
		ImageId:             s.SourceImage,
		BootOnCreation:      ptrBool(true),
		BlockDeviceMappings: &blockDevice,
	}

	if s.VmType != "" {
		createVmReq.Type = s.VmType
	}
	if userData != "" {
		createVmReq.UserData = &userData
	}
	if s.Comm.SSHKeyPairName != "" {
		createVmReq.KeypairName = &s.Comm.SSHKeyPairName
	}
	if subnetID != "" {
		createVmReq.SubnetId = subnetID
	}
	if len(securityGroupIds) > 0 {
		createVmReq.SecurityGroupIds = &securityGroupIds
	}
	if s.ExpectedRootDevice == "bsu" && s.VmInitiatedShutdownBehavior != "" {
		createVmReq.VmInitiatedShutdownBehavior = &s.VmInitiatedShutdownBehavior
	}

	runResp, err := apiClient.CreateVmsWithResponse(ctx, spaceUuid, createVmReq)
	if err != nil {
		err := fmt.Errorf("error launching source VM: %w", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if runResp.JSON201 == nil || runResp.JSON201.Id == nil {
		state.Put("error", errNoVMReturnedFromCreate)
		return multistep.ActionHalt
	}

	vm := runResp.JSON201
	vmId := *vm.Id

	var volumeId string
	if vm.BlockDeviceMappings != nil && len(*vm.BlockDeviceMappings) > 0 {
		bdm := (*vm.BlockDeviceMappings)[0]
		if bdm.Bsu != nil && bdm.Bsu.VolumeId != nil {
			volumeId = *bdm.Bsu.VolumeId
		}
	}

	s.vmId = vmId

	ui.Message(fmt.Sprintf("VM Id: %s", vmId))
	ui.Say(fmt.Sprintf("Waiting for VM (%s) to become ready...", vmId))

	if err := numspot.WaitUntilVMRunning(ctx, apiClient, spaceId, vmId); err != nil {
		err := fmt.Errorf("error waiting for VM (%s) to become ready: %w", vmId, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if len(tags) > 0 {
		if err := createNumspotTags(ctx, apiClient, spaceUuid, vmId, tags); err != nil {
			err := fmt.Errorf("error creating tags for VM (%s): %w", vmId, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	if len(volTags) > 0 && volumeId != "" {
		if err := createNumspotTags(ctx, apiClient, spaceUuid, volumeId, volTags); err != nil {
			err := fmt.Errorf("error creating tags for volume (%s): %w", volumeId, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	publicipId, hasPublicIp := state.Get("publicip_id").(string)
	if hasPublicIp && publicipId != "" {
		ui.Say(fmt.Sprintf("Linking temporary PublicIp %s to instance %s", publicipId, vmId))
		linkReq := numspot.LinkPublicIpJSONBody{
			VmId: &vmId,
		}
		_, err := apiClient.LinkPublicIpWithResponse(
			ctx,
			spaceUuid,
			publicipId,
			numspot.LinkPublicIpJSONRequestBody(linkReq),
		)
		if err != nil {
			state.Put("error", fmt.Errorf("error linking PublicIp to VM: %w", err))
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	vmResp, err := apiClient.ReadVmsByIdWithResponse(ctx, spaceUuid, vmId)
	if err != nil || vmResp.JSON200 == nil {
		err = errFindingSourceVM
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	vmResult := vmResp.JSON200

	if hasPublicIp && publicipId != "" && (vmResult.PublicIp == nil || *vmResult.PublicIp == "") {
		ui.Say("Waiting for public IP to be assigned...")
		for range 30 {
			eipResp, err := apiClient.ReadPublicIpsWithResponse(ctx, spaceUuid, &numspot.ReadPublicIpsParams{
				Ids: &[]string{publicipId},
			})
			if err == nil && eipResp.JSON200 != nil && eipResp.JSON200.Items != nil && len(*eipResp.JSON200.Items) > 0 {
				eip := (*eipResp.JSON200.Items)[0]
				if eip.PublicIp != nil && *eip.PublicIp != "" {
					vmResult.PublicIp = eip.PublicIp
					ui.Message(fmt.Sprintf("Public IP assigned: %s", *vmResult.PublicIp))
					break
				}
			}
			time.Sleep(2 * time.Second)
		}
	}

	if s.Debug {
		if vmResult.PublicDnsName != nil {
			ui.Message(fmt.Sprintf("Public DNS: %s", *vmResult.PublicDnsName))
		}
		if vmResult.PublicIp != nil {
			ui.Message(fmt.Sprintf("Public IP: %s", *vmResult.PublicIp))
		}
	}

	state.Put("vm", *vmResult)
	state.Put("instance_id", vmId)

	return multistep.ActionContinue
}

// Cleanup terminates the source VM if the build was cancelled or failed.
func (s *StepRunSourceVm) Cleanup(state multistep.StateBag) {
	client, ok := state.Get("client").(*numspot.NumspotClient)
	if !ok {
		return
	}
	spaceId, ok := state.Get("space_id").(string)
	if !ok {
		return
	}
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		return
	}
	ctx := context.Background()

	if s.vmId == "" {
		return
	}

	ui.Say("Terminating the source Numspot VM...")
	apiClient, err := client.GetClient(ctx)
	if err != nil {
		ui.Error(fmt.Sprintf("Error getting client: %s", err.Error()))
		return
	}

	spaceUuid := openapi_types.UUID{}
	if err := spaceUuid.UnmarshalText([]byte(spaceId)); err != nil {
		ui.Error(fmt.Sprintf("Error parsing space_id: %s", err.Error()))
		return
	}

	_, err = apiClient.DeleteVmsWithResponse(ctx, spaceUuid, s.vmId)
	if err != nil {
		ui.Error(fmt.Sprintf("Error terminating VM, may still be around: %s", err.Error()))
		return
	}

	if err := numspot.WaitUntilVMDeleted(ctx, apiClient, spaceId, s.vmId); err != nil {
		ui.Error(err.Error())
	}
}

func createNumspotTags(
	ctx context.Context,
	client *numspot.ClientWithResponses,
	spaceUuid openapi_types.UUID,
	resourceId string,
	tags []numspot.Tag,
) error {
	if len(tags) == 0 {
		return nil
	}

	resourceTags := make([]numspot.ResourceTag, len(tags))
	for i, t := range tags {
		if t.Key != nil && t.Value != nil {
			resourceTags[i] = numspot.ResourceTag{
				Key:   *t.Key,
				Value: *t.Value,
			}
		}
	}

	tagReq := numspot.CreateTags{
		ResourceIds: []string{resourceId},
		Tags:        resourceTags,
	}

	_, err := client.CreateTagsWithResponse(ctx, spaceUuid, tagReq)
	if err != nil {
		return fmt.Errorf("creating tags for resource %q: %w", resourceId, err)
	}
	return nil
}

func ptrBool(v bool) *bool {
	return &v
}
