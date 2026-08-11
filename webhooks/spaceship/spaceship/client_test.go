// SPDX-License-Identifier: MIT

package spaceship

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func okResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}
}

func statusResponse(code int, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Status:     fmt.Sprintf("%d %s", code, http.StatusText(code)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestAddTXTRecordSuccess(t *testing.T) {
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPut {
					t.Fatalf("method = %s, want %s", r.Method, http.MethodPut)
				}
				if r.URL.String() != "https://spaceship.dev/api/v1/dns/records/example.com" {
					t.Fatalf("url = %s", r.URL.String())
				}
				if got := r.Header.Get("X-API-Key"); got != "key" {
					t.Fatalf("X-API-Key = %s", got)
				}
				if got := r.Header.Get("X-API-Secret"); got != "secret" {
					t.Fatalf("X-API-Secret = %s", got)
				}
				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("Content-Type = %s", got)
				}

				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				var req DNSSaveRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if !req.Force {
					t.Fatalf("Force = false")
				}
				if len(req.Items) != 1 {
					t.Fatalf("Items len = %d", len(req.Items))
				}
				item := req.Items[0]
				if item.Type != "TXT" || item.Name != "name" || item.Value != "value" || item.TTL != 60 {
					t.Fatalf("item = %+v", item)
				}

				return okResponse(), nil
			}),
		},
	}

	if err := client.AddTXTRecord("example.com", "name", "value", 60); err != nil {
		t.Fatalf("AddTXTRecord error: %v", err)
	}
}

func TestAddTXTRecordStatusError(t *testing.T) {
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return statusResponse(http.StatusBadRequest, "bad request"), nil
			}),
		},
	}

	if err := client.AddTXTRecord("example.com", "name", "value", 60); err == nil {
		t.Fatalf("expected error")
	} else if !strings.Contains(err.Error(), "400 Bad Request") {
		t.Fatalf("error = %v", err)
	}
}

func TestAddTXTRecordNoContentSuccess(t *testing.T) {
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return statusResponse(http.StatusNoContent, ""), nil
			}),
		},
	}

	if err := client.AddTXTRecord("example.com", "name", "value", 60); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestAddTXTRecordTransportError(t *testing.T) {
	wantErr := errors.New("transport error")

	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, wantErr
			}),
		},
	}

	if err := client.AddTXTRecord("example.com", "name", "value", 60); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func TestListTXTRecordsSuccess(t *testing.T) {
	listBody := `{"items":[{"type":"TXT","name":"_acme-challenge","value":"token1"},{"type":"TXT","name":"_acme-challenge","value":"token2"}],"totalItems":2}`
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				if r.URL.Path != "/api/v1/dns/records/example.com" {
					t.Fatalf("path = %s", r.URL.Path)
				}
				if got := r.URL.Query().Get("name"); got != "_acme-challenge" {
					t.Fatalf("name param = %s", got)
				}
				if got := r.URL.Query().Get("type"); got != "TXT" {
					t.Fatalf("type param = %s", got)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(listBody)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	records, err := client.ListTXTRecords("example.com", "_acme-challenge")
	if err != nil {
		t.Fatalf("ListTXTRecords error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records len = %d, want 2", len(records))
	}
}

func TestListTXTRecordsNotFound(t *testing.T) {
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return statusResponse(http.StatusNotFound, ""), nil
			}),
		},
	}

	records, err := client.ListTXTRecords("example.com", "_acme-challenge")
	if err != nil {
		t.Fatalf("expected nil error for 404, got %v", err)
	}
	if records != nil {
		t.Fatalf("expected nil records for 404")
	}
}

// TestRemoveTXTRecordSuccess verifies that RemoveTXTRecord lists records then deletes them all.
func TestRemoveTXTRecordSuccess(t *testing.T) {
	listBody := `{"items":[{"type":"TXT","name":"_acme-challenge","value":"token1"},{"type":"TXT","name":"_acme-challenge","value":"token2"}],"totalItems":2}`
	calls := 0
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method == http.MethodGet {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader(listBody)),
						Header:     make(http.Header),
					}, nil
				}
				if r.Method != http.MethodDelete {
					t.Fatalf("method = %s, want DELETE", r.Method)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				var records []DNSTXTRecord
				if err := json.Unmarshal(body, &records); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(records) != 2 {
					t.Fatalf("DELETE body len = %d, want 2", len(records))
				}
				return okResponse(), nil
			}),
		},
	}

	if err := client.RemoveTXTRecord("example.com", "_acme-challenge"); err != nil {
		t.Fatalf("RemoveTXTRecord error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 HTTP calls (GET + DELETE), got %d", calls)
	}
}

// TestRemoveTXTRecordNoRecords verifies that RemoveTXTRecord returns nil when no records exist.
func TestRemoveTXTRecordNoRecords(t *testing.T) {
	listBody := `{"items":[],"totalItems":0}`
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected method %s, only GET expected (no records to delete)", r.Method)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(listBody)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	if err := client.RemoveTXTRecord("example.com", "_acme-challenge"); err != nil {
		t.Fatalf("expected nil error for empty list, got %v", err)
	}
}

func TestRemoveTXTRecordStatusError(t *testing.T) {
	listBody := `{"items":[{"type":"TXT","name":"_acme-challenge","value":"token"}],"totalItems":1}`
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodGet {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader(listBody)),
						Header:     make(http.Header),
					}, nil
				}
				return statusResponse(http.StatusBadRequest, "bad request"), nil
			}),
		},
	}

	if err := client.RemoveTXTRecord("example.com", "_acme-challenge"); err == nil {
		t.Fatalf("expected error")
	} else if !strings.Contains(err.Error(), "400 Bad Request") {
		t.Fatalf("error = %v", err)
	}
}

func TestRemoveTXTRecordNoContentSuccess(t *testing.T) {
	listBody := `{"items":[{"type":"TXT","name":"_acme-challenge","value":"token"}],"totalItems":1}`
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodGet {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     "200 OK",
						Body:       io.NopCloser(strings.NewReader(listBody)),
						Header:     make(http.Header),
					}, nil
				}
				return statusResponse(http.StatusNoContent, ""), nil
			}),
		},
	}

	if err := client.RemoveTXTRecord("example.com", "_acme-challenge"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestRemoveTXTRecordTransportError(t *testing.T) {
	wantErr := errors.New("transport error")

	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, wantErr
			}),
		},
	}

	if err := client.RemoveTXTRecord("example.com", "_acme-challenge"); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

// --- Additional ListTXTRecords tests ---

func TestListTXTRecordsTransportError(t *testing.T) {
	wantErr := errors.New("transport error")
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, wantErr
			}),
		},
	}

	records, err := client.ListTXTRecords("example.com", "_acme-challenge")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if records != nil {
		t.Fatalf("expected nil records on transport error, got %v", records)
	}
}

func TestListTXTRecordsNon200Non404StatusError(t *testing.T) {
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return statusResponse(http.StatusInternalServerError, "server error"), nil
			}),
		},
	}

	records, err := client.ListTXTRecords("example.com", "_acme-challenge")
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}
	if !strings.Contains(err.Error(), "500 Internal Server Error") {
		t.Fatalf("error = %v, want message to contain status", err)
	}
	if records != nil {
		t.Fatalf("expected nil records on error, got %v", records)
	}
}

func TestListTXTRecordsInvalidJSON(t *testing.T) {
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader("not-json{")),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	records, err := client.ListTXTRecords("example.com", "_acme-challenge")
	if err == nil {
		t.Fatalf("expected parse error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse list response") {
		t.Fatalf("error = %v, want parse error message", err)
	}
	if records != nil {
		t.Fatalf("expected nil records on parse error, got %v", records)
	}
}

func TestListTXTRecordsEmptyItems(t *testing.T) {
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(`{"items":[],"totalItems":0}`)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	records, err := client.ListTXTRecords("example.com", "_acme-challenge")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records == nil {
		t.Fatalf("expected empty slice (not nil) for empty items array")
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestListTXTRecordsMultipleItems(t *testing.T) {
	listBody := `{"items":[{"type":"TXT","name":"_acme-challenge","value":"token1","ttl":60},{"type":"TXT","name":"_acme-challenge","value":"token2","ttl":120},{"type":"TXT","name":"_acme-challenge","value":"token3","ttl":300}],"totalItems":3}`
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Body:       io.NopCloser(strings.NewReader(listBody)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	records, err := client.ListTXTRecords("example.com", "_acme-challenge")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("records len = %d, want 3", len(records))
	}
	wantValues := []string{"token1", "token2", "token3"}
	for i, r := range records {
		if r.Value != wantValues[i] {
			t.Fatalf("records[%d].Value = %q, want %q", i, r.Value, wantValues[i])
		}
		if r.Type != "TXT" {
			t.Fatalf("records[%d].Type = %q, want TXT", i, r.Type)
		}
		if r.Name != "_acme-challenge" {
			t.Fatalf("records[%d].Name = %q, want _acme-challenge", i, r.Name)
		}
	}
}

// --- Additional RemoveTXTRecord tests ---

func TestRemoveTXTRecordList404NoDelete(t *testing.T) {
	deleteCalled := false
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodDelete {
					deleteCalled = true
					t.Errorf("DELETE should not be called when list returns 404")
				}
				return statusResponse(http.StatusNotFound, "not found"), nil
			}),
		},
	}

	if err := client.RemoveTXTRecord("example.com", "_acme-challenge"); err != nil {
		t.Fatalf("expected nil error when list returns 404, got %v", err)
	}
	if deleteCalled {
		t.Fatalf("DELETE was called despite 404 list response")
	}
}

func TestRemoveTXTRecordListTransportErrorNoDelete(t *testing.T) {
	wantErr := errors.New("list transport error")
	deleteCalled := false
	client := &Client{
		APIKey:    "key",
		APISecret: "secret",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodDelete {
					deleteCalled = true
					t.Errorf("DELETE should not be called when list fails")
					return okResponse(), nil
				}
				return nil, wantErr
			}),
		},
	}

	if err := client.RemoveTXTRecord("example.com", "_acme-challenge"); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if deleteCalled {
		t.Fatalf("DELETE was called despite list transport error")
	}
}
