// SPDX-License-Identifier: MIT

package namesilo

// Client is the interface used by the solver to interact with the Namesilo
// DNS API. The production implementation (HTTPClient) delegates to Call; tests
// supply a stub.
type Client interface {
	ListRecords(apiKey, domain string) (DNSRecordListResponse, error)
	AddRecord(apiKey, domain, rrhost, rrvalue string) (Response, error)
	UpdateRecord(apiKey, domain, rrid, rrhost, rrvalue string) (Response, error)
	DeleteRecord(apiKey, domain, rrid string) (Response, error)
}
