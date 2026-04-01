package common

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

var sshHostSleepDuration = time.Second

func isNotEmpty(v *string) bool {
	return v != nil && *v != ""
}

// SSHHost returns a function that resolves the SSH host for the build VM.
func SSHHost( //nolint:gocognit,gocyclo // multi-protocol SSH host resolution with necessary branching per interface type
	_ context.Context,
	client *numspot.NumspotClient,
	spaceID, sshInterface string,
) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) { //nolint:contextcheck // outer context not used intentionally; a fresh background context is used per-call to avoid cancellation issues
		ctx := context.Background()

		var spaceUuid openapi_types.UUID
		if err := spaceUuid.UnmarshalText([]byte(spaceID)); err != nil {
			return "", fmt.Errorf("parsing space ID: %w", err)
		}

		const tries = 2
		for j := 0; j <= tries; j++ {
			var host string
			vm, ok := state.Get("vm").(numspot.Vm)
			if !ok {
				return "", errStateTypeCastFailed
			}

			switch {
			case sshInterface != "":
				switch sshInterface {
				case "public_ip":
					if isNotEmpty(vm.PublicIp) {
						host = *vm.PublicIp
					}
				case "public_dns":
					if isNotEmpty(vm.PublicDnsName) {
						host = *vm.PublicDnsName
					}
				case "private_ip":
					if vm.PrivateIp != nil && *vm.PrivateIp != "" {
						host = *vm.PrivateIp
					}
				case "private_dns":
					if isNotEmpty(vm.PrivateDnsName) {
						host = *vm.PrivateDnsName
					}
				default:
					panic(fmt.Sprintf("Unknown interface type: %s", sshInterface))
				}
			case isNotEmpty(vm.VpcId):
				if isNotEmpty(vm.PublicIp) {
					host = *vm.PublicIp
				} else if vm.PrivateIp != nil && *vm.PrivateIp != "" {
					host = *vm.PrivateIp
				}
			case isNotEmpty(vm.PublicDnsName):
				host = *vm.PublicDnsName
			}

			if host != "" {
				return host, nil
			}

			apiClient, err := client.GetClient(ctx)
			if err != nil {
				return "", fmt.Errorf("getting api client: %w", err)
			}

			if vm.Id == nil {
				return "", errVMNoID
			}

			r, err := apiClient.ReadVmsByIdWithResponse(ctx, spaceUuid, *vm.Id)
			if err != nil {
				return "", fmt.Errorf("reading VM by ID: %w", err)
			}

			if r.JSON200 == nil {
				return "", fmt.Errorf("%w: %s", errVMNotFound, *vm.Id)
			}

			state.Put("vm", *r.JSON200)
			time.Sleep(sshHostSleepDuration)
		}

		return "", errCouldNotDetermineVMAddress
	}
}
