package common

import (
	"reflect"
	"sort"
	"testing"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	registryimage "github.com/hashicorp/packer-plugin-sdk/packer/registry/image"
	"github.com/mitchellh/mapstructure"
)

const testImageID = "foo"

func TestArtifact_Impl(_ *testing.T) {
	var _ packersdk.Artifact = new(Artifact)
}

func TestArtifactId(t *testing.T) {
	expected := `east:foo,west:bar`

	images := make(map[string]string)
	images["east"] = testImageID
	images["west"] = "bar"

	a := &Artifact{
		Images: images,
	}

	result := a.Id()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}
}

func TestArtifactState_atlasMetadata(t *testing.T) {
	a := &Artifact{
		Images: map[string]string{
			"east": testImageID,
			"west": "bar",
		},
	}

	actual := a.State("atlas.artifact.metadata")
	expected := map[string]string{
		"region.east": testImageID,
		"region.west": "bar",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestArtifactString(t *testing.T) {
	expected := `Images were created:
east: foo
west: bar
`

	images := make(map[string]string)
	images["east"] = testImageID
	images["west"] = "bar"

	a := &Artifact{Images: images}
	result := a.String()
	if result != expected {
		t.Fatalf("bad: %s", result)
	}
}

func TestArtifactState(t *testing.T) {
	expectedData := "this is the data"
	artifact := &Artifact{
		StateData: map[string]interface{}{"state_data": expectedData},
	}

	result := artifact.State("state_data")
	if result != expectedData {
		t.Fatalf("Bad: State data was %s instead of %s", result, expectedData)
	}

	result = artifact.State("invalid_key")
	if result != nil {
		t.Fatalf("Bad: State should be nil for invalid state data name")
	}

	artifact = &Artifact{}
	result = artifact.State("key")
	if result != nil {
		t.Fatalf("Bad: State should be nil for nil StateData")
	}
}

func TestArtifactState_hcpPackerRegistryMetadata(t *testing.T) {
	artifact := &Artifact{
		Images: map[string]string{
			"east": testImageID,
			"west": "bar",
		},
		StateData: map[string]interface{}{
			"generated_data": map[string]interface{}{"SourceImage": "ami-12345"},
		},
	}

	result := artifact.State(registryimage.ArtifactStateURI)
	if result == nil {
		t.Fatalf("Bad: HCP Packer registry image data was nil")
	}

	var images []registryimage.Image
	err := mapstructure.Decode(result, &images)
	if err != nil {
		t.Errorf(
			"Bad: unexpected error when trying to decode state into registryimage.Image %v",
			err,
		)
	}

	if len(images) != 2 {
		t.Errorf("Bad: we should have two images for this test Artifact but we got %d", len(images))
	}

	expected := []registryimage.Image{
		{
			ImageID:        testImageID,
			ProviderName:   "numspot",
			ProviderRegion: "east",
			SourceImageID:  "ami-12345",
		},
		{
			ImageID:        "bar",
			ProviderName:   "numspot",
			ProviderRegion: "west",
			SourceImageID:  "ami-12345",
		},
	}

	sort.Slice(expected, func(i, j int) bool {
		return expected[i].ImageID < expected[j].ImageID
	})
	sort.Slice(images, func(i, j int) bool {
		return images[i].ImageID < images[j].ImageID
	})

	if !reflect.DeepEqual(images, expected) {
		t.Fatalf("bad: %#v", images)
	}
}
