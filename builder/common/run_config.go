//go:generate go run -modfile=../../go.mod github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc mapstructure-to-hcl2 -type SecurityGroupFilterOptions,ImageFilterOptions,SubnetFilterOptions,NetFilterOptions,BlockDevice

package common

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

var (
	stopShutdownBehavior      = "stop"
	terminateShutdownBehavior = "terminate"
	reShutdownBehavior        = regexp.MustCompile(
		"^(" + stopShutdownBehavior + "|" + terminateShutdownBehavior + ")$",
	)
)

// ImageFilterOptions contains options for filtering images.
type ImageFilterOptions struct {
	config.NameValueFilter `         mapstructure:",squash"`

	// The list of account aliases (owners) to filter images by.
	Owners []string `mapstructure:"owners" required:"false"`

	// Select the most recent image when multiple images match.
	MostRecent bool `mapstructure:"most_recent" required:"false"`
}

// Empty returns true if no filters are set.
func (d *ImageFilterOptions) Empty() bool {
	return len(d.Owners) == 0 && d.NameValueFilter.Empty()
}

// NoOwner returns true if no owners are specified.
func (d *ImageFilterOptions) NoOwner() bool {
	return len(d.Owners) == 0
}

// SubnetFilterOptions contains options for filtering subnets.
type SubnetFilterOptions struct {
	config.NameValueFilter `     mapstructure:",squash"`

	// Select the subnet with the most free IP addresses.
	MostFree bool `mapstructure:"most_free" required:"false"`

	// Select a random subnet from matching results.
	Random bool `mapstructure:"random" required:"false"`
}

// NetFilterOptions contains options for filtering networks.
type NetFilterOptions struct {
	config.NameValueFilter `mapstructure:",squash"`
}

// SecurityGroupFilterOptions contains options for filtering security groups.
type SecurityGroupFilterOptions struct {
	config.NameValueFilter `mapstructure:",squash"`
}

// RunConfig contains the configuration for running a VM.
type RunConfig struct {
	// Associate a public IP address to the VM.
	AssociatePublicIPAddress bool `mapstructure:"associate_public_ip_address" required:"false"`

	// The availability zone name.
	AvailabilityZone string `mapstructure:"availability_zone" required:"false"`

	// Block duration in minutes for spot instances.
	BlockDurationMinutes int64 `mapstructure:"block_duration_minutes" required:"false"`

	// Disable stopping the VM before creating the image.
	DisableStopVM bool `mapstructure:"disable_stop_vm" required:"false"`

	// Enable BSU optimization.
	BsuOptimized bool `mapstructure:"bsu_optimized" required:"false"`

	// Enable T2 unlimited for burstable instances.
	EnableT2Unlimited bool `mapstructure:"enable_t2_unlimited" required:"false"`

	// The IAM VM profile to attach.
	IamVMProfile string `mapstructure:"iam_vm_profile" required:"false"`

	// The shutdown behavior for the VM. Valid values: stop, terminate.
	VMInitiatedShutdownBehavior string `mapstructure:"shutdown_behavior" required:"false"`

	// The VM type to launch.
	VmType string `mapstructure:"vm_type" required:"true"`

	// Filter to find a security group.
	SecurityGroupFilter SecurityGroupFilterOptions `mapstructure:"security_group_filter" required:"false"`

	// Tags to apply to the VM during the build.
	RunTags map[string]string `mapstructure:"run_tags" required:"false"`

	// The security group ID to use (deprecated, use security_group_ids).
	SecurityGroupID string `mapstructure:"security_group_id" required:"false"`

	// The list of security group IDs to attach to the VM.
	SecurityGroupIDs []string `mapstructure:"security_group_ids" required:"false"`

	// The source image ID to use for the VM.
	SourceImage string `mapstructure:"source_image" required:"true"`

	// Filter to find a source image.
	SourceImageFilter ImageFilterOptions `mapstructure:"source_image_filter" required:"false"`

	// Filter to find a subnet.
	SubnetFilter SubnetFilterOptions `mapstructure:"subnet_filter" required:"false"`

	// The subnet ID to launch the VM in.
	SubnetID string `mapstructure:"subnet_id" required:"false"`

	// The CIDR for the temporary security group source.
	TemporarySGSourceCidr string `mapstructure:"temporary_security_group_source_cidr" required:"false"`

	// User data to pass to the VM (base64 encoded).
	UserData string `mapstructure:"user_data" required:"false"`

	// Path to a file containing user data.
	UserDataFile string `mapstructure:"user_data_file" required:"false"`

	// Filter to find a VPC/Net.
	NetFilter NetFilterOptions `mapstructure:"net_filter" required:"false"`

	// The VPC/Net ID to launch the VM in.
	NetID string `mapstructure:"net_id" required:"false"`

	// Timeout for retrieving Windows password.
	WindowsPasswordTimeout time.Duration `mapstructure:"windows_password_timeout" required:"false"`

	// The boot mode for the VM. Valid values: legacy, uefi.
	BootMode string `mapstructure:"boot_mode" required:"false"`

	Comm         communicator.Config `mapstructure:",squash"`
	SSHInterface string              `mapstructure:"ssh_interface" required:"false"`
}

// Prepare validates configuration fields from a RunConfig
//
//nolint:gocognit,gocyclo // large validation function covering many independent config fields
func (c *RunConfig) Prepare(
	ctx *interpolate.Context,
) []error {
	if c.Comm.SSHKeyPairName == "" && c.Comm.SSHTemporaryKeyPairName == "" &&
		c.Comm.SSHPrivateKeyFile == "" && c.Comm.SSHPassword == "" {

		c.Comm.SSHTemporaryKeyPairName = fmt.Sprintf("pk-%d", time.Now().Unix())
	}

	if c.WindowsPasswordTimeout == 0 {
		c.WindowsPasswordTimeout = 20 * time.Minute
	}

	if c.Comm.SSHTimeout == 0 {
		c.Comm.SSHTimeout = 10 * time.Minute
	}

	if c.RunTags == nil {
		c.RunTags = make(map[string]string)
	}
	errs := c.Comm.Prepare(ctx)

	for _, preparer := range []interface{ Prepare() []error }{
		&c.SourceImageFilter,
		&c.SecurityGroupFilter,
		&c.SubnetFilter,
		&c.NetFilter,
	} {
		errs = append(errs, preparer.Prepare()...)
	}

	if c.SSHInterface != "public_ip" &&
		c.SSHInterface != "private_ip" &&
		c.SSHInterface != "public_dns" &&
		c.SSHInterface != "private_dns" &&
		c.SSHInterface != "" {
		errs = append(errs, fmt.Errorf("%w: %s", errUnknownSSHInterface, c.SSHInterface))
	}

	if c.Comm.SSHKeyPairName != "" {
		if c.Comm.Type == "winrm" && c.Comm.WinRMPassword == "" && c.Comm.SSHPrivateKeyFile == "" {
			errs = append(
				errs,
				errSSHPrivateKeyRequiredForWinRM,
			)
		} else if c.Comm.SSHPrivateKeyFile == "" && !c.Comm.SSHAgentAuth {
			errs = append(errs, errSSHPrivateKeyOrAgentRequired)
		}
	}
	if c.BootMode != "" {
		bootModesSupported := []string{"legacy", "uefi"}
		found := false
		for _, mode := range bootModesSupported {
			if c.BootMode == mode {
				found = true
				break
			}
		}
		if !found {
			errs = append(
				errs,
				fmt.Errorf("%w: %v", errVMBootModeUnsupported, c.BootMode),
			)
		}
	}
	if c.SourceImage == "" && c.SourceImageFilter.Empty() {
		errs = append(errs, errSourceImageRequired)
	}

	if c.SourceImage == "" && c.SourceImageFilter.NoOwner() {
		errs = append(
			errs,
			errSourceAMIFilterOwnerRequired,
		)
	}

	if c.VmType == "" {
		errs = append(errs, errVMTypeRequired)
	}

	if c.BlockDurationMinutes%60 != 0 {
		errs = append(errs, errBlockDurationMultipleOf60)
	}

	if c.UserData != "" && c.UserDataFile != "" {
		errs = append(errs, errUserDataConflict)
	} else if c.UserDataFile != "" {
		if _, err := os.Stat(c.UserDataFile); err != nil {
			errs = append(errs, fmt.Errorf("%w: %s", errUserDataFileNotFound, c.UserDataFile))
		}
	}

	if c.SecurityGroupID != "" {
		if len(c.SecurityGroupIDs) > 0 {
			errs = append(
				errs,
				errSecurityGroupIDConflict,
			)
		} else {
			c.SecurityGroupIDs = []string{c.SecurityGroupID}
			c.SecurityGroupID = ""
		}
	}

	if c.TemporarySGSourceCidr == "" {
		c.TemporarySGSourceCidr = "0.0.0.0/0"
	} else {
		if _, _, err := net.ParseCIDR(c.TemporarySGSourceCidr); err != nil {
			errs = append(
				errs,
				fmt.Errorf("error parsing temporary_security_group_source_cidr: %w", err),
			)
		}
	}

	if c.VMInitiatedShutdownBehavior == "" {
		c.VMInitiatedShutdownBehavior = stopShutdownBehavior
	} else if !reShutdownBehavior.MatchString(c.VMInitiatedShutdownBehavior) {
		errs = append(errs, errShutdownBehaviorInvalid)
	}

	if c.EnableT2Unlimited {
		firstDotIndex := strings.Index(c.VmType, ".")
		if firstDotIndex == -1 {
			errs = append(errs, fmt.Errorf("%w: %s", errUnknownVMType, c.VmType))
		} else if c.VmType[0:firstDotIndex] != "t2" {
			errs = append(
				errs,
				fmt.Errorf("%w: %s", errT2UnlimitedNonT2, c.VmType),
			)
		}
	}

	return errs
}
