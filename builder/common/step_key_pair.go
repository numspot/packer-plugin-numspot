package common

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepKeyPair creates or uses an existing SSH key pair for the build VM.
type StepKeyPair struct {
	Debug        bool
	Comm         *communicator.Config
	DebugKeyPath string

	doCleanup bool
}

// Run executes the step to create or configure the SSH key pair.
//
//nolint:gocyclo // Run is our orchestration function
func (s *StepKeyPair) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if s.Comm.SSHPrivateKeyFile != "" {
		ui.Say("Using existing SSH private key")
		privateKeyBytes, err := s.Comm.ReadSSHPrivateKeyFile()
		if err != nil {
			state.Put("error", err)
			return multistep.ActionHalt
		}

		s.Comm.SSHPrivateKey = privateKeyBytes
		return multistep.ActionContinue
	}

	if s.Comm.SSHAgentAuth && s.Comm.SSHKeyPairName == "" {
		ui.Say("Using SSH Agent with key pair in Source Image")
		return multistep.ActionContinue
	}

	if s.Comm.SSHAgentAuth && s.Comm.SSHKeyPairName != "" {
		ui.Say(fmt.Sprintf("Using SSH Agent for existing key pair %s", s.Comm.SSHKeyPairName))
		return multistep.ActionContinue
	}

	if s.Comm.SSHTemporaryKeyPairName == "" {
		ui.Say("Not using temporary keypair")
		s.Comm.SSHKeyPairName = ""
		return multistep.ActionContinue
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

	ui.Say(fmt.Sprintf("Creating temporary keypair: %s", s.Comm.SSHTemporaryKeyPairName))

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

	resp, err := apiClient.CreateKeypairWithResponse(
		ctx,
		spaceUuid,
		numspot.CreateKeypairRequestSchema{
			Name: s.Comm.SSHTemporaryKeyPairName,
		},
	)
	if err != nil {
		state.Put("error", fmt.Errorf("error creating temporary keypair: %w", err))
		return multistep.ActionHalt
	}

	if resp.JSON201 == nil {
		state.Put("error", fmt.Errorf("error creating temporary keypair: %w: status %d", errNoVMReturnedFromCreate, resp.StatusCode()))
		return multistep.ActionHalt
	}

	if resp.JSON201.PrivateKey == nil {
		state.Put("error", errNoPrivateKeyReturned)
		return multistep.ActionHalt
	}

	s.doCleanup = true
	s.Comm.SSHKeyPairName = s.Comm.SSHTemporaryKeyPairName
	s.Comm.SSHPrivateKey = []byte(*resp.JSON201.PrivateKey)

	if s.Debug {
		ui.Message(fmt.Sprintf("Saving key for debug purposes: %s", s.DebugKeyPath))
		f, err := os.Create(s.DebugKeyPath)
		if err != nil {
			state.Put("error", fmt.Errorf("error saving debug key: %w", err))
			return multistep.ActionHalt
		}
		defer func() { _ = f.Close() }()

		if _, err := f.WriteString(*resp.JSON201.PrivateKey); err != nil {
			state.Put("error", fmt.Errorf("error saving debug key: %w", err))
			return multistep.ActionHalt
		}

		if runtime.GOOS != "windows" {
			if err := f.Chmod(0o600); err != nil {
				state.Put("error", fmt.Errorf("error setting permissions of debug key: %w", err))
				return multistep.ActionHalt
			}
		}
	}

	return multistep.ActionContinue
}

// Cleanup deletes the temporary SSH key pair created during the build.
func (s *StepKeyPair) Cleanup(state multistep.StateBag) {
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

	ui.Say("Deleting temporary keypair...")

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

	_, err = apiClient.DeleteKeypairWithResponse(ctx, spaceUuid, s.Comm.SSHTemporaryKeyPairName)
	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error cleaning up keypair. Please delete the key manually: %s",
			s.Comm.SSHTemporaryKeyPairName,
		))
	}

	if s.Debug {
		if err := os.Remove(s.DebugKeyPath); err != nil {
			ui.Error(fmt.Sprintf(
				"Error removing debug key '%s': %s", s.DebugKeyPath, err.Error()))
		}
	}
}
