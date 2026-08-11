// SPDX-License-Identifier: MIT

package namesilo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Response represents a generic Namesilo API response.
type Response struct {
	Reply struct {
		Code   CodeWrapper `json:"code"`
		Detail string      `json:"detail"`
	} `json:"reply"`
}

// DNSRecordListResponse represents the response from the namesilo API.
// See https://www.namesilo.com/api-reference#dns/dns-list-records
type DNSRecordListResponse struct {
	Reply struct {
		Code           CodeWrapper `json:"code"`
		Detail         string      `json:"detail"`
		ResourceRecord []struct {
			ResourceID string `json:"record_id"`
			Type       string `json:"type"`
			Host       string `json:"host"`
			Value      string `json:"value"`
		} `json:"resource_record"`
	} `json:"reply"`
}

// CodeWrapper holds the API response code string.
// FIXME: namesilo API response code sometimes is string instead of int
type CodeWrapper string

func (c *CodeWrapper) UnmarshalJSON(data []byte) error {
	*c = CodeWrapper(strings.Trim(string(data), "\""))
	return nil
}

// httpClient is the HTTP client used by Call. It is a package-level variable
// so that tests can substitute a client with a custom Transport without
// touching http.DefaultClient.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// Call makes a generic HTTP GET request to the Namesilo API.
//
//nolint:ireturn
func Call[Resp any](apiKey string, operation string, params map[string]string) (Resp, error) {
	var resp Resp

	backgroundCtx := context.Background()

	req, err := http.NewRequestWithContext(backgroundCtx, http.MethodGet, "https://www.namesilo.com/api/"+operation, nil)
	if err != nil {
		return resp, fmt.Errorf("error creating http request: %w", err)
	}

	queryParams := req.URL.Query()
	queryParams.Set("version", "1")
	queryParams.Set("type", "json")
	queryParams.Set("key", apiKey)
	for k, v := range params {
		queryParams.Set(k, v)
	}
	req.URL.RawQuery = queryParams.Encode()

	httpResp, err := httpClient.Do(req)
	if err != nil {
		return resp, fmt.Errorf("error making http request: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

	if httpResp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("error response from namesilo api: %s, %w", httpResp.Status, ErrNamesiloHTTPNotOK)
	}

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return resp, fmt.Errorf("error reading response body: %w", err)
	}

	if err = json.Unmarshal(responseBody, &resp); err != nil {
		slog.ErrorContext(backgroundCtx, "unmarshal error", "body", responseBody, "err", err)
		return resp, fmt.Errorf("namesilo unmarshal json fail: %w", err)
	}

	return resp, nil
}

// GetDomainFromZone strips the trailing dot from a DNS zone name.
func GetDomainFromZone(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}
