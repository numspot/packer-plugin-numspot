package common

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// StepGetPassword retrieves the auto-generated Windows password for WinRM access.
type StepGetPassword struct {
	Debug     bool
	Comm      *communicator.Config
	Timeout   time.Duration
	BuildName string
}

// Run executes the step to retrieve the Windows admin password.
//
//nolint:gocyclo // Run is our orchestration function
func (s *StepGetPassword) Run(
	ctx context.Context,
	state multistep.StateBag,
) multistep.StepAction {
	ui, ok := state.Get("ui").(packersdk.Ui)
	if !ok {
		err := errStateTypeCastFailed
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if s.Comm.Type != "winrm" {
		log.Printf("[INFO] Not using winrm communicator, skipping get password...")
		return multistep.ActionContinue
	}

	if s.Comm.WinRMPassword != "" {
		ui.Say("Skipping waiting for password since WinRM password set...")
		return multistep.ActionContinue
	}

	var password string
	var err error
	cancel := make(chan struct{})
	waitDone := make(chan bool, 1)
	go func() {
		ui.Say("Waiting for auto-generated password for vm...")
		ui.Message(
			"It is normal for this process to take up to 15 minutes,\n" +
				"but it usually takes around 5. Please wait.")
		password, err = s.waitForPassword(ctx, state, cancel)
		waitDone <- true
	}()

	timeout := time.After(s.Timeout)
WaitLoop:
	for {
		select {
		case <-waitDone:
			if err != nil {
				ui.Error(fmt.Sprintf("Error waiting for password: %s", err))
				state.Put("error", err)
				return multistep.ActionHalt
			}

			ui.Message(" \nPassword retrieved!")
			s.Comm.WinRMPassword = password
			break WaitLoop
		case <-timeout:
			err = errTimeoutWaitingForPassword
			state.Put("error", err)
			ui.Error(err.Error())
			close(cancel)
			return multistep.ActionHalt
		case <-time.After(1 * time.Second):
			if _, ok := state.GetOk(multistep.StateCancelled); ok {
				close(cancel)
				log.Println("[WARN] Interrupt detected, quitting waiting for password.")
				return multistep.ActionHalt
			}
		}
	}

	if s.Debug {
		ui.Message(fmt.Sprintf(
			"Password (since debug is enabled): %s", s.Comm.WinRMPassword))
	}
	packersdk.LogSecretFilter.Set(s.Comm.WinRMPassword)

	return multistep.ActionContinue
}

// Cleanup is a no-op for this step.
func (s *StepGetPassword) Cleanup(_ multistep.StateBag) {}

func (s *StepGetPassword) waitForPassword( //nolint:gocyclo // polling loop with cancellation, state reads, and decryption steps
	ctx context.Context,
	state multistep.StateBag,
	cancel <-chan struct{},
) (string, error) {
	client, ok := state.Get("client").(*numspot.NumspotClient)
	if !ok {
		return "", errStateTypeCastFailed
	}
	vm, ok := state.Get("vm").(numspot.Vm)
	if !ok {
		return "", errStateTypeCastFailed
	}
	spaceId, ok := state.Get("space_id").(string)
	if !ok {
		return "", errStateTypeCastFailed
	}
	privateKey := s.Comm.SSHPrivateKey

	if vm.Id == nil {
		return "", errVMIDIsNil
	}

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		return "", fmt.Errorf("getting api client: %w", err)
	}

	spaceUuid, err := parseSpaceID(spaceId)
	if err != nil {
		return "", fmt.Errorf("parsing space_id: %w", err)
	}

	for {
		select {
		case <-cancel:
			log.Println("[INFO] Retrieve password wait cancelled. Exiting loop.")
			return "", errPasswordWaitCancelled
		case <-time.After(15 * time.Second):
		}

		resp, err := apiClient.ReadAdminPasswordWithResponse(ctx, spaceUuid, *vm.Id)
		if err != nil {
			return "", fmt.Errorf("error retrieving auto-generated vm password: %w", err)
		}

		if resp.JSON200 != nil && resp.JSON200.AdminPassword != nil &&
			*resp.JSON200.AdminPassword != "" {
			decryptedPassword, err := decryptPasswordDataWithPrivateKey(
				*resp.JSON200.AdminPassword, privateKey)
			if err != nil {
				return "", fmt.Errorf("error decrypting auto-generated vm password: %w", err)
			}

			return decryptedPassword, nil
		}

		log.Printf("[DEBUG] Password is blank, will retry...")
	}
}

func decryptPasswordDataWithPrivateKey(passwordData string, pemBytes []byte) (string, error) {
	encryptedPasswd, err := base64.StdEncoding.DecodeString(passwordData)
	if err != nil {
		return "", fmt.Errorf("decoding base64 password data: %w", err)
	}

	block, _ := pem.Decode(pemBytes)
	var asn1Bytes []byte
	if _, ok := block.Headers["DEK-Info"]; ok {
		return "", errEncryptedPrivateKeyNotSupported
	} else {
		asn1Bytes = block.Bytes
	}

	key, err := x509.ParsePKCS1PrivateKey(asn1Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing private key: %w", err)
	}

	//nolint:staticcheck // SA1019: Cloud providers encrypt Windows passwords with PKCS#1 v1.5.
	// We cannot change the encryption method used by the cloud API.
	out, err := rsa.DecryptPKCS1v15(nil, key, encryptedPasswd)
	if err != nil {
		return "", fmt.Errorf("decrypting password: %w", err)
	}

	return string(out), nil
}
