package common

import (
	"fmt"
	"log"
	"slices"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

// ImageConfig contains the configuration for creating a machine image.
type ImageConfig struct {
	// The name of the resulting image. Must be 3-128 characters and contain
	// only alphanumeric characters, hyphens, and underscores.
	ImageName string `mapstructure:"image_name" required:"true"`

	// The description of the resulting image.
	ImageDescription string `mapstructure:"image_description" required:"false"`

	// The list of account IDs that will have permission to launch the image.
	ImageAccountIDs []string `mapstructure:"image_account_ids" required:"false"`

	// The list of groups that will have permission to launch the image.
	ImageGroups []string `mapstructure:"image_groups" required:"false"`

	// The list of regions where the image will be copied.
	ImageRegions []string `mapstructure:"image_regions" required:"false"`

	// The boot modes supported by the image. Valid values: legacy, uefi.
	ImageBootModes []string `mapstructure:"image_boot_modes" required:"false"`

	// The product codes to associate with the image.
	ProductCodes []string `mapstructure:"product_codes" required:"false"`

	// Skip validation of image regions.
	SkipRegionValidation bool `mapstructure:"skip_region_validation" required:"false"`

	// Tags to apply to the resulting image.
	Tags TagMap `mapstructure:"tags" required:"false"`

	// Force deregister existing image with same name before creating new one.
	ForceDeregister bool `mapstructure:"force_deregister" required:"false"`

	// Force delete snapshots associated with existing image when deregistering.
	ForceDeleteSnapshot bool `mapstructure:"force_delete_snapshot" required:"false"`

	// Tags to apply to snapshots created during the image build.
	SnapshotTags TagMap `mapstructure:"snapshot_tags" required:"false"`

	// The list of account IDs that will have permission to launch the snapshots.
	SnapshotAccountIDs []string `mapstructure:"snapshot_account_ids" required:"false"`

	// Make the image public (global permission).
	GlobalPermission bool `mapstructure:"global_permission" required:"false"`

	// The root device name for the image.
	RootDeviceName string `mapstructure:"root_device_name" required:"false"`
}

// Prepare validates and prepares the ImageConfig.
func (c *ImageConfig) Prepare(accessConfig *AccessConfig, _ *interpolate.Context) []error {
	var errs []error

	if c.ImageName == "" {
		errs = append(errs, errImageNameRequired)
	}

	c.prepareRegions(accessConfig)

	if len(c.ImageName) < 3 || len(c.ImageName) > 128 {
		errs = append(errs, errImageNameLength)
	}

	if len(c.ImageBootModes) > 0 {
		bootModesSupported := []string{"legacy", "uefi"}
		for _, bootMode := range c.ImageBootModes {
			if !slices.Contains(bootModesSupported, bootMode) {
				errs = append(errs, fmt.Errorf("%w: %s", errImageBootModeUnsupported, bootMode))
			}
		}
	}

	if c.ImageName != templateCleanResourceName(c.ImageName) {
		errs = append(errs, errImageNameInvalidChars)
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// GetBootModes returns the boot modes configured for the image.
func (c *ImageConfig) GetBootModes() *[]string {
	if len(c.ImageBootModes) > 0 {
		return &c.ImageBootModes
	}
	return nil
}

func (c *ImageConfig) prepareRegions(accessConfig *AccessConfig) {
	if len(c.ImageRegions) > 0 {
		regionSet := make(map[string]struct{})
		regions := make([]string, 0, len(c.ImageRegions))

		for _, region := range c.ImageRegions {
			if _, ok := regionSet[region]; ok {
				continue
			}

			regionSet[region] = struct{}{}

			if accessConfig != nil && region == accessConfig.GetSpaceID() {
				log.Printf(
					"Cannot copy image to current space '%s', removing from image_regions",
					region,
				)
				continue
			}
			regions = append(regions, region)
		}

		c.ImageRegions = regions
	}
}
