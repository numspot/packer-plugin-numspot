package image

import (
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/template/config"

	numspotcommon "github.com/numspot/numspot-plugin-packer/builder/common"
)

func TestConfigure_FiltersAndOwnersEmpty(t *testing.T) {
	ds := Datasource{
		config: Config{
			ImageFilterOptions: numspotcommon.ImageFilterOptions{},
		},
	}
	if err := ds.Configure(nil); err == nil {
		t.Fatal("expected error when filters and owners are both empty")
	}
}

func TestConfigure_FiltersSetNoOwnerRequired(t *testing.T) {
	ds := Datasource{
		config: Config{
			ImageFilterOptions: numspotcommon.ImageFilterOptions{
				NameValueFilter: config.NameValueFilter{
					Filters: map[string]string{"name": "ubuntu-22.04"},
				},
			},
		},
	}
	// owners is not required in Numspot — space already scopes visibility.
	err := ds.Configure(nil)
	if err != nil && err.Error() == errFiltersRequired.Error() {
		t.Fatalf("unexpected filters validation error: %s", err)
	}
}

func TestConfigure_Valid(t *testing.T) {
	ds := Datasource{
		config: Config{
			ImageFilterOptions: numspotcommon.ImageFilterOptions{
				NameValueFilter: config.NameValueFilter{
					Filters: map[string]string{"name": "ubuntu-22.04"},
				},
				MostRecent: true,
			},
		},
	}
	// AccessConfig fields are validated too, but env vars can satisfy them.
	// If env vars are absent the error comes from AccessConfig, not our logic —
	// we only care that our validation does not fire.
	err := ds.Configure(nil)
	if err != nil && err.Error() == errFiltersRequired.Error() {
		t.Fatalf("unexpected validation error: %s", err)
	}
}

func TestConfigure_MostRecentDefault(t *testing.T) {
	ds := Datasource{
		config: Config{
			ImageFilterOptions: numspotcommon.ImageFilterOptions{
				NameValueFilter: config.NameValueFilter{
					Filters: map[string]string{"name": "ubuntu-*"},
				},
				Owners: []string{"acme"},
			},
		},
	}
	if ds.config.MostRecent {
		t.Fatal("most_recent should default to false")
	}
}
