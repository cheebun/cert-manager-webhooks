// SPDX-License-Identifier: MIT

package dnspod

import (
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

// dnspodAPI is the subset of the Tencent DNSPod client used by this package.
// Defining it as an interface allows unit tests to supply a stub without
// making real HTTP calls.
type dnspodAPI interface {
	DescribeRecordList(*dnspod.DescribeRecordListRequest) (*dnspod.DescribeRecordListResponse, error)
	CreateRecord(*dnspod.CreateRecordRequest) (*dnspod.CreateRecordResponse, error)
	ModifyRecord(*dnspod.ModifyRecordRequest) (*dnspod.ModifyRecordResponse, error)
	DeleteRecord(*dnspod.DeleteRecordRequest) (*dnspod.DeleteRecordResponse, error)
}
