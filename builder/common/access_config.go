package common

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"

	"github.com/numspot/numspot-plugin-packer/numspot"
)

const defaultRegion = "eu-west-2"

// AccessConfig holds the credentials and connection settings for the Numspot API.
type AccessConfig struct {
	ClientID              string `mapstructure:"client_id" required:"true"`
	ClientSecret          string `mapstructure:"client_secret" required:"true"`
	SpaceID               string `mapstructure:"space_id" required:"true"`
	Region                string `mapstructure:"region" required:"false"`
	InsecureSkipTLSVerify bool   `mapstructure:"insecure_skip_tls_verify" required:"false"`

	host   string
	client *numspot.NumspotClient
}

func buildHostFromRegion(region string) string {
	return fmt.Sprintf("https://api.%s.numspot.com", region)
}

func getValueFromEnvVariables(envVariables []string) (string, bool) {
	for _, envVariable := range envVariables {
		if value, ok := os.LookupEnv(envVariable); ok && value != "" {
			return value, true
		}
	}
	return "", false
}

// Prepare validates the AccessConfig, falling back to environment variables.
func (c *AccessConfig) Prepare(_ *interpolate.Context) []error {
	var errs []error

	if c.ClientID == "" {
		if value, ok := getValueFromEnvVariables([]string{"NUMSPOT_CLIENT_ID"}); ok {
			c.ClientID = value
		} else {
			errs = append(errs, errClientIDRequired)
		}
	}

	if c.ClientSecret == "" {
		if value, ok := getValueFromEnvVariables([]string{"NUMSPOT_CLIENT_SECRET"}); ok {
			c.ClientSecret = value
		} else {
			errs = append(errs, errClientSecretRequired)
		}
	}

	if c.SpaceID == "" {
		if value, ok := getValueFromEnvVariables([]string{"NUMSPOT_SPACE_ID"}); ok {
			c.SpaceID = value
		} else {
			errs = append(errs, errSpaceIDRequired)
		}
	}

	if c.Region == "" {
		if value, ok := getValueFromEnvVariables([]string{"NUMSPOT_REGION"}); ok {
			c.Region = value
		} else {
			c.Region = defaultRegion
		}
	}

	c.host = buildHostFromRegion(c.Region)

	if (c.ClientID != "") != (c.ClientSecret != "") {
		errs = append(errs, errClientIDSecretBothSet)
	}

	return errs
}

// NewNumspotClient creates and caches a new Numspot API client.
func (c *AccessConfig) NewNumspotClient(ctx context.Context) (*numspot.NumspotClient, error) {
	if c.client != nil {
		return c.client, nil
	}

	opts := []numspot.Option{
		numspot.WithHost(c.host),
		numspot.WithClientID(c.ClientID),
		numspot.WithClientSecret(c.ClientSecret),
		numspot.WithSpaceID(c.SpaceID),
	}

	client, err := numspot.NewNumspotClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating numspot client: %w", err)
	}

	c.client = client
	return client, nil
}

// GetClient returns the underlying API client, initializing it if needed.
func (c *AccessConfig) GetClient(ctx context.Context) (*numspot.ClientWithResponses, error) {
	if c.client == nil {
		_, err := c.NewNumspotClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("initializing numspot client: %w", err)
		}
	}
	client, err := c.client.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("c.client.GetClient: %w", err)
	}
	return client, nil
}

// GetSpaceID returns the configured space Id.
func (c *AccessConfig) GetSpaceID() string {
	return c.SpaceID
}

// GetRegion returns the configured region.
func (c *AccessConfig) GetRegion() string {
	return c.Region
}

// GetHost returns the configured host URL.
func (c *AccessConfig) GetHost() string {
	return c.host
}
