//go:generate go run -modfile=../../go.mod github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc mapstructure-to-hcl2 -type DatasourceOutput,Config

// Package image provides a Packer datasource to look up an existing Numspot image.
package image

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/hcl2helper"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/zclconf/go-cty/cty"

	numspotcommon "github.com/numspot/numspot-plugin-packer/builder/common"
	"github.com/numspot/numspot-plugin-packer/numspot"
)

var errFiltersRequired = errors.New("at least one `filters` entry must be specified")

// Datasource implements the Packer datasource interface for Numspot images.
type Datasource struct {
	config Config
}

// Config holds the datasource configuration.
type Config struct {
	common.PackerConfig              `mapstructure:",squash"`
	numspotcommon.AccessConfig       `mapstructure:",squash"`
	numspotcommon.ImageFilterOptions `mapstructure:",squash"`

	ctx interpolate.Context
}

// ConfigSpec returns the HCL2 spec for the datasource configuration.
func (d *Datasource) ConfigSpec() hcldec.ObjectSpec {
	return d.config.FlatMapstructure().HCL2Spec()
}

// Configure decodes and validates the datasource configuration.
func (d *Datasource) Configure(raws ...interface{}) error {
	err := config.Decode(&d.config, nil, raws...)
	if err != nil {
		return fmt.Errorf("decoding datasource config: %w", err)
	}

	var errs *packersdk.MultiError
	errs = packersdk.MultiErrorAppend(errs, d.config.Prepare(&d.config.ctx)...)

	if d.config.NameValueFilter.Empty() && len(d.config.Owners) == 0 {
		errs = packersdk.MultiErrorAppend(errs, errFiltersRequired)
	}

	if errs != nil && len(errs.Errors) > 0 {
		return errs
	}
	return nil
}

// DatasourceOutput holds the values returned by the datasource.
type DatasourceOutput struct {
	// The ID of the image.
	ID string `mapstructure:"id"`
	// The name of the image.
	Name string `mapstructure:"name"`
	// The description of the image.
	Description string `mapstructure:"description"`
	// The date and time of creation of the image, in ISO 8601 format.
	CreationDate string `mapstructure:"creation_date"`
	// The state of the image (pending | available | failed).
	State string `mapstructure:"state"`
	// The architecture of the image.
	Architecture string `mapstructure:"architecture"`
	// The name of the root device.
	RootDeviceName string `mapstructure:"root_device_name"`
	// The type of root device (always bsu).
	RootDeviceType string `mapstructure:"root_device_type"`
	// The key/value combination of the tags assigned to the image.
	Tags map[string]string `mapstructure:"tags"`
}

// OutputSpec returns the HCL2 spec for the datasource output.
func (d *Datasource) OutputSpec() hcldec.ObjectSpec {
	return (&DatasourceOutput{}).FlatMapstructure().HCL2Spec()
}

// Execute queries the Numspot API and returns the matching image.
func (d *Datasource) Execute() (cty.Value, error) {
	ctx := context.Background()

	client, err := d.config.NewNumspotClient(ctx)
	if err != nil {
		return cty.NullVal(cty.EmptyObject), fmt.Errorf("creating numspot client: %w", err)
	}

	img, err := d.config.GetFilteredImage(ctx, client, d.config.GetSpaceID())
	if err != nil {
		return cty.NullVal(cty.EmptyObject), err
	}

	output := buildDatasourceOutput(img)
	return hcl2helper.HCL2ValueFromConfig(output, d.OutputSpec()), nil
}

func buildDatasourceOutput(img *numspot.Image) DatasourceOutput { //nolint:gocyclo // nil-guard checks for each optional image field are all necessary
	output := DatasourceOutput{
		Tags: make(map[string]string),
	}
	if img.Tags != nil {
		for _, tag := range *img.Tags {
			output.Tags[tag.Key] = tag.Value
		}
	}
	if img.Id != nil {
		output.ID = *img.Id
	}
	if img.Name != nil {
		output.Name = *img.Name
	}
	if img.Description != nil {
		output.Description = *img.Description
	}
	if img.CreationDate != nil {
		output.CreationDate = img.CreationDate.String()
	}
	if img.State != nil {
		output.State = *img.State
	}
	if img.Architecture != nil {
		output.Architecture = *img.Architecture
	}
	if img.RootDeviceName != nil {
		output.RootDeviceName = *img.RootDeviceName
	}
	if img.RootDeviceType != nil {
		output.RootDeviceType = *img.RootDeviceType
	}
	return output
}
