//go:build integration

package bsu

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	numspotcommon "github.com/numspot/numspot-plugin-packer/builder/common"
	numspot "github.com/numspot/numspot-plugin-packer/numspot"
)

func getIntegrationCredentials(t *testing.T) (clientID, clientSecret, spaceID, host, subnetID string) {
	clientID = os.Getenv("NUMSPOT_CLIENT_ID")
	clientSecret = os.Getenv("NUMSPOT_CLIENT_SECRET")
	spaceID = os.Getenv("NUMSPOT_SPACE_ID")
	host = os.Getenv("NUMSPOT_HOST")
	subnetID = os.Getenv("NUMSPOT_SUBNET_ID")

	if clientID == "" || clientSecret == "" || spaceID == "" || host == "" || subnetID == "" {
		t.Skip("Integration test credentials not set")
	}
	return
}

func createTestClient(t *testing.T) *numspot.NumspotClient {
	clientID, clientSecret, spaceID, host, _ := getIntegrationCredentials(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := numspot.NewNumspotClient(ctx,
		numspot.WithHost(host),
		numspot.WithClientID(clientID),
		numspot.WithClientSecret(clientSecret),
		numspot.WithSpaceID(spaceID),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}

func TestIntegration_FindSourceImage(t *testing.T) {
	client := createTestClient(t)
	_, _, spaceID, _, _ := getIntegrationCredentials(t)
	ctx := context.Background()

	filter := numspotcommon.ImageFilterOptions{
		MostRecent: true,
	}
	filter.Filters = map[string]string{"image-name": "Ubuntu*"}

	image, err := filter.GetFilteredImage(ctx, client, spaceID)
	if err != nil {
		t.Fatalf("Failed to find image: %v", err)
	}

	if image == nil || image.Id == nil {
		t.Fatal("Expected image, got nil")
	}

	t.Logf("Found image: %s (%s)", *image.Name, *image.Id)
}

func TestIntegration_CreateAndDeleteVM(t *testing.T) {
	client := createTestClient(t)
	_, _, spaceID, _, subnetID := getIntegrationCredentials(t)
	ctx := context.Background()

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		t.Fatalf("Failed to get API client: %v", err)
	}

	var spaceUuid numspot.SpaceId
	if err := spaceUuid.UnmarshalText([]byte(spaceID)); err != nil {
		t.Fatalf("Invalid space ID: %v", err)
	}

	keypairName := fmt.Sprintf("pk-integration-test-%d", time.Now().Unix())

	t.Run("CreateKeypair", func(t *testing.T) {
		resp, err := apiClient.CreateKeypairWithResponse(ctx, spaceUuid, numspot.CreateKeypairJSONRequestBody{
			Name: keypairName,
		})
		if err != nil {
			t.Fatalf("Failed to create keypair: %v", err)
		}
		if resp.JSON201 == nil {
			t.Fatalf("Expected keypair, got status %d", resp.StatusCode())
		}
		t.Logf("Created keypair: %s", *resp.JSON201.Name)
	})

	defer func() {
		t.Run("CleanupKeypair", func(t *testing.T) {
			_, err := apiClient.DeleteKeypair(ctx, spaceUuid, keypairName)
			if err != nil {
				t.Logf("Warning: failed to delete keypair: %v", err)
			}
		})
	}()

	imageID := "ami-42e5719f"
	vmType := "ns-eco7-2c2r"

	var vmID string

	t.Run("CreateVM", func(t *testing.T) {
		resp, err := apiClient.CreateVmsWithResponse(ctx, spaceUuid, numspot.CreateVms{
			ImageId:     imageID,
			Type:        vmType,
			KeypairName: &keypairName,
			SubnetId:    subnetID,
		})
		if err != nil {
			t.Fatalf("Failed to create VM: %v", err)
		}
		if resp.JSON201 == nil {
			t.Fatalf("Expected VM, got status %d: %s", resp.StatusCode(), string(resp.Body))
		}
		if resp.JSON201.Id == nil {
			t.Fatal("VM ID is nil")
		}

		vmID = *resp.JSON201.Id
		t.Logf("Created VM: %s", vmID)

		t.Run("WaitForRunning", func(t *testing.T) {
			err := numspot.WaitUntilVMRunning(ctx, apiClient, spaceID, vmID)
			if err != nil {
				t.Fatalf("VM did not reach running state: %v", err)
			}
			t.Log("VM is running")
		})

		t.Run("StopVM", func(t *testing.T) {
			_, err := apiClient.StopVmWithResponse(ctx, spaceUuid, vmID, numspot.StopVm{})
			if err != nil {
				t.Fatalf("Failed to stop VM: %v", err)
			}

			err = numspot.WaitUntilVMStopped(ctx, apiClient, spaceID, vmID)
			if err != nil {
				t.Fatalf("VM did not reach stopped state: %v", err)
			}
			t.Log("VM is stopped")
		})

		t.Run("DeleteVM", func(t *testing.T) {
			_, err := apiClient.DeleteVmsWithResponse(ctx, spaceUuid, vmID)
			if err != nil {
				t.Fatalf("Failed to delete VM: %v", err)
			}

			err = numspot.WaitUntilVMDeleted(ctx, apiClient, spaceID, vmID)
			if err != nil {
				t.Fatalf("VM was not deleted: %v", err)
			}
			t.Log("VM is deleted")
		})
	})
}

func TestIntegration_CreateAndDeleteImage(t *testing.T) {
	client := createTestClient(t)
	_, _, spaceID, _, subnetID := getIntegrationCredentials(t)
	ctx := context.Background()

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		t.Fatalf("Failed to get API client: %v", err)
	}

	var spaceUuid numspot.SpaceId
	if err := spaceUuid.UnmarshalText([]byte(spaceID)); err != nil {
		t.Fatalf("Invalid space ID: %v", err)
	}

	imageName := fmt.Sprintf("packer-test-image-%d", time.Now().Unix())
	sourceImageID := "ami-42e5719f"
	vmType := "ns-eco7-2c2r"
	keypairName := fmt.Sprintf("pk-img-test-%d", time.Now().Unix())

	t.Run("SetupKeypair", func(t *testing.T) {
		_, err := apiClient.CreateKeypairWithResponse(ctx, spaceUuid, numspot.CreateKeypairJSONRequestBody{
			Name: keypairName,
		})
		if err != nil {
			t.Fatalf("Failed to create keypair: %v", err)
		}
	})
	defer func() {
		_, _ = apiClient.DeleteKeypair(ctx, spaceUuid, keypairName)
	}()

	var vmID string
	var createdImageID string

	t.Run("CreateSourceVM", func(t *testing.T) {
		resp, err := apiClient.CreateVmsWithResponse(ctx, spaceUuid, numspot.CreateVms{
			ImageId:     sourceImageID,
			Type:        vmType,
			KeypairName: &keypairName,
			SubnetId:    subnetID,
		})
		if err != nil || resp.JSON201 == nil {
			t.Fatalf("Failed to create VM: %v", err)
		}
		vmID = *resp.JSON201.Id
		t.Logf("Created source VM: %s", vmID)

		err = numspot.WaitUntilVMRunning(ctx, apiClient, spaceID, vmID)
		if err != nil {
			t.Fatalf("VM did not start: %v", err)
		}
	})

	t.Run("StopSourceVM", func(t *testing.T) {
		_, err := apiClient.StopVmWithResponse(ctx, spaceUuid, vmID, numspot.StopVm{})
		if err != nil {
			t.Fatalf("Failed to stop VM: %v", err)
		}
		err = numspot.WaitUntilVMStopped(ctx, apiClient, spaceID, vmID)
		if err != nil {
			t.Fatalf("VM did not stop: %v", err)
		}
	})

	t.Run("CreateImage", func(t *testing.T) {
		desc := "Integration test image"
		resp, err := apiClient.CreateImageWithResponse(ctx, spaceUuid, numspot.CreateImage{
			Name:        &imageName,
			Description: &desc,
			VmId:        &vmID,
		})
		if err != nil {
			t.Fatalf("Failed to create image: %v", err)
		}
		if resp.JSON201 == nil {
			t.Fatalf("Expected image, got status %d", resp.StatusCode())
		}
		if resp.JSON201.Id == nil {
			t.Fatal("Image ID is nil")
		}
		createdImageID = *resp.JSON201.Id
		t.Logf("Created image: %s", createdImageID)

		err = numspot.WaitUntilImageAvailable(ctx, apiClient, spaceID, createdImageID)
		if err != nil {
			t.Fatalf("Image did not become available: %v", err)
		}
		t.Log("Image is available")
	})

	t.Run("VerifyImage", func(t *testing.T) {
		resp, err := apiClient.ReadImagesByIdWithResponse(ctx, spaceUuid, createdImageID)
		if err != nil {
			t.Fatalf("Failed to read image: %v", err)
		}
		if resp.JSON200 == nil {
			t.Fatal("Expected image, got nil")
		}
		if resp.JSON200.Name == nil || *resp.JSON200.Name != imageName {
			t.Fatalf("Image name mismatch: expected %s, got %v", imageName, resp.JSON200.Name)
		}
		t.Logf("Verified image: %s", *resp.JSON200.Name)
	})

	t.Run("DeleteImage", func(t *testing.T) {
		_, err := apiClient.DeleteImageWithResponse(ctx, spaceUuid, createdImageID)
		if err != nil {
			t.Fatalf("Failed to delete image: %v", err)
		}
		t.Log("Image deleted")
	})

	t.Run("CleanupVM", func(t *testing.T) {
		_, err := apiClient.DeleteVmsWithResponse(ctx, spaceUuid, vmID)
		if err != nil {
			t.Logf("Warning: failed to delete VM: %v", err)
		}
	})
}

func TestIntegration_Tags(t *testing.T) {
	client := createTestClient(t)
	_, _, spaceID, _, subnetID := getIntegrationCredentials(t)
	ctx := context.Background()

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		t.Fatalf("Failed to get API client: %v", err)
	}

	var spaceUuid numspot.SpaceId
	if err := spaceUuid.UnmarshalText([]byte(spaceID)); err != nil {
		t.Fatalf("Invalid space ID: %v", err)
	}

	keypairName := fmt.Sprintf("pk-tags-test-%d", time.Now().Unix())

	t.Run("CreateKeypair", func(t *testing.T) {
		_, err := apiClient.CreateKeypairWithResponse(ctx, spaceUuid, numspot.CreateKeypairJSONRequestBody{
			Name: keypairName,
		})
		if err != nil {
			t.Fatalf("Failed to create keypair: %v", err)
		}
	})
	defer func() {
		_, _ = apiClient.DeleteKeypair(ctx, spaceUuid, keypairName)
	}()

	var resourceID string
	t.Run("CreateResource", func(t *testing.T) {
		resp, err := apiClient.CreateVmsWithResponse(ctx, spaceUuid, numspot.CreateVms{
			ImageId:     "ami-42e5719f",
			Type:        "ns-eco7-2c2r",
			KeypairName: &keypairName,
			SubnetId:    subnetID,
		})
		if err != nil || resp.JSON201 == nil {
			t.Fatalf("Failed to create VM: %v", err)
		}
		resourceID = *resp.JSON201.Id
		t.Logf("Created VM: %s", resourceID)

		err = numspot.WaitUntilVMRunning(ctx, apiClient, spaceID, resourceID)
		if err != nil {
			t.Fatalf("VM did not start: %v", err)
		}
	})

	t.Run("CreateTags", func(t *testing.T) {
		tags := []numspot.ResourceTag{
			{Key: "CreatedBy", Value: "packer-integration-test"},
			{Key: "Environment", Value: "test"},
		}

		_, err := apiClient.CreateTagsWithResponse(ctx, spaceUuid, numspot.CreateTags{
			ResourceIds: []string{resourceID},
			Tags:        tags,
		})
		if err != nil {
			t.Fatalf("Failed to create tags: %v", err)
		}
		t.Log("Tags created")
	})

	t.Run("VerifyTags", func(t *testing.T) {
		resp, err := apiClient.ReadVmsByIdWithResponse(ctx, spaceUuid, resourceID)
		if err != nil {
			t.Fatalf("Failed to read VM: %v", err)
		}
		if resp.JSON200 == nil || resp.JSON200.Tags == nil {
			t.Fatal("Expected tags on VM")
		}

		tagMap := make(map[string]string)
		for _, tag := range *resp.JSON200.Tags {
			tagMap[tag.Key] = tag.Value
		}

		if tagMap["CreatedBy"] != "packer-integration-test" {
			t.Fatalf("Tag CreatedBy not found or wrong value")
		}
		t.Logf("Verified tags: %v", tagMap)
	})

	t.Run("Cleanup", func(t *testing.T) {
		_, _ = apiClient.DeleteVmsWithResponse(ctx, spaceUuid, resourceID)
	})
}

func TestIntegration_SecurityGroup(t *testing.T) {
	client := createTestClient(t)
	_, _, spaceID, _, subnetID := getIntegrationCredentials(t)
	ctx := context.Background()

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		t.Fatalf("Failed to get API client: %v", err)
	}

	var spaceUuid numspot.SpaceId
	if err := spaceUuid.UnmarshalText([]byte(spaceID)); err != nil {
		t.Fatalf("Invalid space ID: %v", err)
	}

	var vpcID string
	subnetResp, err := apiClient.ReadSubnetsWithResponse(ctx, spaceUuid, nil)
	if err != nil {
		t.Fatalf("Failed to get subnet: %v", err)
	}
	if subnetResp.JSON200 != nil && subnetResp.JSON200.Items != nil && len(*subnetResp.JSON200.Items) > 0 {
		for _, subnet := range *subnetResp.JSON200.Items {
			if subnet.Id != nil && *subnet.Id == subnetID && subnet.VpcId != nil {
				vpcID = *subnet.VpcId
				break
			}
		}
	}

	sgName := fmt.Sprintf("packer-sg-test-%d", time.Now().Unix())
	var sgID string

	t.Run("CreateSecurityGroup", func(t *testing.T) {
		desc := "Integration test security group"
		resp, err := apiClient.CreateSecurityGroupWithResponse(ctx, spaceUuid, numspot.CreateSecurityGroup{
			Name:        sgName,
			Description: desc,
			VpcId:       vpcID,
		})
		if err != nil {
			t.Fatalf("Failed to create security group: %v", err)
		}
		if resp.JSON201 == nil {
			t.Fatalf("Expected security group, got status %d", resp.StatusCode())
		}
		sgID = *resp.JSON201.Id
		t.Logf("Created security group: %s (%s)", *resp.JSON201.Name, sgID)
	})

	t.Run("CreateSecurityGroupRule", func(t *testing.T) {
		port := 22
		protocol := "tcp"
		cidr := "0.0.0.0/0"

		_, err := apiClient.CreateSecurityGroupRuleWithResponse(ctx, spaceUuid, sgID, numspot.CreateSecurityGroupRule{
			Flow:          "Inbound",
			IpProtocol:    &protocol,
			FromPortRange: &port,
			ToPortRange:   &port,
			IpRange:       &cidr,
		})
		if err != nil {
			t.Fatalf("Failed to create security group rule: %v", err)
		}
		t.Log("Created SSH rule")
	})

	t.Run("CleanupSecurityGroup", func(t *testing.T) {
		_, err := apiClient.DeleteSecurityGroupWithResponse(ctx, spaceUuid, sgID)
		if err != nil {
			t.Logf("Warning: failed to delete security group: %v", err)
		} else {
			t.Log("Security group deleted")
		}
	})
}

func TestIntegration_PublicIP(t *testing.T) {
	client := createTestClient(t)
	_, _, spaceID, _, _ := getIntegrationCredentials(t)
	ctx := context.Background()

	apiClient, err := client.GetClient(ctx)
	if err != nil {
		t.Fatalf("Failed to get API client: %v", err)
	}

	var spaceUuid numspot.SpaceId
	if err := spaceUuid.UnmarshalText([]byte(spaceID)); err != nil {
		t.Fatalf("Invalid space ID: %v", err)
	}

	var publicIPID string

	t.Run("CreatePublicIP", func(t *testing.T) {
		resp, err := apiClient.CreatePublicIpWithResponse(ctx, spaceUuid)
		if err != nil {
			t.Fatalf("Failed to create public IP: %v", err)
		}
		if resp.JSON201 == nil {
			t.Fatalf("Expected public IP, got status %d", resp.StatusCode())
		}
		publicIPID = *resp.JSON201.Id
		t.Logf("Created public IP: %s", publicIPID)
	})

	t.Run("DeletePublicIP", func(t *testing.T) {
		_, err := apiClient.DeletePublicIp(ctx, spaceUuid, publicIPID)
		if err != nil {
			t.Fatalf("Failed to delete public IP: %v", err)
		}
		t.Log("Public IP deleted")
	})
}
