//go:build integration

package namesilo

import (
	"testing"
)

const namesiloAPIKey = "your-api-key"

func TestAddDNSRecord(t *testing.T) { //nolint:paralleltest
	// sets a record in the DNS provider's console
	resp, err := Call[Response](namesiloAPIKey, "dnsAddRecord", map[string]string{
		"domain":  "961125.xyz",
		"rrtype":  "TXT",
		"rrhost":  "test",
		"rrvalue": "test-cert-manager",
	})
	if err != nil {
		t.Error(err)

		return
	}

	if resp.Reply.Code != "300" {
		t.Error(resp.Reply.Detail)

		return
	}
}

func TestListDNSRecords(t *testing.T) { //nolint:paralleltest
	// fetch the TXT record id
	listResp, err := Call[DNSRecordListResponse](namesiloAPIKey, "dnsListRecords", map[string]string{
		"domain": "961125.xyz",
	})
	if err != nil {
		t.Error(err)

		return
	}

	if listResp.Reply.Code != "300" {
		t.Error(listResp.Reply.Detail)

		return
	}

	for _, r := range listResp.Reply.ResourceRecord {
		if r.Host == "test.961125.xyz" && r.Type == "TXT" && r.Value == "test-cert-manager" {
			t.Log(r.ResourceID, r.Type, r.Host, r.Value)
		}
	}
}

func TestDeleteDNSRecord(t *testing.T) { //nolint:paralleltest
	// delete a record from the DNS provider's console
	deleteResp, err := Call[Response](namesiloAPIKey, "dnsDeleteRecord", map[string]string{
		"domain": "961125.xyz",
		"rrid":   "57adc9fbec1ee06a080e2817332fbb08",
	})
	if err != nil {
		t.Error(err)

		return
	}

	if deleteResp.Reply.Code != "300" {
		t.Error(deleteResp.Reply.Detail)

		return
	}
}
