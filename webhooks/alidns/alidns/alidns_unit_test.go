// SPDX-License-Identifier: MIT

package alidns

import (
	"errors"
	"testing"

	sdkalidns "github.com/alibabacloud-go/alidns-20150109/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Stub implementation of API
// ---------------------------------------------------------------------------

type stubAliDNSAPI struct {
	// DescribeDomainRecords
	describeResp *sdkalidns.DescribeDomainRecordsResponse
	describeErr  error

	// AddDomainRecord
	addResp   *sdkalidns.AddDomainRecordResponse
	addErr    error
	addCalled bool

	// UpdateDomainRecord
	updateResp     *sdkalidns.UpdateDomainRecordResponse
	updateErr      error
	updateCalled   bool
	updateRecordId string

	// DeleteDomainRecord
	deleteDomainResp   *sdkalidns.DeleteDomainRecordResponse
	deleteDomainErr    error
	deleteDomainCalled bool
}

func (s *stubAliDNSAPI) DescribeDomainRecords(req *sdkalidns.DescribeDomainRecordsRequest) (*sdkalidns.DescribeDomainRecordsResponse, error) {
	return s.describeResp, s.describeErr
}

func (s *stubAliDNSAPI) AddDomainRecord(req *sdkalidns.AddDomainRecordRequest) (*sdkalidns.AddDomainRecordResponse, error) {
	s.addCalled = true
	return s.addResp, s.addErr
}

func (s *stubAliDNSAPI) UpdateDomainRecord(req *sdkalidns.UpdateDomainRecordRequest) (*sdkalidns.UpdateDomainRecordResponse, error) {
	s.updateCalled = true
	if req.RecordId != nil {
		s.updateRecordId = *req.RecordId
	}
	return s.updateResp, s.updateErr
}

func (s *stubAliDNSAPI) DeleteDomainRecord(req *sdkalidns.DeleteDomainRecordRequest) (*sdkalidns.DeleteDomainRecordResponse, error) {
	s.deleteDomainCalled = true
	return s.deleteDomainResp, s.deleteDomainErr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newDescribeResponse(records ...*sdkalidns.DescribeDomainRecordsResponseBodyDomainRecordsRecord) *sdkalidns.DescribeDomainRecordsResponse {
	count := int64(len(records))
	return &sdkalidns.DescribeDomainRecordsResponse{
		Body: &sdkalidns.DescribeDomainRecordsResponseBody{
			TotalCount: tea.Int64(count),
			DomainRecords: &sdkalidns.DescribeDomainRecordsResponseBodyDomainRecords{
				Record: records,
			},
		},
	}
}

func newRecord(id, value string) *sdkalidns.DescribeDomainRecordsResponseBodyDomainRecordsRecord {
	return &sdkalidns.DescribeDomainRecordsResponseBodyDomainRecordsRecord{
		RecordId: tea.String(id),
		Value:    tea.String(value),
	}
}

func newAliDNS(stub *stubAliDNSAPI) *AliDNS {
	return &AliDNS{cli: stub}
}

const (
	testFQDN  = "_acme-challenge.example.com."
	testZone  = "example.com."
	testValue = "challenge-token"
)

// ---------------------------------------------------------------------------
// AddRecord unit tests
// ---------------------------------------------------------------------------

func TestAliDNS_AddRecord_Unit(t *testing.T) {
	t.Run("no existing records calls AddDomainRecord", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeResp: newDescribeResponse(),
			addResp:      &sdkalidns.AddDomainRecordResponse{},
		}
		dns := newAliDNS(stub)

		err := dns.AddRecord(testFQDN, testZone, testValue)

		require.NoError(t, err)
		assert.True(t, stub.addCalled, "AddDomainRecord should have been called")
		assert.False(t, stub.updateCalled, "UpdateDomainRecord should not have been called")
	})

	t.Run("existing record with same value returns nil without Add or Update", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeResp: newDescribeResponse(newRecord("rec-1", testValue)),
		}
		dns := newAliDNS(stub)

		err := dns.AddRecord(testFQDN, testZone, testValue)

		require.NoError(t, err)
		assert.False(t, stub.addCalled, "AddDomainRecord should not have been called")
		assert.False(t, stub.updateCalled, "UpdateDomainRecord should not have been called")
	})

	t.Run("existing record with different value calls UpdateDomainRecord with correct RecordId", func(t *testing.T) {
		const existingRecordID = "rec-42"
		stub := &stubAliDNSAPI{
			describeResp: newDescribeResponse(newRecord(existingRecordID, "old-value")),
			updateResp:   &sdkalidns.UpdateDomainRecordResponse{},
		}
		dns := newAliDNS(stub)

		err := dns.AddRecord(testFQDN, testZone, testValue)

		require.NoError(t, err)
		assert.False(t, stub.addCalled, "AddDomainRecord should not have been called")
		assert.True(t, stub.updateCalled, "UpdateDomainRecord should have been called")
		assert.Equal(t, existingRecordID, stub.updateRecordId)
	})

	t.Run("DescribeDomainRecords error is propagated", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeErr: errors.New("describe API error"),
		}
		dns := newAliDNS(stub)

		err := dns.AddRecord(testFQDN, testZone, testValue)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "describe API error")
	})

	t.Run("AddDomainRecord error is propagated", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeResp: newDescribeResponse(),
			addErr:       errors.New("add API error"),
		}
		dns := newAliDNS(stub)

		err := dns.AddRecord(testFQDN, testZone, testValue)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "add API error")
	})

	t.Run("UpdateDomainRecord error is propagated", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeResp: newDescribeResponse(newRecord("rec-1", "old-value")),
			updateErr:    errors.New("update API error"),
		}
		dns := newAliDNS(stub)

		err := dns.AddRecord(testFQDN, testZone, testValue)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "update API error")
	})
}

// ---------------------------------------------------------------------------
// DeleteRecord unit tests
// ---------------------------------------------------------------------------

func TestAliDNS_DeleteRecord_Unit(t *testing.T) {
	t.Run("matching record is deleted by ID", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeResp:     newDescribeResponse(newRecord("rec-1", testValue)),
			deleteDomainResp: &sdkalidns.DeleteDomainRecordResponse{},
		}
		dns := newAliDNS(stub)

		err := dns.DeleteRecord(testFQDN, testZone, testValue)

		require.NoError(t, err)
		assert.True(t, stub.deleteDomainCalled, "DeleteDomainRecord should have been called")
	})

	t.Run("no matching value returns nil without calling Delete", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeResp: newDescribeResponse(newRecord("rec-1", "other-value")),
		}
		dns := newAliDNS(stub)

		err := dns.DeleteRecord(testFQDN, testZone, testValue)

		require.NoError(t, err)
		assert.False(t, stub.deleteDomainCalled, "DeleteDomainRecord should not be called when value does not match")
	})

	t.Run("no records returns nil without calling Delete", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeResp: newDescribeResponse(),
		}
		dns := newAliDNS(stub)

		err := dns.DeleteRecord(testFQDN, testZone, testValue)

		require.NoError(t, err)
		assert.False(t, stub.deleteDomainCalled, "DeleteDomainRecord should not be called when no records exist")
	})

	t.Run("DescribeDomainRecords error is propagated", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeErr: errors.New("describe API error"),
		}
		dns := newAliDNS(stub)

		err := dns.DeleteRecord(testFQDN, testZone, testValue)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "describe API error")
	})

	t.Run("DeleteDomainRecord error is propagated", func(t *testing.T) {
		stub := &stubAliDNSAPI{
			describeResp:    newDescribeResponse(newRecord("rec-1", testValue)),
			deleteDomainErr: errors.New("delete API error"),
		}
		dns := newAliDNS(stub)

		err := dns.DeleteRecord(testFQDN, testZone, testValue)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete API error")
	})
}
