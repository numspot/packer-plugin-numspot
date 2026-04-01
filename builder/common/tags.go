package common

import (
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

// TagMap is a map of tag keys to values.
type TagMap map[string]string

// IsSet returns true if any tags are set.
func (t TagMap) IsSet() bool {
	return len(t) > 0
}

// ToNumspotTags converts TagMap to a slice of numspot tags.
func (t TagMap) ToNumspotTags(
	ctx *interpolate.Context,
	state multistep.StateBag,
) ([]numspot.Tag, error) {
	var tags []numspot.Tag
	ctx.Data = extractBuildInfo("", state)

	for key, value := range t {
		interpolatedKey, err := interpolate.Render(key, ctx)
		if err != nil {
			return nil, fmt.Errorf("error processing tag: %s:%s - %w", key, value, err)
		}
		interpolatedValue, err := interpolate.Render(value, ctx)
		if err != nil {
			return nil, fmt.Errorf("error processing tag: %s:%s - %w", key, value, err)
		}
		tags = append(tags, numspot.Tag{
			Key:   &interpolatedKey,
			Value: &interpolatedValue,
		})
	}
	return tags, nil
}

// Report logs all tags to the UI.
func (t TagMap) Report(ui packersdk.Ui) {
	for key, value := range t {
		ui.Message(fmt.Sprintf("Adding tag: %q: %q", key, value))
	}
}
