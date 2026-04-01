package common

import (
	"github.com/hashicorp/packer-plugin-sdk/multistep"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// BuildInfoTemplate contains template variables for build information.
type BuildInfoTemplate struct {
	BuildRegion     string
	SourceImage     string
	SourceImageName string
	SourceImageTags map[string]string
}

func extractBuildInfo(region string, state multistep.StateBag) *BuildInfoTemplate {
	rawSourceImage, hasSourceImage := state.GetOk("source_image")
	if !hasSourceImage {
		return &BuildInfoTemplate{
			BuildRegion: region,
		}
	}

	sourceImage, ok := rawSourceImage.(numspot.Image)
	if !ok {
		return &BuildInfoTemplate{
			BuildRegion: region,
		}
	}
	sourceImageTags := make(map[string]string)
	if sourceImage.Tags != nil {
		for _, tag := range *sourceImage.Tags {
			sourceImageTags[tag.Key] = tag.Value
		}
	}

	imageID := ""
	if sourceImage.Id != nil {
		imageID = *sourceImage.Id
	}

	imageName := ""
	if sourceImage.Name != nil {
		imageName = *sourceImage.Name
	}

	return &BuildInfoTemplate{
		BuildRegion:     region,
		SourceImage:     imageID,
		SourceImageName: imageName,
		SourceImageTags: sourceImageTags,
	}
}
