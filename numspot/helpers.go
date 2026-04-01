package numspot

import (
	"context"
	"fmt"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	defaultPollInterval = 10 * time.Second
	defaultTimeout      = 10 * time.Minute
)

// WaitUntilVMRunning polls until the VM reaches "running" state.
func WaitUntilVMRunning(
	ctx context.Context,
	client *ClientWithResponses,
	spaceID, vmID string,
	opts ...WaitOption,
) error {
	return waitUntilVMState(ctx, client, spaceID, vmID, "running", opts...)
}

// WaitUntilVMStopped polls until the VM reaches "stopped" state.
func WaitUntilVMStopped(
	ctx context.Context,
	client *ClientWithResponses,
	spaceID, vmID string,
	opts ...WaitOption,
) error {
	return waitUntilVMState(ctx, client, spaceID, vmID, "stopped", opts...)
}

// WaitUntilVMDeleted polls until the VM is deleted or terminated.
func WaitUntilVMDeleted(
	ctx context.Context,
	client *ClientWithResponses,
	spaceID, vmID string,
	opts ...WaitOption,
) error {
	config := newWaitConfig(opts...)
	ticker := time.NewTicker(config.interval)
	defer ticker.Stop()

	timeout := time.After(config.timeout)
	spaceUUID := openapi_types.UUID{}
	if err := spaceUUID.UnmarshalText([]byte(spaceID)); err != nil {
		return fmt.Errorf("invalid spaceId: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-timeout:
			return fmt.Errorf("%w: %s", errTimeoutWaitingForVMDeleted, vmID)
		case <-ticker.C:
			done, err := checkVMDeleted(ctx, client, spaceUUID, vmID)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

func checkVMDeleted(ctx context.Context, client *ClientWithResponses, spaceUUID openapi_types.UUID, vmID string) (bool, error) {
	resp, err := client.ReadVmsByIdWithResponse(ctx, spaceUUID, vmID)
	if err != nil {
		return false, fmt.Errorf("error checking VM status: %w", err)
	}
	if resp.StatusCode() == 404 {
		return true, nil
	}
	if resp.JSON200 != nil && resp.JSON200.State != nil && *resp.JSON200.State == "terminated" {
		return true, nil
	}
	return false, nil
}

func waitUntilVMState( //nolint:gocyclo // select loop with cancellation/timeout/state cases; || in nil-guard pushes count just above threshold
	ctx context.Context,
	client *ClientWithResponses,
	spaceID, vmID, targetState string,
	opts ...WaitOption,
) error {
	config := newWaitConfig(opts...)
	ticker := time.NewTicker(config.interval)
	defer ticker.Stop()

	timeout := time.After(config.timeout)
	spaceUUID := openapi_types.UUID{}
	if err := spaceUUID.UnmarshalText([]byte(spaceID)); err != nil {
		return fmt.Errorf("invalid spaceId: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-timeout:
			return fmt.Errorf("%w: %s -> %s", errTimeoutWaitingForVMState, vmID, targetState)
		case <-ticker.C:
			resp, err := client.ReadVmsByIdWithResponse(ctx, spaceUUID, vmID)
			if err != nil {
				return fmt.Errorf("error checking VM status: %w", err)
			}
			if resp.JSON200 == nil || resp.JSON200.State == nil {
				continue
			}
			done, err := checkVMStateReached(*resp.JSON200.State, targetState, vmID)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

func checkVMStateReached(state, targetState, vmID string) (bool, error) {
	if state == targetState {
		return true, nil
	}
	if state == "terminated" || state == "quarantine" {
		return false, fmt.Errorf("%w: %s is %s", errVMUnexpectedState, vmID, state)
	}
	return false, nil
}

// WaitUntilImageAvailable polls until the image reaches "available" state.
func WaitUntilImageAvailable( //nolint:gocyclo // select loop with cancellation/timeout/state cases; || in nil-guard pushes count just above threshold
	ctx context.Context,
	client *ClientWithResponses,
	spaceID, imageID string,
	opts ...WaitOption,
) error {
	config := newWaitConfig(opts...)
	ticker := time.NewTicker(config.interval)
	defer ticker.Stop()

	timeout := time.After(config.timeout)
	spaceUUID := openapi_types.UUID{}
	if err := spaceUUID.UnmarshalText([]byte(spaceID)); err != nil {
		return fmt.Errorf("invalid spaceId: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		case <-timeout:
			return fmt.Errorf("%w: %s", errTimeoutWaitingForImage, imageID)
		case <-ticker.C:
			resp, err := client.ReadImagesByIdWithResponse(ctx, spaceUUID, imageID)
			if err != nil {
				return fmt.Errorf("error checking image status: %w", err)
			}
			if resp.JSON200 == nil || resp.JSON200.State == nil {
				continue
			}
			done, err := checkImageAvailable(*resp.JSON200.State, imageID)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}

func checkImageAvailable(state, imageID string) (bool, error) {
	if state == "available" {
		return true, nil
	}
	if state == "error" {
		return false, fmt.Errorf("%w: %s", errImageEnteredErrorState, imageID)
	}
	return false, nil
}

// WaitConfig holds configuration for polling wait functions.
type WaitConfig struct {
	interval time.Duration
	timeout  time.Duration
}

// WaitOption is a functional option for configuring WaitConfig.
type WaitOption func(*WaitConfig)

// WithPollInterval sets the polling interval.
func WithPollInterval(d time.Duration) WaitOption {
	return func(c *WaitConfig) {
		c.interval = d
	}
}

// WithTimeout sets the maximum wait timeout.
func WithTimeout(d time.Duration) WaitOption {
	return func(c *WaitConfig) {
		c.timeout = d
	}
}

func newWaitConfig(opts ...WaitOption) *WaitConfig {
	config := &WaitConfig{
		interval: defaultPollInterval,
		timeout:  defaultTimeout,
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}
