package common

import (
	"reflect"
	"testing"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

func ptrTo[T any](t T) *T {
	return &t
}

func TestBlockDevice_LaunchDevices(t *testing.T) {
	cases := []struct {
		Config *BlockDevice
		Result numspot.BlockDeviceMappingVmCreation
	}{
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				SnapshotID:         "snap-1234",
				VolumeType:         "standard",
				VolumeSize:         8,
				DeleteOnVMDeletion: true,
			},

			Result: numspot.BlockDeviceMappingVmCreation{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					SnapshotId:         ptrTo("snap-1234"),
					VolumeType:         ptrTo("standard"),
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(true),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName: "/dev/sdb",
				VolumeSize: 8,
			},

			Result: numspot.BlockDeviceMappingVmCreation{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(false),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				VolumeType:         "io1",
				VolumeSize:         8,
				DeleteOnVMDeletion: true,
				IOPS:               1000,
			},

			Result: numspot.BlockDeviceMappingVmCreation{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeType:         ptrTo("io1"),
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(true),
					Iops:               ptrTo(1000),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				VolumeType:         "gp2",
				VolumeSize:         8,
				DeleteOnVMDeletion: true,
			},

			Result: numspot.BlockDeviceMappingVmCreation{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeType:         ptrTo("gp2"),
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(true),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				VolumeType:         "gp2",
				VolumeSize:         8,
				DeleteOnVMDeletion: true,
			},

			Result: numspot.BlockDeviceMappingVmCreation{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeType:         ptrTo("gp2"),
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(true),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				VolumeType:         "standard",
				DeleteOnVMDeletion: true,
			},

			Result: numspot.BlockDeviceMappingVmCreation{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeType:         ptrTo("standard"),
					DeleteOnVmDeletion: ptrTo(true),
				},
			},
		},
	}

	for _, tc := range cases {

		launchBlockDevices := LaunchBlockDevices{
			LaunchMappings: []BlockDevice{*tc.Config},
		}

		expected := []numspot.BlockDeviceMappingVmCreation{tc.Result}

		launchResults := launchBlockDevices.BuildNumspotLaunchDevices()
		if !reflect.DeepEqual(expected, launchResults) {
			t.Fatalf("Bad block device, \nexpected: %#v\n\ngot: %#v",
				expected, launchResults)
		}
	}
}

func TestBlockDevice_Image(t *testing.T) {
	cases := []struct {
		Config *BlockDevice
		Result numspot.BlockDeviceMappingImage
	}{
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				SnapshotID:         "snap-1234",
				VolumeType:         "standard",
				VolumeSize:         8,
				DeleteOnVMDeletion: true,
			},

			Result: numspot.BlockDeviceMappingImage{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					SnapshotId:         ptrTo("snap-1234"),
					VolumeType:         ptrTo("standard"),
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(true),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				VolumeSize:         8,
				DeleteOnVMDeletion: true,
			},

			Result: numspot.BlockDeviceMappingImage{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(true),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				VolumeType:         "io1",
				VolumeSize:         8,
				DeleteOnVMDeletion: true,
				IOPS:               1000,
			},

			Result: numspot.BlockDeviceMappingImage{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeType:         ptrTo("io1"),
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(true),
					Iops:               ptrTo(1000),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				VolumeType:         "gp2",
				VolumeSize:         8,
				DeleteOnVMDeletion: true,
			},

			Result: numspot.BlockDeviceMappingImage{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeType:         ptrTo("gp2"),
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(true),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				VolumeType:         "gp2",
				VolumeSize:         8,
				DeleteOnVMDeletion: true,
			},

			Result: numspot.BlockDeviceMappingImage{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeType:         ptrTo("gp2"),
					VolumeSize:         ptrTo(8),
					DeleteOnVmDeletion: ptrTo(true),
				},
			},
		},
		{
			Config: &BlockDevice{
				DeviceName:         "/dev/sdb",
				VolumeType:         "standard",
				DeleteOnVMDeletion: true,
			},

			Result: numspot.BlockDeviceMappingImage{
				DeviceName: ptrTo("/dev/sdb"),
				Bsu: &numspot.BsuToCreate{
					VolumeType:         ptrTo("standard"),
					DeleteOnVmDeletion: ptrTo(true),
				},
			},
		},
	}

	for i, tc := range cases {
		imageBlockDevices := ImageBlockDevices{
			ImageMappings: []BlockDevice{*tc.Config},
		}

		expected := []numspot.BlockDeviceMappingImage{tc.Result}

		imageResults := imageBlockDevices.BuildNumspotImageDevices()
		if !reflect.DeepEqual(expected, imageResults) {
			t.Fatalf("%d - Bad block device, \nexpected: %+#v\n\ngot: %+#v",
				i, expected, imageResults)
		}
	}
}
