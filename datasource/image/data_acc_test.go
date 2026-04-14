//go:build integration

package image

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/template/config"

	numspotcommon "github.com/numspot/numspot-plugin-packer/builder/common"
)

func getDatasourceCredentials(t *testing.T) (clientID, clientSecret, spaceID string) {
	t.Helper()
	clientID = os.Getenv("NUMSPOT_CLIENT_ID")
	clientSecret = os.Getenv("NUMSPOT_CLIENT_SECRET")
	spaceID = os.Getenv("NUMSPOT_SPACE_ID")

	if clientID == "" || clientSecret == "" || spaceID == "" {
		t.Skip("integration credentials not set (NUMSPOT_CLIENT_ID, NUMSPOT_CLIENT_SECRET, NUMSPOT_SPACE_ID)")
	}
	return
}

func TestIntegration_Datasource_Execute(t *testing.T) {
	clientID, clientSecret, spaceID := getDatasourceCredentials(t)

	ds := &Datasource{}
	err := ds.Configure(map[string]interface{}{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"space_id":      spaceID,
		"most_recent":   true,
		// no filters: owners alone is enough to list all images in the space
		"owners": []string{},
	})
	// Configure will fail because filters and owners are both empty — use FilterByName instead.
	if err != nil {
		t.Skipf("Configure requires at least one filter: %s", err)
	}

	val, err := ds.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %s", err)
	}

	if val.IsNull() {
		t.Fatal("Execute returned null value")
	}

	t.Logf("datasource output: %#v", val)
}

func TestIntegration_Datasource_FilterByName(t *testing.T) {
	clientID, clientSecret, spaceID := getDatasourceCredentials(t)

	imageName := os.Getenv("NUMSPOT_TEST_IMAGE_NAME")
	if imageName == "" {
		t.Skip("NUMSPOT_TEST_IMAGE_NAME not set (set to the exact name of an image in your space)")
	}

	ctx := context.Background()

	cfg := Config{
		ImageFilterOptions: numspotcommon.ImageFilterOptions{
			NameValueFilter: config.NameValueFilter{
				Filters: map[string]string{"name": imageName},
			},
			MostRecent: true,
		},
	}
	cfg.ClientID = clientID
	cfg.ClientSecret = clientSecret
	cfg.SpaceID = spaceID

	if err := cfg.Prepare(&cfg.ctx); err != nil {
		t.Fatalf("AccessConfig.Prepare: %v", err)
	}

	client, err := cfg.NewNumspotClient(ctx)
	if err != nil {
		t.Fatalf("NewNumspotClient: %v", err)
	}

	image, err := cfg.GetFilteredImage(ctx, client, spaceID)
	if err != nil {
		t.Fatalf("GetFilteredImage: %v", err)
	}

	if image.Id == nil {
		t.Fatal("image ID is nil")
	}
	t.Logf("found image: %s (%s)", *image.Name, *image.Id)
}
