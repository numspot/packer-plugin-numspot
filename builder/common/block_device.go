package common

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// BlockDevice represents a block device mapping configuration.
type BlockDevice struct {
	// Whether the volume should be deleted on VM termination.
	DeleteOnVMDeletion bool `mapstructure:"delete_on_vm_deletion" required:"false"`

	// The device name exposed to the VM.
	DeviceName string `mapstructure:"device_name" required:"false"`

	// The number of IOPS for io1 volume type.
	IOPS int `mapstructure:"iops" required:"false"`

	// Suppress the specified device.
	NoDevice bool `mapstructure:"no_device" required:"false"`

	// The snapshot ID to create the volume from.
	SnapshotID string `mapstructure:"snapshot_id" required:"false"`

	// The volume type. Valid values: standard, gp2, io1.
	VolumeType string `mapstructure:"volume_type" required:"false"`

	// The size of the volume in GiB.
	VolumeSize int64 `mapstructure:"volume_size" required:"false"`
}

// BlockDevices holds both image and launch block device mappings.
type BlockDevices struct {
	ImageBlockDevices  `mapstructure:",squash"`
	LaunchBlockDevices `mapstructure:",squash"`
}

// ImageBlockDevices holds the image block device mapping configuration.
type ImageBlockDevices struct {
	ImageMappings []BlockDevice `mapstructure:"image_block_device_mappings"`
}

// LaunchBlockDevices holds the launch block device mapping configuration.
type LaunchBlockDevices struct {
	LaunchMappings []BlockDevice `mapstructure:"launch_block_device_mappings"`
}

func setBsuToCreate(blockDevice *BlockDevice) *numspot.BsuToCreate {
	defaultDeleteOnVMDeletion := true
	bsu := &numspot.BsuToCreate{
		DeleteOnVmDeletion: &defaultDeleteOnVMDeletion,
	}
	if deleteOnVmDeletion := blockDevice.DeleteOnVMDeletion; !deleteOnVmDeletion {
		bsu.DeleteOnVmDeletion = &deleteOnVmDeletion
	}
	if volType := blockDevice.VolumeType; volType != "" {
		bsu.VolumeType = &volType
	}
	if volSize := int(blockDevice.VolumeSize); volSize > 0 {
		bsu.VolumeSize = &volSize
	}
	if blockDevice.VolumeType == "io1" {
		bsu.Iops = &blockDevice.IOPS
	}
	if snapID := blockDevice.SnapshotID; snapID != "" {
		bsu.SnapshotId = &snapID
	}

	return bsu
}

func buildNumspotBlockDevicesImage(b []BlockDevice) []numspot.BlockDeviceMappingImage {
	blockDevices := make([]numspot.BlockDeviceMappingImage, 0, len(b))
	for i := range b {
		mapping := numspot.BlockDeviceMappingImage{}

		if deviceName := b[i].DeviceName; deviceName != "" {
			mapping.DeviceName = &deviceName
		}
		mapping.Bsu = setBsuToCreate(&b[i])
		blockDevices = append(blockDevices, mapping)
	}
	return blockDevices
}

func buildNumspotBlockDevicesVMCreation(b []BlockDevice) []numspot.BlockDeviceMappingVmCreation {
	blockDevices := make([]numspot.BlockDeviceMappingVmCreation, 0, len(b))
	for i := range b {
		mapping := numspot.BlockDeviceMappingVmCreation{}

		if deviceName := b[i].DeviceName; deviceName != "" {
			mapping.DeviceName = &deviceName
		}

		if b[i].NoDevice {
			mapping.NoDevice = aws.String("")
		} else {
			mapping.Bsu = setBsuToCreate(&b[i])
		}
		blockDevices = append(blockDevices, mapping)
	}
	return blockDevices
}

// Prepare validates the BlockDevice configuration.
func (b *BlockDevice) Prepare(_ *interpolate.Context) error {
	if b.DeviceName == "" {
		return errDeviceNameRequired
	}
	return nil
}

// Prepare validates all block device mappings.
func (b *BlockDevices) Prepare(ctx *interpolate.Context) (errs []error) {
	for _, d := range b.ImageMappings {
		if err := d.Prepare(ctx); err != nil {
			errs = append(errs, fmt.Errorf("ImageMapping: %w", err))
		}
	}
	for _, d := range b.LaunchMappings {
		if err := d.Prepare(ctx); err != nil {
			errs = append(errs, fmt.Errorf("LaunchMapping: %w", err))
		}
	}
	return errs
}

// BuildNumspotImageDevices returns the image block device mappings in Numspot format.
func (b *ImageBlockDevices) BuildNumspotImageDevices() []numspot.BlockDeviceMappingImage {
	return buildNumspotBlockDevicesImage(b.ImageMappings)
}

// BuildNumspotLaunchDevices returns the launch block device mappings in Numspot format.
func (b *LaunchBlockDevices) BuildNumspotLaunchDevices() []numspot.BlockDeviceMappingVmCreation {
	return buildNumspotBlockDevicesVMCreation(b.LaunchMappings)
}
