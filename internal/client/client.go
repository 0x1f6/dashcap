// Package client provides an HTTP client for the dashcap REST API.
package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to a dashcap REST API server.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Options configures a Client.
type Options struct {
	Host          string
	Port          int
	Token         string
	TLS           bool
	TLSSkipVerify bool
}

// New creates a Client from the given options.
func New(opts Options) *Client {
	scheme := "http"
	if opts.TLS {
		scheme = "https"
	}
	c := &Client{
		baseURL: fmt.Sprintf("%s://%s:%d", scheme, opts.Host, opts.Port),
		token:   opts.Token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	if opts.TLS && opts.TLSSkipVerify {
		c.httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // user-requested
		}
	}
	return c
}

// APIError is returned when the server responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error %d", e.StatusCode)
}

// HealthResponse is the response from GET /api/v1/health.
type HealthResponse struct {
	Status string `json:"status"`
}

// StatusResponse is the response from GET /api/v1/status.
type StatusResponse struct {
	Interface    string `json:"interface"`
	Uptime       string `json:"uptime"`
	SegmentCount int    `json:"segment_count"`
	TotalPackets int64  `json:"total_packets"`
	TotalBytes   int64  `json:"total_bytes"`
}

// TriggerResponse is the response from POST /api/v1/trigger.
type TriggerResponse struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	Status    string `json:"status"`
	SavedPath string `json:"saved_path,omitempty"`
	Error     string `json:"error,omitempty"`
	Warning   string `json:"warning,omitempty"`
}

// TriggerRequest is the optional JSON body for POST /api/v1/trigger.
type TriggerRequest struct {
	Duration string `json:"duration,omitempty"`
	Since    string `json:"since,omitempty"`
}

// TriggerStatusResponse is the response from GET /api/v1/trigger/{id}.
type TriggerStatusResponse struct {
	ID         string          `json:"id"`
	Timestamp  string          `json:"timestamp"`
	Source     string          `json:"source"`
	Status     string          `json:"status"`
	SavedPath  string          `json:"saved_path,omitempty"`
	Error      string          `json:"error,omitempty"`
	Warning    string          `json:"warning,omitempty"`
	RetryAfter int             `json:"retry_after,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// SegmentInfo is a single ring buffer segment from GET /api/v1/ring.
type SegmentInfo struct {
	Index     int    `json:"Index"`
	Path      string `json:"Path"`
	StartTime string `json:"StartTime"`
	EndTime   string `json:"EndTime"`
	Packets   int64  `json:"Packets"`
	Bytes     int64  `json:"Bytes"`
}

// Health calls GET /api/v1/health.
func (c *Client) Health() (*HealthResponse, error) {
	var resp HealthResponse
	if err := c.get("/api/v1/health", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Status calls GET /api/v1/status.
func (c *Client) Status() (*StatusResponse, error) {
	var resp StatusResponse
	if err := c.get("/api/v1/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Trigger calls POST /api/v1/trigger.
func (c *Client) Trigger(req TriggerRequest) (*TriggerResponse, error) {
	var resp TriggerResponse
	if err := c.post("/api/v1/trigger", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// TriggerStatus calls GET /api/v1/trigger/{id}.
func (c *Client) TriggerStatus(id string) (*TriggerStatusResponse, error) {
	var resp TriggerStatusResponse
	if err := c.get("/api/v1/trigger/"+id, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Triggers calls GET /api/v1/triggers.
func (c *Client) Triggers() ([]*TriggerResponse, error) {
	var resp []*TriggerResponse
	if err := c.get("/api/v1/triggers", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Ring calls GET /api/v1/ring.
func (c *Client) Ring() ([]SegmentInfo, error) {
	var resp []SegmentInfo
	if err := c.get("/api/v1/ring", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) get(path string, dest any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, dest)
}

func (c *Client) post(path string, body any, dest any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, dest)
}

func (c *Client) do(req *http.Request, dest any) error {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		var errBody struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &errBody) == nil && errBody.Error != "" {
			apiErr.Message = errBody.Error
		}
		return apiErr
	}

	if dest != nil {
		if err := json.Unmarshal(data, dest); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}
