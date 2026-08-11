// SPDX-License-Identifier: MIT

package dnspod

import (
	"errors"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	terrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

// --- stub implementation of dnspodAPI ---

type stubClient struct {
	// DescribeRecordList
	describeResp *dnspod.DescribeRecordListResponse
	describeErr  error

	// CreateRecord
	createCalled bool
	createErr    error

	// ModifyRecord
	modifyCalled   bool
	modifyRecordId uint64 // the RecordId that was passed in
	modifyErr      error

	// DeleteRecord
	deletedIds []uint64
	deleteErr  error
}

func (s *stubClient) DescribeRecordList(req *dnspod.DescribeRecordListRequest) (*dnspod.DescribeRecordListResponse, error) {
	return s.describeResp, s.describeErr
}

func (s *stubClient) CreateRecord(req *dnspod.CreateRecordRequest) (*dnspod.CreateRecordResponse, error) {
	s.createCalled = true
	if s.createErr != nil {
		return nil, s.createErr
	}
	return &dnspod.CreateRecordResponse{}, nil
}

func (s *stubClient) ModifyRecord(req *dnspod.ModifyRecordRequest) (*dnspod.ModifyRecordResponse, error) {
	s.modifyCalled = true
	if req.RecordId != nil {
		s.modifyRecordId = *req.RecordId
	}
	if s.modifyErr != nil {
		return nil, s.modifyErr
	}
	return &dnspod.ModifyRecordResponse{}, nil
}

func (s *stubClient) DeleteRecord(req *dnspod.DeleteRecordRequest) (*dnspod.DeleteRecordResponse, error) {
	if s.deleteErr != nil {
		return nil, s.deleteErr
	}
	if req.RecordId != nil {
		s.deletedIds = append(s.deletedIds, *req.RecordId)
	}
	return &dnspod.DeleteRecordResponse{}, nil
}

// recordNotFoundErr returns a Tencent SDK error that signals "no data of record".
func recordNotFoundErr() error {
	return terrors.NewTencentCloudSDKError("ResourceNotFound.NoDataOfRecord", "no record", "fake-req-id")
}

// makeDescribeResp builds a DescribeRecordListResponse with the given records.
func makeDescribeResp(items []*dnspod.RecordListItem) *dnspod.DescribeRecordListResponse {
	resp := &dnspod.DescribeRecordListResponse{}
	resp.Response = &dnspod.DescribeRecordListResponseParams{
		RecordList: items,
	}
	return resp
}

// newSolverForTest returns a Solver that has a valid (JSON) logger so that
// the s.log.* calls inside record.go don't panic.
func newSolverForTest() *Solver {
	return NewSolver()
}

// --- helper constants ---

const (
	testZone = "example.com"
	testFQDN = "_acme-challenge.example.com."
	testKey  = "token-value"
	// getRecordName("_acme-challenge.example.com.", "example.com") == "_acme-challenge"
	testRecordName = "_acme-challenge"
)

// ============================================================
// Tests for createTxtRecord
// ============================================================

func TestCreateTxtRecord_NoExistingRecords_CallsCreate(t *testing.T) {
	stub := &stubClient{
		// Empty record list (nil response simulates ResourceNotFound)
		describeErr: recordNotFoundErr(),
	}
	s := newSolverForTest()
	err := s.createTxtRecord(stub, testZone, testFQDN, testKey, "", nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !stub.createCalled {
		t.Error("expected CreateRecord to be called, but it was not")
	}
	if stub.modifyCalled {
		t.Error("expected ModifyRecord NOT to be called, but it was")
	}
}

func TestCreateTxtRecord_SameValue_ReturnsNilWithoutCreate(t *testing.T) {
	stub := &stubClient{
		describeResp: makeDescribeResp([]*dnspod.RecordListItem{
			{
				RecordId: common.Uint64Ptr(42),
				Name:     common.StringPtr(testRecordName),
				Value:    common.StringPtr(testKey),
			},
		}),
	}
	s := newSolverForTest()
	err := s.createTxtRecord(stub, testZone, testFQDN, testKey, "", nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if stub.createCalled {
		t.Error("expected CreateRecord NOT to be called, but it was")
	}
	if stub.modifyCalled {
		t.Error("expected ModifyRecord NOT to be called, but it was")
	}
}

func TestCreateTxtRecord_DifferentValue_CallsModify(t *testing.T) {
	const recordID uint64 = 99
	stub := &stubClient{
		describeResp: makeDescribeResp([]*dnspod.RecordListItem{
			{
				RecordId: common.Uint64Ptr(recordID),
				Name:     common.StringPtr(testRecordName),
				Value:    common.StringPtr("old-token"),
			},
		}),
	}
	s := newSolverForTest()
	err := s.createTxtRecord(stub, testZone, testFQDN, testKey, "", nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !stub.modifyCalled {
		t.Error("expected ModifyRecord to be called, but it was not")
	}
	if stub.modifyRecordId != recordID {
		t.Errorf("ModifyRecord called with RecordId %d, want %d", stub.modifyRecordId, recordID)
	}
	if stub.createCalled {
		t.Error("expected CreateRecord NOT to be called, but it was")
	}
}

func TestCreateTxtRecord_ResourceNotFoundError_CallsCreate(t *testing.T) {
	// DescribeRecordList returns the "no data" error — must be treated as empty list.
	stub := &stubClient{
		describeErr: recordNotFoundErr(),
	}
	s := newSolverForTest()
	err := s.createTxtRecord(stub, testZone, testFQDN, testKey, "", nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !stub.createCalled {
		t.Error("expected CreateRecord to be called, but it was not")
	}
}

func TestCreateTxtRecord_DescribeError_ReturnsError(t *testing.T) {
	someErr := errors.New("network failure")
	stub := &stubClient{
		describeErr: someErr,
	}
	s := newSolverForTest()
	err := s.createTxtRecord(stub, testZone, testFQDN, testKey, "", nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if stub.createCalled {
		t.Error("expected CreateRecord NOT to be called after DescribeRecordList error")
	}
}

func TestCreateTxtRecord_ModifyError_ReturnsError(t *testing.T) {
	stub := &stubClient{
		describeResp: makeDescribeResp([]*dnspod.RecordListItem{
			{
				RecordId: common.Uint64Ptr(7),
				Name:     common.StringPtr(testRecordName),
				Value:    common.StringPtr("different-value"),
			},
		}),
		modifyErr: errors.New("modify failed"),
	}
	s := newSolverForTest()
	err := s.createTxtRecord(stub, testZone, testFQDN, testKey, "", nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestCreateTxtRecord_CreateError_ReturnsError(t *testing.T) {
	stub := &stubClient{
		describeErr: recordNotFoundErr(),
		createErr:   errors.New("create failed"),
	}
	s := newSolverForTest()
	err := s.createTxtRecord(stub, testZone, testFQDN, testKey, "", nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// ============================================================
// Tests for ensureTxtRecordsDeleted
// ============================================================

func TestEnsureTxtRecordsDeleted_NoRecords_ReturnsNil(t *testing.T) {
	stub := &stubClient{
		describeErr: recordNotFoundErr(),
	}
	s := newSolverForTest()
	err := s.ensureTxtRecordsDeleted(stub, testZone, testFQDN, testKey)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(stub.deletedIds) != 0 {
		t.Errorf("expected no DeleteRecord calls, got %d", len(stub.deletedIds))
	}
}

func TestEnsureTxtRecordsDeleted_RecordsFound_DeletesAll(t *testing.T) {
	stub := &stubClient{
		describeResp: makeDescribeResp([]*dnspod.RecordListItem{
			{RecordId: common.Uint64Ptr(1), Name: common.StringPtr(testRecordName), Value: common.StringPtr(testKey)},
			{RecordId: common.Uint64Ptr(2), Name: common.StringPtr(testRecordName), Value: common.StringPtr("other-value")},
		}),
	}
	s := newSolverForTest()
	err := s.ensureTxtRecordsDeleted(stub, testZone, testFQDN, testKey)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(stub.deletedIds) != 2 {
		t.Errorf("expected 2 DeleteRecord calls, got %d", len(stub.deletedIds))
	}
}

func TestEnsureTxtRecordsDeleted_DifferentName_NotDeleted(t *testing.T) {
	// The response contains a record whose Name doesn't match the subdomain.
	stub := &stubClient{
		describeResp: makeDescribeResp([]*dnspod.RecordListItem{
			{RecordId: common.Uint64Ptr(5), Name: common.StringPtr("_other-subdomain"), Value: common.StringPtr(testKey)},
		}),
	}
	s := newSolverForTest()
	err := s.ensureTxtRecordsDeleted(stub, testZone, testFQDN, testKey)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(stub.deletedIds) != 0 {
		t.Errorf("expected no DeleteRecord calls for mismatched name, got %d", len(stub.deletedIds))
	}
}

func TestEnsureTxtRecordsDeleted_DeleteError_ReturnsError(t *testing.T) {
	stub := &stubClient{
		describeResp: makeDescribeResp([]*dnspod.RecordListItem{
			{RecordId: common.Uint64Ptr(3), Name: common.StringPtr(testRecordName), Value: common.StringPtr(testKey)},
		}),
		deleteErr: errors.New("delete failed"),
	}
	s := newSolverForTest()
	err := s.ensureTxtRecordsDeleted(stub, testZone, testFQDN, testKey)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
