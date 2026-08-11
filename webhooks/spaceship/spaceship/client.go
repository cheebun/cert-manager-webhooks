// SPDX-License-Identifier: MIT

package spaceship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// defaultHTTPClient is used when Client.HTTPClient is nil.
// A 30-second timeout prevents a hung Spaceship API from blocking the webhook.
var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

type Client struct {
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
}

func NewClient(apiKey, apiSecret string, httpClient *http.Client) *Client {
	return &Client{
		APIKey:     apiKey,
		APISecret:  apiSecret,
		HTTPClient: httpClient,
	}
}

type DNSSaveRequest struct {
	Force bool           `json:"force"`
	Items []DNSTXTRecord `json:"items"`
}

type DNSTXTRecord struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Name  string `json:"name"`
	TTL   int    `json:"ttl,omitempty"`
}

type DNSListResponse struct {
	Items      []DNSTXTRecord `json:"items"`
	TotalItems int            `json:"totalItems"`
}

func (c *Client) AddTXTRecord(domain, name, value string, ttl int) error {
	record := DNSTXTRecord{
		Type:  "TXT",
		Name:  name,
		Value: value,
		TTL:   ttl,
	}

	request := DNSSaveRequest{
		Force: true,
		Items: []DNSTXTRecord{record},
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal add request: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut,
		fmt.Sprintf("https://spaceship.dev/api/v1/dns/records/%s", domain),
		bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create add request: %w", err)
	}
	req.Header.Set("X-API-Key", c.APIKey)
	req.Header.Set("X-API-Secret", c.APISecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to add TXT record: %s", responseDetail(resp))
	}

	return nil
}

// ListTXTRecords returns all TXT records for the given host name under domain.
func (c *Client) ListTXTRecords(domain, name string) ([]DNSTXTRecord, error) {
	params := url.Values{}
	params.Set("take", "100")
	params.Set("type", "TXT")
	params.Set("name", name)

	rawURL := fmt.Sprintf("https://spaceship.dev/api/v1/dns/records/%s?%s", domain, params.Encode())
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}
	req.Header.Set("X-API-Key", c.APIKey)
	req.Header.Set("X-API-Secret", c.APISecret)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list TXT records: %s", responseDetail(resp))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var listResp DNSListResponse
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("failed to parse list response: %w", err)
	}

	return listResp.Items, nil
}

// RemoveTXTRecord lists all TXT records for name under domain and deletes them all,
// regardless of value — consistent with namesilo/tencent CleanUp behaviour.
func (c *Client) RemoveTXTRecord(domain, name string) error {
	records, err := c.ListTXTRecords(domain, name)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	payload, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("failed to marshal remove request: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete,
		fmt.Sprintf("https://spaceship.dev/api/v1/dns/records/%s", domain),
		bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create remove request: %w", err)
	}
	req.Header.Set("X-API-Key", c.APIKey)
	req.Header.Set("X-API-Secret", c.APISecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to remove TXT record: %s", responseDetail(resp))
	}

	return nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return defaultHTTPClient
}

func responseDetail(resp *http.Response) string {
	if resp == nil {
		return "no response"
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.Status
	}
	if len(body) == 0 {
		return resp.Status
	}
	return fmt.Sprintf("%s: %s", resp.Status, string(body))
}
