package common

import (
	"context"
	"fmt"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// GetOwners returns the list of image owners as string pointers.
func (d *ImageFilterOptions) GetOwners() []*string {
	res := make([]*string, 0, len(d.Owners))
	for _, owner := range d.Owners {
		i := owner
		res = append(res, &i)
	}
	return res
}

// GetFilteredImage returns the image matching the configured filters.
func (d *ImageFilterOptions) GetFilteredImage( //nolint:gocyclo // multiple nil-guard checks and filter branches are all necessary
	ctx context.Context,
	client *numspot.NumspotClient,
	spaceID string,
) (*numspot.Image, error) {
	params := &numspot.ReadImagesParams{}

	if len(d.Filters) > 0 {
		params = buildNumspotImageParams(d.Filters)
	}

	if len(d.Owners) > 0 {
		accountAliases := make([]string, 0, len(d.Owners))
		accountAliases = append(accountAliases, d.Owners...)
		if len(accountAliases) > 0 {
			params.AccountAliases = &accountAliases
		}
	}

	spaceUUID := openapi_types.UUID{}
	if err := spaceUUID.UnmarshalText([]byte(spaceID)); err != nil {
		return nil, fmt.Errorf("invalid space_id: %w", err)
	}

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting client: %w", err)
	}

	imageResp, err := apiClient.ReadImagesWithResponse(ctx, spaceUUID, params)
	if err != nil {
		return nil, fmt.Errorf("error querying Image: %w", err)
	}

	if imageResp.JSON200 == nil ||
		imageResp.JSON200.Items == nil ||
		len(*imageResp.JSON200.Items) == 0 {
		return nil, errNoImageFoundMatchingFilters
	}

	if len(*imageResp.JSON200.Items) > 1 && !d.MostRecent {
		return nil, errMultipleImagesFound
	}

	var image numspot.Image
	if d.MostRecent {
		image = mostRecentImage(*imageResp.JSON200.Items)
	} else {
		image = (*imageResp.JSON200.Items)[0]
	}
	return &image, nil
}

//nolint:gocyclo // Function is a giant switch case
func buildNumspotImageParams(
	filters map[string]string,
) *numspot.ReadImagesParams {
	params := &numspot.ReadImagesParams{}

	for key, value := range filters {
		switch key {
		case "architecture":
			params.Architectures = &[]string{value}
		case "description":
			params.Descriptions = &[]string{value}
		case "image_name", "name":
			params.ImageNames = &[]string{value}
		case "image_id", "id":
			params.Ids = &[]string{value}
		case "state":
			params.States = &[]string{value}
		case "root_device_type":
			params.RootDeviceTypes = &[]string{value}
		case "root_device_name":
			params.RootDeviceNames = &[]string{value}
		case "hypervisor":
			params.Hypervisors = &[]string{value}
		case "product_code":
			params.ProductCodes = &[]string{value}
		case "product_code_name":
			params.ProductCodeNames = &[]string{value}
		case "file_location":
			params.FileLocations = &[]string{value}
		}
	}

	return params
}

func mostRecentImage(images []numspot.Image) numspot.Image {
	var mostRecent numspot.Image
	var mostRecentTime int64

	for _, image := range images {
		if image.CreationDate != nil {
			time := image.CreationDate.Unix()
			if time > mostRecentTime {
				mostRecentTime = time
				mostRecent = image
			}
		}
	}

	if mostRecent.Id == nil {
		if len(images) > 0 {
			return images[0]
		}
	}

	return mostRecent
}
