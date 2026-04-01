# Code Templates

This document contains code templates for key components of the Numspot Packer plugin.

## 1. Numspot SDK Client

`numspot-sdk-go/client.go`

```go
package numspot

import (
    "context"
    "crypto/tls"
    "encoding/base64"
    "fmt"
    "net/http"
    "time"

    "github.com/deepmap/oapi-codegen/pkg/securityprovider"
    "github.com/google/uuid"
)

const (
    UserAgentHeader    = "User-Agent"
    PackerUserAgent    = "PACKER-NUMSPOT"
    Credentials        = "client_credentials"
)

type NumspotClient struct {
    ID                    string
    Client                *ClientWithResponses
    HTTPClient            *http.Client
    SpaceID               uuid.UUID
    ClientID              uuid.UUID
    ClientSecret          string
    Host                  string
    AccessTokenExpiration time.Time
}

type ClientConfig struct {
    Host         string
    SpaceID      string
    ClientID     string
    ClientSecret string
    HTTPClient   *http.Client
}

type Option func(s *NumspotClient) error

func WithHost(host string) Option {
    return func(s *NumspotClient) error {
        s.Host = host
        return nil
    }
}

func WithSpaceID(spaceID string) Option {
    return func(s *NumspotClient) error {
        numSpotSpaceID, err := uuid.Parse(spaceID)
        if err != nil {
            return fmt.Errorf("invalid space_id: %w", err)
        }
        s.SpaceID = numSpotSpaceID
        return nil
    }
}

func WithClientID(clientID string) Option {
    return func(s *NumspotClient) error {
        clientUUID, err := uuid.Parse(clientID)
        if err != nil {
            return fmt.Errorf("invalid client_id: %w", err)
        }
        s.ClientID = clientUUID
        return nil
    }
}

func WithClientSecret(clientSecret string) Option {
    return func(s *NumspotClient) error {
        s.ClientSecret = clientSecret
        return nil
    }
}

func WithHTTPClient(client *http.Client) Option {
    return func(s *NumspotClient) error {
        s.HTTPClient = client
        return nil
    }
}

func NewClient(ctx context.Context, options ...Option) (*NumspotClient, error) {
    client := &NumspotClient{
        ID:                    uuid.NewString(),
        AccessTokenExpiration: time.Now(),
    }

    for _, o := range options {
        if err := o(client); err != nil {
            return nil, err
        }
    }

    if err := client.createClientAPI(); err != nil {
        return nil, err
    }

    if err := client.authenticate(ctx); err != nil {
        return nil, err
    }

    return client, nil
}

func isTokenExpired(expirationTime time.Time) bool {
    return time.Now().After(expirationTime)
}

func (s *NumspotClient) GetClient(ctx context.Context) (*ClientWithResponses, error) {
    if isTokenExpired(s.AccessTokenExpiration) {
        if err := s.authenticate(ctx); err != nil {
            return nil, fmt.Errorf("error refreshing access token: %w", err)
        }
    }
    return s.Client, nil
}

func (s *NumspotClient) createClientAPI() error {
    requestEditor := WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
        req.Header.Add(UserAgentHeader, PackerUserAgent)
        return nil
    })

    var err error
    s.Client, err = NewClientWithResponses(s.Host, s.newTransport(), requestEditor)
    if err != nil {
        return err
    }

    return nil
}

func (s *NumspotClient) authenticate(ctx context.Context) error {
    basicAuth := buildBasicAuth(s.ClientID.String(), s.ClientSecret)

    response, err := s.Client.TokenWithFormdataBodyWithResponse(ctx, &TokenParams{
        Authorization: &basicAuth,
    }, TokenReq{
        GrantType:    Credentials,
        ClientId:     &s.ClientID,
        ClientSecret: &s.ClientSecret,
    })
    if err != nil {
        return fmt.Errorf("token request failed: %w", err)
    }

    if response.StatusCode() != 200 {
        return fmt.Errorf("authentication failed: status %d", response.StatusCode())
    }

    expirationTimeMargin := 5 * 60 // 5 minutes margin
    var expirationTime int
    if response.JSON200.ExpiresIn > expirationTimeMargin {
        expirationTime = response.JSON200.ExpiresIn - expirationTimeMargin
    } else {
        return fmt.Errorf("token expiration too short: %d", response.JSON200.ExpiresIn)
    }

    s.AccessTokenExpiration = time.Now().Add(time.Duration(expirationTime) * time.Second)

    bearerProvider, err := securityprovider.NewSecurityProviderBearerToken(response.JSON200.AccessToken)
    if err != nil {
        return fmt.Errorf("bearer token setup failed: %w", err)
    }

    s.Client, err = NewClientWithResponses(s.Host, s.newTransport(), 
        WithRequestEditorFn(bearerProvider.Intercept))
    if err != nil {
        return err
    }

    return nil
}

func buildBasicAuth(username, password string) string {
    auth := username + ":" + password
    return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func (s *NumspotClient) newTransport() func(c *Client) error {
    return func(c *Client) error {
        if s.HTTPClient != nil {
            c.Client = s.HTTPClient
        } else {
            c.Client = &http.Client{
                Transport: &http.Transport{
                    TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
                    Proxy:           http.ProxyFromEnvironment,
                },
            }
        }
        return nil
    }
}
```

## 2. Packer Access Config

`builder/common/access_config.go`

```go
package common

import (
    "context"
    "errors"
    "fmt"
    "os"

    "github.com/hashicorp/packer-plugin-sdk/template/interpolate"
    "github.com/numspot/numspot-sdk-go"
)

type AccessConfig struct {
    ClientId     string `mapstructure:"client_id"`
    ClientSecret string `mapstructure:"client_secret"`
    SpaceId      string `mapstructure:"space_id"`
    Host         string `mapstructure:"numspot_host"`

    client *numspot.NumspotClient
}

func (c *AccessConfig) Prepare(ctx *interpolate.Context) []error {
    var errs []error

    // Resolve from environment variables
    if c.ClientId == "" {
        c.ClientId = os.Getenv("NUMSPOT_CLIENT_ID")
    }
    if c.ClientSecret == "" {
        c.ClientSecret = os.Getenv("NUMSPOT_CLIENT_SECRET")
    }
    if c.SpaceId == "" {
        c.SpaceId = os.Getenv("NUMSPOT_SPACE_ID")
    }
    if c.Host == "" {
        c.Host = os.Getenv("NUMSPOT_HOST")
    }

    // Validation
    if c.ClientId == "" {
        errs = append(errs, errors.New("client_id is required (or set NUMSPOT_CLIENT_ID env var)"))
    }
    if c.ClientSecret == "" {
        errs = append(errs, errors.New("client_secret is required (or set NUMSPOT_CLIENT_SECRET env var)"))
    }
    if c.SpaceId == "" {
        errs = append(errs, errors.New("space_id is required (or set NUMSPOT_SPACE_ID env var)"))
    }
    if c.Host == "" {
        errs = append(errs, errors.New("numspot_host is required (or set NUMSPOT_HOST env var)"))
    }

    return errs
}

func (c *AccessConfig) NewNumspotClient(ctx context.Context) (*numspot.NumspotClient, error) {
    client, err := numspot.NewClient(ctx,
        numspot.WithHost(c.Host),
        numspot.WithSpaceID(c.SpaceId),
        numspot.WithClientID(c.ClientId),
        numspot.WithClientSecret(c.ClientSecret),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create Numspot client: %w", err)
    }

    c.client = client
    return client, nil
}

func (c *AccessConfig) GetClient(ctx context.Context) (*numspot.NumspotClient, error) {
    if c.client == nil {
        return c.NewNumspotClient(ctx)
    }
    return c.client.GetClient(ctx)
}
```

## 3. Image Config

`builder/common/image_config.go`

```go
package common

import (
    "errors"
    "fmt"
    "log"
    "slices"

    "github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

type ImageConfig struct {
    ImageName           string   `mapstructure:"image_name"`
    ImageDescription    string   `mapstructure:"image_description"`
    ImageAccountIds     []string `mapstructure:"image_account_ids"`
    ImageGroups         []string `mapstructure:"image_groups"`
    ImageRegions        []string `mapstructure:"image_regions"`
    ImageBootModes      []string `mapstructure:"image_boot_modes"`
    SkipRegionValidation bool    `mapstructure:"skip_region_validation"`
    Tags                TagMap   `mapstructure:"tags"`
    ForceDeregister     bool     `mapstructure:"force_deregister"`
    ForceDeleteSnapshot bool     `mapstructure:"force_delete_snapshot"`
    SnapshotTags        TagMap   `mapstructure:"snapshot_tags"`
    SnapshotAccountIds  []string `mapstructure:"snapshot_account_ids"`
    GlobalPermission    bool     `mapstructure:"global_permission"`
    RootDeviceName      string   `mapstructure:"root_device_name"`
}

func (c *ImageConfig) Prepare(accessConfig *AccessConfig, ctx *interpolate.Context) []error {
    var errs []error

    if c.ImageName == "" {
        errs = append(errs, errors.New("image_name must be specified"))
    }

    errs = append(errs, c.prepareRegions(accessConfig)...)

    if len(c.ImageName) < 3 || len(c.ImageName) > 128 {
        errs = append(errs, errors.New("image_name must be between 3 and 128 characters long"))
    }

    if len(c.ImageBootModes) > 0 {
        bootModesSupported := []string{"legacy", "uefi"}
        for _, bootMode := range c.ImageBootModes {
            if !slices.Contains(bootModesSupported, bootMode) {
                errs = append(errs, fmt.Errorf("image_boot_modes '%s' is not supported", bootMode))
            }
        }
    }

    if c.ImageName != templateCleanResourceName(c.ImageName) {
        errs = append(errs, errors.New("image_name contains invalid characters"))
    }

    return errs
}

func (c *ImageConfig) prepareRegions(accessConfig *AccessConfig) []error {
    // Remove duplicates and the current region from ImageRegions
    if len(c.ImageRegions) > 0 {
        regionSet := make(map[string]struct{})
        regions := make([]string, 0, len(c.ImageRegions))

        for _, region := range c.ImageRegions {
            if _, ok := regionSet[region]; ok {
                continue
            }
            regionSet[region] = struct{}{}

            if accessConfig != nil && region == accessConfig.SpaceId {
                log.Printf("Cannot copy image to current space '%s', removing from image_regions", region)
                continue
            }
            regions = append(regions, region)
        }

        c.ImageRegions = regions
    }
    return nil
}
```

## 4. Step Run Source VM

`builder/common/step_run_source_vm.go`

```go
package common

import (
    "context"
    "encoding/base64"
    "errors"
    "fmt"
    "log"
    "os"

    "github.com/hashicorp/packer-plugin-sdk/communicator"
    "github.com/hashicorp/packer-plugin-sdk/multistep"
    packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
    "github.com/hashicorp/packer-plugin-sdk/template/interpolate"
    "github.com/numspot/numspot-sdk-go"
)

type StepRunSourceVm struct {
    BlockDevices                BlockDevices
    Comm                        *communicator.Config
    Ctx                         interpolate.Context
    Debug                       bool
    BsuOptimized                bool
    EnableT2Unlimited           bool
    ExpectedRootDevice          string
    IamVmProfile                string
    VmInitiatedShutdownBehavior string
    VmType                      string
    IsRestricted                bool
    SourceImage                 string
    Tags                        TagMap
    UserData                    string
    UserDataFile                string
    VolumeTags                  TagMap
    RawRegion                   string
    BootMode                    string
    vmId                        string
}

func (s *StepRunSourceVm) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
    client := state.Get("client").(*numspot.NumspotClient)
    spaceId := state.Get("spaceId").(string)
    securityGroupIds := state.Get("securityGroupIds").([]string)
    ui := state.Get("ui").(packersdk.Ui)

    userData := s.UserData
    if s.UserDataFile != "" {
        contents, err := os.ReadFile(s.UserDataFile)
        if err != nil {
            state.Put("error", fmt.Errorf("problem reading user data file: %s", err))
            return multistep.ActionHalt
        }
        userData = string(contents)
    }

    if _, err := base64.StdEncoding.DecodeString(userData); err != nil {
        log.Printf("[DEBUG] base64 encoding user data...")
        userData = base64.StdEncoding.EncodeToString([]byte(userData))
    }

    ui.Say("Launching a source Numspot VM...")
    image, ok := state.Get("source_image").(numspot.Image)
    if !ok {
        state.Put("error", errors.New("source_image type assertion failed"))
        return multistep.ActionHalt
    }
    s.SourceImage = image.Id

    if s.ExpectedRootDevice != "" && image.RootDeviceType != s.ExpectedRootDevice {
        state.Put("error", fmt.Errorf(
            "invalid root device type. Expected '%s', got '%s'",
            s.ExpectedRootDevice, image.RootDeviceType))
        return multistep.ActionHalt
    }

    ui.Say("Adding tags to source VM")
    if _, exists := s.Tags["Name"]; !exists {
        s.Tags["Name"] = "Packer Builder"
    }

    tags, err := s.Tags.ToNumspotTags(s.Ctx, state)
    if err != nil {
        state.Put("error", fmt.Errorf("error tagging source VM: %s", err))
        ui.Error(err.Error())
        return multistep.ActionHalt
    }

    volTags, err := s.VolumeTags.ToNumspotTags(s.Ctx, state)
    if err != nil {
        state.Put("error", fmt.Errorf("error tagging volumes: %s", err))
        ui.Error(err.Error())
        return multistep.ActionHalt
    }

    blockDevice := s.BlockDevices.BuildNumspotLaunchDevices()
    subnetID := state.Get("subnet_id").(string)

    runOpts := numspot.CreateVmRequest{
        ImageId:             s.SourceImage,
        Type:                s.VmType,
        UserData:            &userData,
        BsuOptimized:        &s.BsuOptimized,
        BlockDeviceMappings: blockDevice,
    }

    if s.BootMode != "" {
        runOpts.BootMode = &s.BootMode
    }

    if s.Comm.SSHKeyPairName != "" {
        runOpts.KeypairName = &s.Comm.SSHKeyPairName
    }

    runOpts.SubnetId = &subnetID
    runOpts.SecurityGroupIds = securityGroupIds

    if s.ExpectedRootDevice == "bsu" {
        runOpts.InitiatedShutdownBehavior = &s.VmInitiatedShutdownBehavior
    }

    apiClient, err := client.GetClient(ctx)
    if err != nil {
        state.Put("error", err)
        ui.Error(err.Error())
        return multistep.ActionHalt
    }

    runResp, err := apiClient.CreateVm(ctx, spaceId, runOpts)
    if err != nil {
        err := fmt.Errorf("error launching source VM: %w", err)
        state.Put("error", err)
        ui.Error(err.Error())
        return multistep.ActionHalt
    }

    vmId := runResp.Id
    volumeId := ""
    if len(runResp.BlockDeviceMappings) > 0 {
        volumeId = runResp.BlockDeviceMappings[0].Bsu.VolumeId
    }

    s.vmId = vmId

    ui.Message(fmt.Sprintf("VM ID: %s", vmId))
    ui.Say(fmt.Sprintf("Waiting for VM (%s) to become ready...", vmId))

    if err := waitUntilVmRunning(ctx, apiClient, spaceId, vmId); err != nil {
        err := fmt.Errorf("error waiting for VM (%s) to become ready: %w", vmId, err)
        state.Put("error", err)
        ui.Error(err.Error())
        return multistep.ActionHalt
    }

    // Set VM tags and volume tags
    if len(tags) > 0 {
        if err := createTags(ctx, apiClient, spaceId, s.vmId, ui, tags); err != nil {
            err := fmt.Errorf("error creating tags for VM (%s): %w", s.vmId, err)
            state.Put("error", err)
            ui.Error(err.Error())
            return multistep.ActionHalt
        }
    }

    if len(volTags) > 0 && volumeId != "" {
        if err := createTags(ctx, apiClient, spaceId, volumeId, ui, volTags); err != nil {
            err := fmt.Errorf("error creating tags for volume (%s): %w", volumeId, err)
            state.Put("error", err)
            ui.Error(err.Error())
            return multistep.ActionHalt
        }
    }

    // Get updated VM info
    vmResp, err := apiClient.ReadVmById(ctx, spaceId, vmId)
    if err != nil {
        err := errors.New("error finding source VM")
        state.Put("error", err)
        ui.Error(err.Error())
        return multistep.ActionHalt
    }

    vm := vmResp

    if s.Debug {
        if vm.PublicDnsName != nil {
            ui.Message(fmt.Sprintf("Public DNS: %s", *vm.PublicDnsName))
        }
        if vm.PublicIp != nil {
 ui.Message(fmt.Sprintf("Public IP: %s", *vm.PublicIp))
        }
    }

    state.Put("vm", vm)
    state.Put("instance_id", vmId)

    return multistep.ActionContinue
}

func (s *StepRunSourceVm) Cleanup(state multistep.StateBag) {
    client := state.Get("client").(*numspot.NumspotClient)
    spaceId := state.Get("spaceId").(string)
    ui := state.Get("ui").(packersdk.Ui)
    ctx := context.Background()

    if s.vmId != "" {
        ui.Say("Terminating the source Numspot VM...")
        apiClient, err := client.GetClient(ctx)
        if err != nil {
            ui.Error(fmt.Sprintf("Error getting client: %s", err))
            return
        }

        _, err = apiClient.DeleteVm(ctx, spaceId, s.vmId)
        if err != nil {
            ui.Error(fmt.Sprintf("Error terminating VM: %s", err))
            return
        }

        if err := waitUntilVmDeleted(ctx, apiClient, spaceId, s.vmId); err != nil {
            ui.Error(err.Error())
        }
    }
}

func waitUntilVmRunning(ctx context.Context, client *numspot.ClientWithResponses, spaceId, vmId string) error {
    // Poll VM status until running
    // Implementation similar to Outscale plugin
    return nil
}

func waitUntilVmDeleted(ctx context.Context, client *numspot.ClientWithResponses, spaceId, vmId string) error {
    // Poll VM status until deleted
    return nil
}

func createTags(ctx context.Context, client *numspot.ClientWithResponses, spaceId, resourceId string, ui packersdk.Ui, tags []numspot.Tag) error {
    // Create tags on resource
    return nil
}
```

## 5. go.mod

```go
module github.com/numspot/packer-plugin-numspot

go 1.22

require (
    github.com/hashicorp/packer-plugin-sdk v0.6.1
    github.com/numspot/numspot-sdk-go v1.0.0
    github.com/google/uuid v1.6.0
    github.com/deepmap/oapi-codegen v1.16.3
)

require (
    // Packer SDK dependencies
    github.com/hashicorp/go-hclog v1.6.3
    github.com/hashicorp/hcl/v2 v2.21.0
    github.com/zclconf/go-cty v1.15.0
    // ... other dependencies from packer-plugin-sdk
)
```

## 6. main.go

```go
package main

import (
    "fmt"
    "os"

    "github.com/hashicorp/packer-plugin-sdk/plugin"
    "github.com/numspot/packer-plugin-numspot/builder/bsu"
    "github.com/numspot/packer-plugin-numspot/datasource/image"
)

func main() {
   pps := plugin.NewSet()
    pps.RegisterBuilder(plugin.DEFAULT_NAME, "bsu", new(bsu.Builder))
    pps.RegisterDatasource(plugin.DEFAULT_NAME, "image", new(image.Datasource))
    
    if err := pps.Run(); err != nil {
        fmt.Fprintln(os.Stderr, err.Error())
        os.Exit(1)
    }
}
```
