// Package pinata is used to send requests to the Pinata pinByHash endpoint (https://pinata.cloud/documentation#PinByHash).
package pinata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

const (

	// DefaultEndpoint is the API endpoint to use for pinata.
	DefaultEndpoint = "https://api.pinata.cloud/pinning/addHashToPinQueue"

	// DefaultAuthEndpoint is the API endpoint to use for authentication testing.
	DefaultAuthEndpoint = "https://api.pinata.cloud/data/testAuthentication"

	// DefaultTimeout is maximum response time to wait before the client hangs up.
	DefaultTimeout = 42 * time.Second

	// MetaDataLimit is the maximum number of key value pairs permitted by Pinata in metadata.
	MetaDataLimit = 10
)

var (

	// ErrMetaLimit is issued when too many key value pairs are added to the Pinata metadata.
	ErrMetaLimit = fmt.Errorf("metadata capacity reached (only %d key values pairs allowed)", MetaDataLimit)
)

// APIResponse is a struct to unmarshal
// the Pinata API response into.
type APIResponse struct {
	ID       string
	Ipfshash string
	Status   string
	Name     string
}

// Client handles pinata requests. It wraps the
// standard http.Client and adds the pinata
// requirements.
type Client struct {
	http.Client
	hostNode     string // the host node where the CIDs are pinned
	apiKey       string // the pinata API key
	apiSecret    string // the pinata API secret
	apiEndpoint  string // the pinata API endpoint for pinByHash
	authEndpoint string // the pinata API endpoint for authentication testing
}

// NewClient takes an API Key, API Secret and
// the hostNode where the pinned CIDs are and
// returns a Client and any error.
//
// Note: host can be blank ("") and it will
// be left out of pinByHash requests.
func NewClient(key, secret, host string) (*Client, error) {
	client := Client{
		hostNode:     host,
		apiKey:       key,
		apiSecret:    secret,
		apiEndpoint:  DefaultEndpoint,
		authEndpoint: DefaultAuthEndpoint,
	}
	client.Timeout = DefaultTimeout

	// check credentials
	resp, err := client.TestAuthentication()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to authenticate, bad status code: %d", resp.StatusCode)
	}
	return &client, nil
}

// NewPinataRequest returns a http.NewRequest
// with the required headers required by
// Pinata.
func (client *Client) NewPinataRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("pinata_api_key", client.apiKey)
	req.Header.Set("pinata_secret_api_key", client.apiSecret)
	return req, nil
}

// TestAuthentication is used to test Pinata
// authentication.
func (client *Client) TestAuthentication() (*http.Response, error) {
	req, err := client.NewPinataRequest("GET", client.authEndpoint, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// PinByHashWithMetadata attaches metadata
// to the Pinata request.
//
// For more details on using the metadata,
// see:
// pinata.cloud/documentation#PinByHash
func (client *Client) PinByHashWithMetadata(cid string, metadata *Metadata) (*APIResponse, error) {
	request := PinQueueRequest{
		Cid: cid,
	}
	if client.hostNode != "" {
		request.PinataOptions = make(map[string]interface{})
		request.PinataOptions["hostNodes"] = []string{client.hostNode}
	}

	if metadata.Name != "" || len(metadata.Keyvalues) > 0 {
		request.Metadata = metadata
	}
	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := client.NewPinataRequest("POST", client.apiEndpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	apiResp := &APIResponse{}
	if err := json.Unmarshal(body, apiResp); err != nil {
		return nil, err
	}
	return apiResp, nil
}

// PinQueueRequest is the request structure as outlined in
// https://pinata.cloud/documentation#PinByHash
type PinQueueRequest struct {
	Cid           string                 `json:"hashToPin"`
	PinataOptions map[string]interface{} `json:"pinataOptions,omitempty"`
	Metadata      *Metadata              `json:"pinataMetadata,omitempty"`
}

// Metadata is a general purpose data structure
// as described in the Pinata API docs.
//
// "Similarily to other endpoints, Pinata also
// allows metadata to be attached to when
// pinning content by hash. This metadata
// can later be used for easy querying on
// what you've pinned with our userPinList
// request."
type Metadata struct {

	// Name is a custom name you can have for referencing your pinned content
	Name string `json:"name,omitempty"`

	// Keyvalues encode the metadata (as strings, numbers or ISO dates)
	Keyvalues map[string]interface{} `json:"keyvalues,omitempty"`
}

// NewMetadata initialises a Meatadata struct
// with the provided name.
func NewMetadata(name string) *Metadata {
	return &Metadata{
		Name:      name,
		Keyvalues: make(map[string]interface{}),
	}
}

// Add will set a keyvalue pair in the metadata struct.
// Only Strings, Numbers and ISO date are permitted
// and there is a limit of 10 pairs.
func (metadata *Metadata) Add(key string, value interface{}) error {

	// only 10 items currently allowed by Pinata.
	if len(metadata.Keyvalues) == MetaDataLimit {
		return ErrMetaLimit
	}

	// check we are adding one of the permitted types
	switch t := value.(type) {
	case string, int, int64, float32, float64:

		// string and numbers are fine
		metadata.Keyvalues[key] = value
	case time.Time:

		// format time with the RFC3339 implementation of ISO8601
		metadata.Keyvalues[key] = value.(time.Time).UTC().Format(time.RFC3339)
	default:
		return fmt.Errorf("can't add unsupported type to metadata (%T)", t)
	}
	return nil
}
