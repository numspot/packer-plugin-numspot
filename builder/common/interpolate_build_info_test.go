package common

import (
	"reflect"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

func testImage() numspot.Image {
	imageID := "ami-abcd1234"
	imageName := "ami_test_name"
	return numspot.Image{
		Id:   &imageID,
		Name: &imageName,
		Tags: &[]numspot.ResourceTag{
			{
				Key:   "key-1",
				Value: "value-1",
			},
			{
				Key:   "key-2",
				Value: "value-2",
			},
		},
	}
}

func testState() multistep.StateBag {
	state := new(multistep.BasicStateBag)
	return state
}

func TestInterpolateBuildInfo_extractBuildInfo_noSourceImage(t *testing.T) {
	state := testState()
	buildInfo := extractBuildInfo("foo", state)

	expected := BuildInfoTemplate{
		BuildRegion: "foo",
	}
	if !reflect.DeepEqual(*buildInfo, expected) {
		t.Fatalf("Unexpected BuildInfoTemplate: expected %#v got %#v\n", expected, *buildInfo)
	}
}

func TestInterpolateBuildInfo_extractBuildInfo_withSourceImage(t *testing.T) {
	state := testState()
	state.Put("source_image", testImage())
	buildInfo := extractBuildInfo("foo", state)

	expected := BuildInfoTemplate{
		BuildRegion:     "foo",
		SourceImage:     "ami-abcd1234",
		SourceImageName: "ami_test_name",
		SourceImageTags: map[string]string{
			"key-1": "value-1",
			"key-2": "value-2",
		},
	}
	if !reflect.DeepEqual(*buildInfo, expected) {
		t.Fatalf("Unexpected BuildInfoTemplate: expected %#v got %#v\n", expected, *buildInfo)
	}
}
