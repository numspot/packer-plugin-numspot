//go:generate go run -modfile=../../go.mod github.com/hashicorp/packer-plugin-sdk/cmd/packer-sdc mapstructure-to-hcl2 -type Config

package bsu

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"

	numspotcommon "github.com/numspot/numspot-plugin-packer/builder/common"
)

// BuilderID is the unique identifier for the BSU builder.
const BuilderID = "numspot.bsu"

var (
	errMissingRootDeviceName = errors.New(
		"missing root_device_name for image_block_device_mappings",
	)
	errNoArtifactCreated      = errors.New("no artifact was created")
	errImageNoIDReturned      = errors.New("error creating Image: no Id returned")
	errUnknownWaitingForImage = errors.New("unknown error waiting for Image")
	errImageNoImageReturned   = errors.New("error while reading the image: no image returned")
	errImageEmptyID           = errors.New("error while reading an empty image id")
	errImageWaitingForReason  = errors.New("error waiting for Image")
)

// Config is the configuration for the BSU builder.
type Config struct {
	common.PackerConfig        `mapstructure:",squash"`
	numspotcommon.AccessConfig `mapstructure:",squash"`
	numspotcommon.ImageConfig  `mapstructure:",squash"`
	numspotcommon.BlockDevices `mapstructure:",squash"`
	numspotcommon.RunConfig    `mapstructure:",squash"`

	// Tags to apply to volumes created during the build.
	VolumeRunTags numspotcommon.TagMap `mapstructure:"run_volume_tags" required:"false"`

	// Skip creating the image after provisioning the VM.
	SkipCreateImage bool `mapstructure:"skip_create_image" required:"false"`

	ctx interpolate.Context
}

// Builder implements the packer.Builder interface for BSU-backed images.
type Builder struct {
	config Config
	runner multistep.Runner
}

// ConfigSpec returns the HCL2 spec for the config.
func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

// Prepare prepares the builder configuration.
func (b *Builder) Prepare(raws ...interface{}) (warnings, deprecated []string, err error) {
	b.config.ctx.Funcs = numspotcommon.TemplateFuncs
	err = config.Decode(&b.config, &config.DecodeOpts{
		PluginType:         BuilderID,
		Interpolate:        true,
		InterpolateContext: &b.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"image_description",
				"run_tags",
				"run_volume_tags",
				"snapshot_tags",
				"tags",
			},
		},
	}, raws...)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding config: %w", err)
	}

	if b.config.PackerForce {
		b.config.ForceDeregister = true
	}

	var errs *packersdk.MultiError
	errs = packersdk.MultiErrorAppend(errs, b.config.AccessConfig.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs,
		b.config.ImageConfig.Prepare(&b.config.AccessConfig, &b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.BlockDevices.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.RunConfig.Prepare(&b.config.ctx)...)

	if errs != nil && len(errs.Errors) > 0 {
		return nil, nil, errs
	}

	return nil, nil, nil
}

// Run executes the build process.
func (b *Builder) Run(
	ctx context.Context,
	ui packersdk.Ui,
	hook packersdk.Hook,
) (packersdk.Artifact, error) {
	client, err := b.config.NewNumspotClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating numspot client: %w", err)
	}

	spaceID := b.config.GetSpaceID()

	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("client", client)
	state.Put("space_id", spaceID)
	state.Put("accessConfig", &b.config.AccessConfig)
	state.Put("hook", hook)
	state.Put("ui", ui)

	steps := []multistep.Step{
		&numspotcommon.StepPreValidate{
			DestImageName:   b.config.ImageName,
			ForceDeregister: b.config.ForceDeregister,
		},
		&numspotcommon.StepSourceImageInfo{
			SourceImage:  b.config.SourceImage,
			ImageFilters: b.config.SourceImageFilter,
		},
		&numspotcommon.StepNetworkInfo{
			NetID:                b.config.NetID,
			NetFilter:            b.config.NetFilter,
			SecurityGroupIDs:     b.config.SecurityGroupIDs,
			SecurityGroupFilter:  b.config.SecurityGroupFilter,
			SubnetID:             b.config.SubnetID,
			SubnetFilter:         b.config.SubnetFilter,
			AvailabilityZoneName: b.config.AvailabilityZone,
		},
		&numspotcommon.StepKeyPair{
			Debug:        b.config.PackerDebug,
			Comm:         &b.config.Comm,
			DebugKeyPath: fmt.Sprintf("numspot_%s", b.config.PackerBuildName),
		},
		&numspotcommon.StepPublicIp{
			AssociatePublicIpAddress: b.config.AssociatePublicIPAddress,
			Debug:                    b.config.PackerDebug,
		},
		&numspotcommon.StepSecurityGroup{
			SecurityGroupFilter:   b.config.SecurityGroupFilter,
			SecurityGroupIDs:      b.config.SecurityGroupIDs,
			CommConfig:            &b.config.Comm,
			TemporarySGSourceCidr: b.config.TemporarySGSourceCidr,
		},
		&numspotcommon.StepCleanupVolumes{
			BlockDevices: b.config.BlockDevices,
		},
		&numspotcommon.StepRunSourceVm{
			BlockDevices:                b.config.BlockDevices,
			Comm:                        &b.config.Comm,
			Ctx:                         b.config.ctx,
			Debug:                       b.config.PackerDebug,
			BsuOptimized:                b.config.BsuOptimized,
			EnableT2Unlimited:           b.config.EnableT2Unlimited,
			ExpectedRootDevice:          numspotcommon.RunSourceVmBSUExpectedRootDevice,
			IamVmProfile:                b.config.IamVMProfile,
			VmInitiatedShutdownBehavior: b.config.VMInitiatedShutdownBehavior,
			VmType:                      b.config.VmType,
			IsRestricted:                false,
			SourceImage:                 b.config.SourceImage,
			Tags:                        b.config.RunTags,
			UserData:                    b.config.UserData,
			UserDataFile:                b.config.UserDataFile,
			VolumeTags:                  b.config.VolumeRunTags,
			RawRegion:                   spaceID,
		},
		&numspotcommon.StepGetPassword{
			Debug:     b.config.PackerDebug,
			Comm:      &b.config.Comm,
			Timeout:   b.config.WindowsPasswordTimeout,
			BuildName: b.config.PackerBuildName,
		},
		&communicator.StepConnect{
			Config: &b.config.Comm,
			Host: numspotcommon.SSHHost(
				ctx,
				client,
				spaceID,
				b.config.SSHInterface),
			SSHConfig: b.config.Comm.SSHConfigFunc(),
		},
		&commonsteps.StepProvision{},
		&commonsteps.StepCleanupTempKeys{
			Comm: &b.config.Comm,
		},
		&numspotcommon.StepStopBSUBackedVM{
			Skip:          false,
			DisableStopVM: b.config.DisableStopVM,
		},
		&numspotcommon.StepDeregisterImage{
			AccessConfig:        &b.config.AccessConfig,
			ForceDeregister:     b.config.ForceDeregister,
			ForceDeleteSnapshot: b.config.ForceDeleteSnapshot,
			ImageName:           b.config.ImageName,
		},
	}

	if !b.config.SkipCreateImage {
		bootModes := b.config.GetBootModes()
		if bootModes == nil {
			bootModes = &[]string{}
		}
		steps = append(steps,
			&stepCreateImage{
				RawRegion:    spaceID,
				ProductCodes: b.config.ProductCodes,
				BootModes:    *bootModes,
			},
			&numspotcommon.StepUpdateImageAttributes{
				AccountIDs:         b.config.ImageAccountIDs,
				SnapshotAccountIDs: b.config.SnapshotAccountIDs,
				RawRegion:          spaceID,
				GlobalPermission:   b.config.GlobalPermission,
				Ctx:                b.config.ctx,
			},
			&numspotcommon.StepCreateTags{
				Tags:         b.config.Tags,
				SnapshotTags: b.config.SnapshotTags,
				Ctx:          b.config.ctx,
			})
	}

	b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, state)

	if rawErr, ok := state.GetOk("error"); ok {
		if buildErr, ok := rawErr.(error); ok {
			return nil, buildErr
		}
		return nil, errNoArtifactCreated
	}

	if images, ok := state.GetOk("images"); ok {
		imagesMap, ok := images.(map[string]string)
		if !ok {
			return nil, errNoArtifactCreated
		}
		artifact := &numspotcommon.Artifact{
			Images:         imagesMap,
			BuilderIDValue: BuilderID,
			StateData:      map[string]interface{}{"generated_data": state.Get("generated_data")},
		}

		return artifact, nil
	}

	return nil, errNoArtifactCreated
}
