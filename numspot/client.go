// Package numspot provides a client for interacting with the Numspot API.
package numspot

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// HTTP header constants.
const (
	UserAgentHeader = "User-Agent"
	PackerUserAgent = "PACKER-NUMSPOT"
	Credentials     = "client_credentials"
	TokenPath       = "/iam/token"
)

// NumspotClient is a client for the Numspot API.
type NumspotClient struct { //nolint:revive // name is intentional to distinguish from the generated Client type
	ID                    string
	Client                *ClientWithResponses
	HTTPClient            *http.Client
	SpaceID               uuid.UUID
	ClientID              uuid.UUID
	ClientSecret          string
	Host                  string
	AccessTokenExpiration time.Time
	tokenMutex            sync.RWMutex
}

// ClientConfig holds configuration for the Numspot client.
type ClientConfig struct {
	Host         string
	SpaceID      string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

// Option is a functional option for configuring the Numspot client.
type Option func(s *NumspotClient) error

// WithHost sets the API host.
func WithHost(host string) Option {
	return func(s *NumspotClient) error {
		s.Host = strings.TrimSuffix(host, "/")
		return nil
	}
}

// WithSpaceID sets the space Id.
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

// WithClientID sets the client Id.
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

// WithClientSecret sets the client secret.
func WithClientSecret(clientSecret string) Option {
	return func(s *NumspotClient) error {
		s.ClientSecret = clientSecret
		return nil
	}
}

// WithCustomHTTPClient sets a custom HTTP client.
func WithCustomHTTPClient(client *http.Client) Option {
	return func(s *NumspotClient) error {
		s.HTTPClient = client
		return nil
	}
}

// NewNumspotClient creates a new Numspot API client.
func NewNumspotClient(ctx context.Context, options ...Option) (*NumspotClient, error) {
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

// GetClient returns the API client, refreshing the token if needed.
func (s *NumspotClient) GetClient(ctx context.Context) (*ClientWithResponses, error) {
	if s.isTokenExpired() {
		s.tokenMutex.Lock()
		defer s.tokenMutex.Unlock()
		if time.Now().After(s.AccessTokenExpiration) {
			if err := s.authenticate(ctx); err != nil {
				return nil, fmt.Errorf("error refreshing access token: %w", err)
			}
		}
	}
	return s.Client, nil
}

// SpaceIDString returns the space ID as a string.
func (s *NumspotClient) SpaceIDString() string {
	return s.SpaceID.String()
}

func (s *NumspotClient) isTokenExpired() bool {
	s.tokenMutex.RLock()
	defer s.tokenMutex.RUnlock()
	return time.Now().After(s.AccessTokenExpiration)
}

func (s *NumspotClient) createClientAPI() error {
	requestEditor := WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
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

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (s *NumspotClient) authenticate(ctx context.Context) error {
	tokenURL := s.Host + TokenPath

	formData := url.Values{}
	formData.Set("grant_type", Credentials)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", buildBasicAuth(s.ClientID.String(), s.ClientSecret))

	httpClient := s.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
					MinVersion:         tls.VersionTLS13,
				},
				Proxy: http.ProxyFromEnvironment,
			},
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return fmt.Errorf("%w: status %d", errAuthenticationFailed, resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	expirationTimeMargin := 5 * 60
	var expirationTime int
	if tokenResp.ExpiresIn > expirationTimeMargin {
		expirationTime = tokenResp.ExpiresIn - expirationTimeMargin
	} else {
		return fmt.Errorf("%w: %d seconds", errTokenExpirationTooShort, tokenResp.ExpiresIn)
	}

	s.AccessTokenExpiration = time.Now().Add(time.Duration(expirationTime) * time.Second)

	bearerToken := tokenResp.AccessToken
	requestEditor := WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
		req.Header.Add(UserAgentHeader, PackerUserAgent)
		return nil
	})

	s.Client, err = NewClientWithResponses(s.Host, s.newTransport(), requestEditor)
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
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: false,
						MinVersion:         tls.VersionTLS13,
					},
					Proxy: http.ProxyFromEnvironment,
				},
			}
		}
		return nil
	}
}
