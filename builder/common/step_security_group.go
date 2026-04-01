package common

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepSecurityGroup creates or selects a security group for the build VM.
type StepSecurityGroup struct {
	CommConfig            *communicator.Config
	SecurityGroupFilter   SecurityGroupFilterOptions
	SecurityGroupIDs      []string
	TemporarySGSourceCidr string

	createdGroupID string
}

// Run executes the step to create or select a security group.
func (s *StepSecurityGroup) Run( //nolint:gocyclo // three mutually exclusive paths: use IDs, use filter, or create temporary group
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
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
	spaceID, ok := state.Get("space_id").(string)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}
	netID, ok := state.Get("net_id").(string)
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

	if len(s.SecurityGroupIDs) > 0 {
		resp, err := apiClient.ReadSecurityGroupsWithResponse(
			ctx,
			spaceUUID,
			&numspot.ReadSecurityGroupsParams{
				SecurityGroupIds: &s.SecurityGroupIDs,
			},
		)
		if err != nil || resp.JSON200 == nil || resp.JSON200.Items == nil ||
			len(*resp.JSON200.Items) == 0 {
			err := fmt.Errorf("couldn't find specified security group: %w", err)
			log.Printf("[DEBUG] %s", err.Error())
			state.Put("error", err)
			return multistep.ActionHalt
		}

		log.Printf("Using specified security groups: %v", s.SecurityGroupIDs)
		state.Put("securityGroupIds", s.SecurityGroupIDs)
		return multistep.ActionContinue
	}

	if !s.SecurityGroupFilter.Empty() {
		log.Printf("Using SecurityGroup Filters %v", s.SecurityGroupFilter.Filters)

		resp, err := apiClient.ReadSecurityGroupsWithResponse(
			ctx,
			spaceUUID,
			&numspot.ReadSecurityGroupsParams{
				SecurityGroupNames: buildStringSliceFromMap(s.SecurityGroupFilter.Filters, "name"),
			},
		)
		if err != nil || resp.JSON200 == nil || resp.JSON200.Items == nil ||
			len(*resp.JSON200.Items) == 0 {
			err := fmt.Errorf("couldn't find security groups for filter: %w", err)
			log.Printf("[DEBUG] %s", err.Error())
			state.Put("error", err)
			return multistep.ActionHalt
		}

		securityGroupIds := []string{}
		for _, sg := range *resp.JSON200.Items {
			if sg.Id != nil {
				securityGroupIds = append(securityGroupIds, *sg.Id)
			}
		}

		ui.Message(fmt.Sprintf("Found Security Group(s): %s", strings.Join(securityGroupIds, ", ")))
		state.Put("securityGroupIds", securityGroupIds)
		return multistep.ActionContinue
	}

	groupName := fmt.Sprintf("packer-sg-%d", time.Now().Unix())
	ui.Say(fmt.Sprintf("Creating temporary security group for this instance: %s", groupName))

	if netID == "" {
		state.Put("error", errSecurityGroupVpcIdRequired)
		ui.Error(errSecurityGroupVpcIdRequired.Error())
		return multistep.ActionHalt
	}

	createSGReq := numspot.CreateSecurityGroup{
		Name:        groupName,
		Description: "Temporary group for Packer",
		VpcId:       netID,
	}

	resp, err := apiClient.CreateSecurityGroupWithResponse(ctx, spaceUUID, createSGReq)
	if err != nil {
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if resp.JSON201 == nil {
		err = fmt.Errorf("error creating security group: %w: status %d", errNoSecurityGroupIDReturned, resp.StatusCode())
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	if resp.JSON201.Id == nil {
		err = errNoSecurityGroupIDReturned
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.createdGroupID = *resp.JSON201.Id

	port := s.CommConfig.Port()
	if port == 0 {
		if s.CommConfig.Type != "none" {
			state.Put("error", "port must be set to a non-zero value.")
			return multistep.ActionHalt
		}
	}

	ipProtocol := "tcp"
	ui.Say(
		fmt.Sprintf(
			"Authorizing access to port %d from %s in the temporary security group...",
			port,
			s.TemporarySGSourceCidr,
		),
	)

	createSGRReq := numspot.CreateSecurityGroupRule{
		Flow:          "Inbound",
		IpProtocol:    &ipProtocol,
		FromPortRange: &port,
		ToPortRange:   &port,
		IpRange:       &s.TemporarySGSourceCidr,
	}

	_, err = apiClient.CreateSecurityGroupRuleWithResponse(
		ctx,
		spaceUUID,
		s.createdGroupID,
		createSGRReq,
	)
	if err != nil {
		err := fmt.Errorf("error authorizing temporary security group: %w", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("securityGroupIds", []string{s.createdGroupID})
	return multistep.ActionContinue
}

// Cleanup deletes the temporary security group created during the build.
func (s *StepSecurityGroup) Cleanup(state multistep.StateBag) {
	if s.createdGroupID == "" {
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

	ui.Say("Deleting temporary security group...")

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

	_, err = apiClient.DeleteSecurityGroupWithResponse(ctx, spaceUuid, s.createdGroupID)
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error cleaning up security group. Please delete the group manually: %s",
			s.createdGroupID,
		))
	}
}

func buildStringSliceFromMap(input map[string]string, key string) *[]string {
	if v, ok := input[key]; ok {
		return &[]string{v}
	}
	return nil
}
