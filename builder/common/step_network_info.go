package common

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sort"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepNetworkInfo resolves VPC/subnet information for the build VM.
type StepNetworkInfo struct {
	NetID                string
	NetFilter            NetFilterOptions
	SubnetID             string
	SubnetFilter         SubnetFilterOptions
	AvailabilityZoneName string
	SecurityGroupIDs     []string
	SecurityGroupFilter  SecurityGroupFilterOptions
}

type subnetsSort []numspot.Subnet

func (a subnetsSort) Len() int      { return len(a) }
func (a subnetsSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a subnetsSort) Less(i, j int) bool {
	var countI, countJ int
	if a[i].AvailableIpsCount != nil {
		countI = *a[i].AvailableIpsCount
	}
	if a[j].AvailableIpsCount != nil {
		countJ = *a[j].AvailableIpsCount
	}
	return countI < countJ
}

func mostFreeSubnet(subnets []numspot.Subnet) numspot.Subnet {
	sortedSubnets := subnets
	sort.Sort(subnetsSort(sortedSubnets))
	return sortedSubnets[len(sortedSubnets)-1]
}

// Run executes the step to resolve network configuration.
//
//nolint:gocognit,gocyclo // network resolution branches across subnet/VPC/default-net cases with necessary validation
func (s *StepNetworkInfo) Run(
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

	// VPC (Net)
	if s.NetID == "" && !s.NetFilter.Empty() {
		log.Printf("Using VPC Filters %v", s.NetFilter.Filters)

		vpcResp, err := apiClient.ReadVpcsWithResponse(ctx, spaceUUID, &numspot.ReadVpcsParams{})
		if err != nil {
			err := fmt.Errorf("error querying VPCs: %w", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		if vpcResp.JSON200 == nil || vpcResp.JSON200.Items == nil ||
			len(*vpcResp.JSON200.Items) != 1 {
			err = fmt.Errorf(
				"%w: %d VPCs were found",
				errMultipleVPCsMatched,
				len(*vpcResp.JSON200.Items),
			)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		vpc := (*vpcResp.JSON200.Items)[0]
		if vpc.Id != nil {
			s.NetID = *vpc.Id
		}
		ui.Message(fmt.Sprintf("Found VPC Id: %s", s.NetID))
	}

	// Subnet
	if s.SubnetID == "" && !s.SubnetFilter.Empty() {
		log.Printf("Using Subnet Filters %v", s.SubnetFilter.Filters)

		subnetsResp, err := apiClient.ReadSubnetsWithResponse(
			ctx,
			spaceUUID,
			&numspot.ReadSubnetsParams{},
		)
		if err != nil {
			err := fmt.Errorf("error querying Subnets: %w", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		if subnetsResp.JSON200 == nil || subnetsResp.JSON200.Items == nil ||
			len(*subnetsResp.JSON200.Items) == 0 {
			err = errNoSubnetsFoundMatchingFilters
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		subnets := *subnetsResp.JSON200.Items
		if len(subnets) > 1 && !s.SubnetFilter.Random && !s.SubnetFilter.MostFree {
			err := fmt.Errorf(
				"%w: %d subnets found",
				errMultipleSubnetsMatched,
				len(subnets),
			)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		var subnet numspot.Subnet
		switch {
		case s.SubnetFilter.MostFree:
			subnet = mostFreeSubnet(subnets)
		case s.SubnetFilter.Random:
			n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(len(subnets))))
			if err != nil {
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}
			subnet = subnets[n.Int64()]
		default:
			subnet = subnets[0]
		}

		if subnet.Id != nil {
			s.SubnetID = *subnet.Id
		}
		ui.Message(fmt.Sprintf("Found Subnet Id: %s", s.SubnetID))
	}

	// Try to find AvailabilityZone and VPC Id from Subnet if they are not yet found/given
	if s.SubnetID != "" && (s.AvailabilityZoneName == "" || s.NetID == "") {
		log.Printf("[INFO] Finding AvailabilityZone and NetId for the given subnet '%s'", s.SubnetID)

		resp, err := apiClient.ReadSubnetsByIdWithResponse(ctx, spaceUUID, s.SubnetID)
		if err != nil {
			err := fmt.Errorf("describing the subnet: %s returned error: %w", s.SubnetID, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		if resp.JSON200 != nil {
			if s.AvailabilityZoneName == "" && resp.JSON200.AvailabilityZoneName != nil {
				s.AvailabilityZoneName = string(*resp.JSON200.AvailabilityZoneName)
				log.Printf("[INFO] AvailabilityZoneName found: '%s'", s.AvailabilityZoneName)
			}
			if s.NetID == "" && resp.JSON200.VpcId != nil {
				s.NetID = *resp.JSON200.VpcId
				log.Printf("[INFO] NetId found: '%s'", s.NetID)
			}
		}
	}

	state.Put("net_id", s.NetID)
	state.Put("availability_zone", s.AvailabilityZoneName)
	state.Put("subnet_id", s.SubnetID)
	return multistep.ActionContinue
}

// Cleanup is a no-op for this step.
func (s *StepNetworkInfo) Cleanup(multistep.StateBag) {}
