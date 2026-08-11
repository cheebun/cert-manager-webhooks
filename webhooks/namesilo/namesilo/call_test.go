package namesilo

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// makeRedirectTransport returns an http.RoundTripper that rewrites every
// outgoing request to hit addr (scheme=http) instead of its original host,
// preserving the path and query string. This allows Call to target a local
// httptest.Server without modifying Call itself.
func makeRedirectTransport(addr string) http.RoundTripper {
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req2 := req.Clone(req.Context())
		req2.URL.Scheme = "http"
		req2.URL.Host = addr
		return http.DefaultTransport.RoundTrip(req2)
	})
}

// roundTripFunc is a functional http.RoundTripper used in tests.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// swapDefaultClient replaces the package-level httpClient for the duration of
// the test and restores it via t.Cleanup. Tests using this helper must NOT run
// in parallel because httpClient is a package-level variable.
func swapDefaultClient(t *testing.T, transport http.RoundTripper) {
	t.Helper()
	orig := httpClient
	httpClient = &http.Client{Transport: transport}
	t.Cleanup(func() { httpClient = orig })
}

// -----------------------------------------------------------------------------
// CodeWrapper tests — pure data, safe to run in parallel.
// -----------------------------------------------------------------------------

func TestCodeWrapper_UnmarshalJSON_String(t *testing.T) {
	t.Parallel()
	var c CodeWrapper
	if err := json.Unmarshal([]byte(`"300"`), &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != "300" {
		t.Errorf("expected %q, got %q", "300", c)
	}
}

func TestCodeWrapper_UnmarshalJSON_Integer(t *testing.T) {
	t.Parallel()
	var c CodeWrapper
	if err := json.Unmarshal([]byte(`300`), &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != "300" {
		t.Errorf("expected %q, got %q", "300", c)
	}
}

// -----------------------------------------------------------------------------
// Call tests — these swap the package-level httpClient so they must NOT run
// in parallel with each other or with any other test that does the same.
// -----------------------------------------------------------------------------

func TestCall_HTTP200_ValidJSON(t *testing.T) { //nolint:paralleltest
	body := `{"reply":{"code":"300","detail":"success"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	swapDefaultClient(t, makeRedirectTransport(srv.Listener.Addr().String()))

	resp, err := Call[Response]("test-key", "dnsListRecords", map[string]string{"domain": "example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Reply.Code != "300" {
		t.Errorf("expected code %q, got %q", "300", resp.Reply.Code)
	}
	if resp.Reply.Detail != "success" {
		t.Errorf("expected detail %q, got %q", "success", resp.Reply.Detail)
	}
}

func TestCall_HTTPNon200_ReturnsErrNamesiloHTTPNotOK(t *testing.T) { //nolint:paralleltest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	swapDefaultClient(t, makeRedirectTransport(srv.Listener.Addr().String()))

	_, err := Call[Response]("test-key", "dnsListRecords", map[string]string{"domain": "example.com"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, ErrNamesiloHTTPNotOK) {
		t.Errorf("expected errors.Is(err, ErrNamesiloHTTPNotOK), got: %v", err)
	}
}

func TestCall_HTTP404_ReturnsErrNamesiloHTTPNotOK(t *testing.T) { //nolint:paralleltest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	swapDefaultClient(t, makeRedirectTransport(srv.Listener.Addr().String()))

	_, err := Call[Response]("test-key", "dnsAddRecord", map[string]string{"domain": "example.com"})
	if err == nil {
		t.Fatal("expected an error for 404, got nil")
	}
	if !errors.Is(err, ErrNamesiloHTTPNotOK) {
		t.Errorf("expected ErrNamesiloHTTPNotOK, got: %v", err)
	}
}

func TestCall_InvalidJSON_ReturnsUnmarshalError(t *testing.T) { //nolint:paralleltest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-json`))
	}))
	t.Cleanup(srv.Close)

	swapDefaultClient(t, makeRedirectTransport(srv.Listener.Addr().String()))

	_, err := Call[Response]("test-key", "dnsListRecords", map[string]string{"domain": "example.com"})
	if err == nil {
		t.Fatal("expected an unmarshal error, got nil")
	}
	// Must NOT be ErrNamesiloHTTPNotOK — it is a JSON parse error.
	if errors.Is(err, ErrNamesiloHTTPNotOK) {
		t.Errorf("expected a JSON unmarshal error, not ErrNamesiloHTTPNotOK")
	}
}

func TestCall_TransportError_ReturnsError(t *testing.T) { //nolint:paralleltest
	transportErr := errors.New("simulated dial error")
	swapDefaultClient(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, transportErr
	}))

	_, err := Call[Response]("test-key", "dnsListRecords", map[string]string{"domain": "example.com"})
	if err == nil {
		t.Fatal("expected a transport error, got nil")
	}
	if !errors.Is(err, transportErr) {
		t.Errorf("expected errors.Is(err, transportErr), got: %v", err)
	}
}

func TestCall_QueryParams_ContainKeyAndOperation(t *testing.T) { //nolint:paralleltest
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"reply":{"code":"300","detail":"ok"}}`))
	}))
	t.Cleanup(srv.Close)

	swapDefaultClient(t, makeRedirectTransport(srv.Listener.Addr().String()))

	_, err := Call[Response]("my-api-key", "dnsAddRecord", map[string]string{"domain": "test.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse the captured URL to inspect query params.
	req, _ := http.NewRequest(http.MethodGet, "http://x"+capturedURL, nil)
	params := req.URL.Query()

	if got := params.Get("key"); got != "my-api-key" {
		t.Errorf("expected query param key=%q, got %q", "my-api-key", got)
	}
	if got := params.Get("type"); got != "json" {
		t.Errorf("expected query param type=%q, got %q", "json", got)
	}
	if got := params.Get("version"); got != "1" {
		t.Errorf("expected query param version=%q, got %q", "1", got)
	}
	if got := params.Get("domain"); got != "test.com" {
		t.Errorf("expected query param domain=%q, got %q", "test.com", got)
	}
}

func TestCall_DNSRecordListResponse_Parsed(t *testing.T) { //nolint:paralleltest
	body := `{
		"reply": {
			"code": "300",
			"detail": "success",
			"resource_record": [
				{
					"record_id": "abc123",
					"type": "TXT",
					"host": "_acme-challenge",
					"value": "some-challenge-token"
				}
			]
		}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	swapDefaultClient(t, makeRedirectTransport(srv.Listener.Addr().String()))

	resp, err := Call[DNSRecordListResponse]("test-key", "dnsListRecords", map[string]string{"domain": "example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Reply.Code != "300" {
		t.Errorf("expected code %q, got %q", "300", resp.Reply.Code)
	}
	if len(resp.Reply.ResourceRecord) != 1 {
		t.Fatalf("expected 1 resource record, got %d", len(resp.Reply.ResourceRecord))
	}
	rr := resp.Reply.ResourceRecord[0]
	if rr.ResourceID != "abc123" {
		t.Errorf("expected ResourceID %q, got %q", "abc123", rr.ResourceID)
	}
	if rr.Type != "TXT" {
		t.Errorf("expected Type %q, got %q", "TXT", rr.Type)
	}
	if rr.Host != "_acme-challenge" {
		t.Errorf("expected Host %q, got %q", "_acme-challenge", rr.Host)
	}
	if rr.Value != "some-challenge-token" {
		t.Errorf("expected Value %q, got %q", "some-challenge-token", rr.Value)
	}
}

// TestCall_CodeWrapper_IntegerInResponse verifies that a numeric (unquoted)
// code field in the JSON response body is decoded correctly by CodeWrapper.
func TestCall_CodeWrapper_IntegerInResponse(t *testing.T) { //nolint:paralleltest
	body := `{"reply":{"code":300,"detail":"success numeric code"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	swapDefaultClient(t, makeRedirectTransport(srv.Listener.Addr().String()))

	resp, err := Call[Response]("test-key", "dnsListRecords", map[string]string{"domain": "example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Reply.Code != "300" {
		t.Errorf("expected code %q after integer decode, got %q", "300", resp.Reply.Code)
	}
}

func TestCall_EmptyResourceRecordList(t *testing.T) { //nolint:paralleltest
	body := `{"reply":{"code":"300","detail":"success","resource_record":[]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	swapDefaultClient(t, makeRedirectTransport(srv.Listener.Addr().String()))

	resp, err := Call[DNSRecordListResponse]("test-key", "dnsListRecords", map[string]string{"domain": "example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Reply.ResourceRecord) != 0 {
		t.Errorf("expected 0 resource records, got %d", len(resp.Reply.ResourceRecord))
	}
}
