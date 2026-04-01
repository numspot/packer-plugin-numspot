package common

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	registryimage "github.com/hashicorp/packer-plugin-sdk/packer/registry/image"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Artifact represents the result of a build.
type Artifact struct {
	Images         map[string]string
	BuilderIDValue string
	StateData      map[string]interface{}
}

// BuilderId returns the builder Id.
func (a *Artifact) BuilderId() string {
	return a.BuilderIDValue
}

// Files returns the list of files created by the artifact.
func (*Artifact) Files() []string {
	return nil
}

// Id returns the artifact Id.
func (a *Artifact) Id() string {
	parts := make([]string, 0, len(a.Images))
	for region, imageID := range a.Images {
		parts = append(parts, fmt.Sprintf("%s:%s", region, imageID))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// String returns a string representation of the artifact.
func (a *Artifact) String() string {
	imageStrings := make([]string, 0, len(a.Images))
	for region, id := range a.Images {
		single := fmt.Sprintf("%s: %s", region, id)
		imageStrings = append(imageStrings, single)
	}
	sort.Strings(imageStrings)
	return fmt.Sprintf("Images were created:\n%s\n", strings.Join(imageStrings, "\n"))
}

// State returns the state value for the given name.
func (a *Artifact) State(name string) interface{} {
	if _, ok := a.StateData[name]; ok {
		return a.StateData[name]
	}

	switch name {
	case "atlas.artifact.metadata":
		return a.stateAtlasMetadata()
	case registryimage.ArtifactStateURI:
		return a.stateHCPPackerRegistryMetadata()
	default:
		return nil
	}
}

// Destroy cleans up the artifact by deregistering images.
func (a *Artifact) Destroy() error {
	errs := make([]error, 0)

	config, ok := a.State("accessConfig").(*AccessConfig)
	if !ok {
		return errStateTypeCastFailed
	}
	spaceId := config.GetSpaceID()
	ctx := context.Background()

	client, err := config.NewNumspotClient(ctx)
	if err != nil {
		return err
	}

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("client.GetClient: %w", err)
	}

	for _, imageID := range a.Images {
		log.Printf("Deregistering image Id (%s)", imageID)

		var spaceUuid openapi_types.UUID
		if err := spaceUuid.UnmarshalText([]byte(spaceId)); err != nil {
			errs = append(errs, fmt.Errorf("invalid space_id: %w", err))
			continue
		}

		_, err := apiClient.DeleteImageWithResponse(ctx, spaceUuid, imageID)
		if err != nil {
			errs = append(errs, fmt.Errorf("deleting image %q: %w", imageID, err))
		}
	}

	if len(errs) > 0 {
		if len(errs) == 1 {
			return errs[0]
		}
		return &packersdk.MultiError{Errors: errs}
	}

	return nil
}

func (a *Artifact) stateAtlasMetadata() interface{} {
	metadata := make(map[string]string)
	for region, imageID := range a.Images {
		k := fmt.Sprintf("region.%s", region)
		metadata[k] = imageID
	}
	return metadata
}

func (a *Artifact) stateHCPPackerRegistryMetadata() interface{} {
	f := func(k, v interface{}) (*registryimage.Image, error) {
		region, ok := k.(string)
		if !ok {
			return nil, errUnexpectedKeyInImagesMap
		}
		imageId, ok := v.(string)
		if !ok {
			return nil, errUnexpectedValueInImagesMap
		}
		image := registryimage.Image{
			ImageID:        imageId,
			ProviderRegion: region,
			ProviderName:   "numspot",
		}
		return &image, nil
	}

	images, err := registryimage.FromMappedData(a.Images, f)
	if err != nil {
		log.Printf("[TRACE] error creating HCP Packer registry image: %s", err)
		return nil
	}

	if a.StateData == nil {
		return images
	}

	data, ok := a.StateData["generated_data"].(map[string]interface{})
	if !ok {
		return images
	}

	for _, image := range images {
		if sourceImage, ok := data["SourceImage"].(string); ok {
			image.SourceImageID = sourceImage
		}
	}

	return images
}
