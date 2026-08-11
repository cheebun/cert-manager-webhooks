//go:build unit

package namesilo

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// stubClient is a test double for Client.
type stubClient struct {
	listResp   DNSRecordListResponse
	listErr    error
	addResp    Response
	addErr     error
	updateResp Response
	updateErr  error
	deleteResp Response
	deleteErr  error

	addCalls    [][]string // [apiKey, domain, rrhost, rrvalue]
	updateCalls [][]string // [apiKey, domain, rrid, rrhost, rrvalue]
	deleteCalls [][]string // [apiKey, domain, rrid]
}

func (s *stubClient) ListRecords(apiKey, domain string) (DNSRecordListResponse, error) {
	return s.listResp, s.listErr
}

func (s *stubClient) AddRecord(apiKey, domain, rrhost, rrvalue string) (Response, error) {
	s.addCalls = append(s.addCalls, []string{apiKey, domain, rrhost, rrvalue})
	return s.addResp, s.addErr
}

func (s *stubClient) UpdateRecord(apiKey, domain, rrid, rrhost, rrvalue string) (Response, error) {
	s.updateCalls = append(s.updateCalls, []string{apiKey, domain, rrid, rrhost, rrvalue})
	return s.updateResp, s.updateErr
}

func (s *stubClient) DeleteRecord(apiKey, domain, rrid string) (Response, error) {
	s.deleteCalls = append(s.deleteCalls, []string{apiKey, domain, rrid})
	return s.deleteResp, s.deleteErr
}

// ok300 returns a Response with code "300" (success).
func ok300() Response {
	r := Response{}
	r.Reply.Code = "300"
	return r
}

// listOK returns a DNSRecordListResponse with code "300" and optional records.
func listOK(records ...DNSRecordListResponse) DNSRecordListResponse {
	r := DNSRecordListResponse{}
	r.Reply.Code = "300"
	if len(records) > 0 {
		r.Reply.ResourceRecord = records[0].Reply.ResourceRecord
	}
	return r
}

func makeRecord(host, typ, value, id string) struct {
	ResourceID string `json:"record_id"`
	Type       string `json:"type"`
	Host       string `json:"host"`
	Value      string `json:"value"`
} {
	return struct {
		ResourceID string `json:"record_id"`
		Type       string `json:"type"`
		Host       string `json:"host"`
		Value      string `json:"value"`
	}{ResourceID: id, Type: typ, Host: host, Value: value}
}

// -----------------------------------------------------------------------------
// RRHost tests
// -----------------------------------------------------------------------------

func TestRRHost_SimpleSubdomain(t *testing.T) {
	t.Parallel()

	got := RRHost("_acme-challenge.example.com.", "example.com.")
	want := "_acme-challenge"
	if got != want {
		t.Errorf("RRHost(%q, %q) = %q, want %q", "_acme-challenge.example.com.", "example.com.", got, want)
	}
}

func TestRRHost_NestedSubdomain(t *testing.T) {
	t.Parallel()

	got := RRHost("_acme-challenge.sub.example.com.", "example.com.")
	want := "_acme-challenge.sub"
	if got != want {
		t.Errorf("RRHost(%q, %q) = %q, want %q", "_acme-challenge.sub.example.com.", "example.com.", got, want)
	}
}

func TestRRHost_ZoneUppercase_IsLowercasedBeforeTrim(t *testing.T) {
	t.Parallel()

	got := RRHost("_acme-challenge.example.com.", "EXAMPLE.COM.")
	want := "_acme-challenge"
	if got != want {
		t.Errorf("RRHost(%q, %q) = %q, want %q", "_acme-challenge.example.com.", "EXAMPLE.COM.", got, want)
	}
}

func TestRRHost_Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fqdn string
		zone string
		want string
	}{
		{
			name: "simple challenge",
			fqdn: "_acme-challenge.example.com.",
			zone: "example.com.",
			want: "_acme-challenge",
		},
		{
			name: "nested subdomain",
			fqdn: "_acme-challenge.sub.example.com.",
			zone: "example.com.",
			want: "_acme-challenge.sub",
		},
		{
			name: "uppercase zone",
			fqdn: "_acme-challenge.example.com.",
			zone: "EXAMPLE.COM.",
			want: "_acme-challenge",
		},
		{
			name: "deep nesting",
			fqdn: "_acme-challenge.a.b.c.example.com.",
			zone: "example.com.",
			want: "_acme-challenge.a.b.c",
		},
		{
			name: "zone already lowercase",
			fqdn: "_acme-challenge.sub.example.com.",
			zone: "sub.example.com.",
			want: "_acme-challenge",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := RRHost(tc.fqdn, tc.zone)
			if got != tc.want {
				t.Errorf("RRHost(%q, %q) = %q, want %q", tc.fqdn, tc.zone, got, tc.want)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// LoadConfig tests
// -----------------------------------------------------------------------------

func TestLoadConfig_NilInput_ReturnsEmptyConfig(t *testing.T) {
	t.Parallel()
	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.APIKey.Name != "" {
		t.Errorf("expected empty APIKey.Name, got %q", cfg.APIKey.Name)
	}
	if cfg.APIKey.Key != "" {
		t.Errorf("expected empty APIKey.Key, got %q", cfg.APIKey.Key)
	}
}

func TestLoadConfig_ValidJSON_ParsesCorrectly(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"apiKey":{"name":"my-secret","key":"api-key"}}`)
	cfg, err := LoadConfig(&extapi.JSON{Raw: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey.Name != "my-secret" {
		t.Errorf("expected APIKey.Name %q, got %q", "my-secret", cfg.APIKey.Name)
	}
	if cfg.APIKey.Key != "api-key" {
		t.Errorf("expected APIKey.Key %q, got %q", "api-key", cfg.APIKey.Key)
	}
}

func TestLoadConfig_InvalidJSON_ReturnsError(t *testing.T) {
	t.Parallel()

	raw := []byte(`not-valid-json`)
	_, err := LoadConfig(&extapi.JSON{Raw: raw})
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

func TestLoadConfig_EmptyJSONObject_ReturnsEmptyConfig(t *testing.T) {
	t.Parallel()

	raw := []byte(`{}`)
	cfg, err := LoadConfig(&extapi.JSON{Raw: raw})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey.Name != "" || cfg.APIKey.Key != "" {
		t.Errorf("expected empty config for {}, got %+v", cfg)
	}
}

// -----------------------------------------------------------------------------
// Present tests
// -----------------------------------------------------------------------------

func TestPresent_NoExistingRecord_Adds(t *testing.T) {
	t.Parallel()

	stub := &stubClient{
		listResp: listOK(),
		addResp:  ok300(),
	}

	if err := Present(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com.", "token1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.addCalls) != 1 {
		t.Fatalf("expected 1 AddRecord call, got %d", len(stub.addCalls))
	}
	if stub.addCalls[0][3] != "token1" {
		t.Errorf("expected rrvalue=token1, got %q", stub.addCalls[0][3])
	}
	if len(stub.updateCalls) != 0 {
		t.Errorf("expected no UpdateRecord calls, got %d", len(stub.updateCalls))
	}
}

func TestPresent_SameValueAlreadyExists_Noop(t *testing.T) {
	t.Parallel()

	existing := listOK()
	existing.Reply.ResourceRecord = append(existing.Reply.ResourceRecord,
		makeRecord("_acme-challenge", "TXT", "token1", "rid1"))

	stub := &stubClient{listResp: existing}

	if err := Present(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com.", "token1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.addCalls) != 0 || len(stub.updateCalls) != 0 {
		t.Errorf("expected no add/update, got add=%d update=%d", len(stub.addCalls), len(stub.updateCalls))
	}
}

func TestPresent_DifferentValueExists_Updates(t *testing.T) {
	t.Parallel()

	existing := listOK()
	existing.Reply.ResourceRecord = append(existing.Reply.ResourceRecord,
		makeRecord("_acme-challenge", "TXT", "old-token", "rid42"))

	stub := &stubClient{
		listResp:   existing,
		updateResp: ok300(),
	}

	if err := Present(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com.", "new-token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.updateCalls) != 1 {
		t.Fatalf("expected 1 UpdateRecord call, got %d", len(stub.updateCalls))
	}
	call := stub.updateCalls[0]
	if call[2] != "rid42" {
		t.Errorf("expected rrid=rid42, got %q", call[2])
	}
	if call[4] != "new-token" {
		t.Errorf("expected rrvalue=new-token, got %q", call[4])
	}
	if len(stub.addCalls) != 0 {
		t.Errorf("expected no AddRecord, got %d", len(stub.addCalls))
	}
}

func TestPresent_ListError_ReturnsError(t *testing.T) {
	t.Parallel()

	stub := &stubClient{listErr: errors.New("network error")}

	err := Present(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com.", "token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPresent_AddError_ReturnsError(t *testing.T) {
	t.Parallel()

	stub := &stubClient{
		listResp: listOK(),
		addErr:   errors.New("add failed"),
	}

	err := Present(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com.", "token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// -----------------------------------------------------------------------------
// CleanUp tests
// -----------------------------------------------------------------------------

func TestCleanUp_RecordExists_Deletes(t *testing.T) {
	t.Parallel()

	existing := listOK()
	existing.Reply.ResourceRecord = append(existing.Reply.ResourceRecord,
		makeRecord("_acme-challenge", "TXT", "token1", "rid99"))

	stub := &stubClient{
		listResp:   existing,
		deleteResp: ok300(),
	}

	if err := CleanUp(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.deleteCalls) != 1 {
		t.Fatalf("expected 1 DeleteRecord call, got %d", len(stub.deleteCalls))
	}
	if stub.deleteCalls[0][2] != "rid99" {
		t.Errorf("expected rrid=rid99, got %q", stub.deleteCalls[0][2])
	}
}

func TestCleanUp_NoRecord_Noop(t *testing.T) {
	t.Parallel()

	stub := &stubClient{listResp: listOK()}

	if err := CleanUp(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.deleteCalls) != 0 {
		t.Errorf("expected no DeleteRecord calls, got %d", len(stub.deleteCalls))
	}
}

func TestCleanUp_MultipleRecords_DeletesAll(t *testing.T) {
	t.Parallel()

	existing := listOK()
	existing.Reply.ResourceRecord = append(existing.Reply.ResourceRecord,
		makeRecord("_acme-challenge", "TXT", "token-a", "rid1"),
		makeRecord("_acme-challenge", "TXT", "token-b", "rid2"),
	)

	stub := &stubClient{
		listResp:   existing,
		deleteResp: ok300(),
	}

	if err := CleanUp(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.deleteCalls) != 2 {
		t.Fatalf("expected 2 DeleteRecord calls, got %d", len(stub.deleteCalls))
	}
}

func TestCleanUp_ListError_ReturnsError(t *testing.T) {
	t.Parallel()

	stub := &stubClient{listErr: errors.New("network error")}

	err := CleanUp(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com.")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCleanUp_DeleteError_ReturnsError(t *testing.T) {
	t.Parallel()

	existing := listOK()
	existing.Reply.ResourceRecord = append(existing.Reply.ResourceRecord,
		makeRecord("_acme-challenge", "TXT", "token1", "rid1"))

	stub := &stubClient{
		listResp:  existing,
		deleteErr: errors.New("delete failed"),
	}

	err := CleanUp(context.Background(), slog.Default(), stub, "key", "_acme-challenge.example.com.", "example.com.")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
