// SPDX-License-Identifier: MIT

package alidns

import (
	alidns "github.com/alibabacloud-go/alidns-20150109/v4/client"
)

// API abstracts the Alibaba Cloud DNS client methods used by AliDNS,
// allowing the concrete *alidns.Client to be replaced with a test stub.
type API interface {
	DescribeDomainRecords(*alidns.DescribeDomainRecordsRequest) (*alidns.DescribeDomainRecordsResponse, error)
	AddDomainRecord(*alidns.AddDomainRecordRequest) (*alidns.AddDomainRecordResponse, error)
	UpdateDomainRecord(*alidns.UpdateDomainRecordRequest) (*alidns.UpdateDomainRecordResponse, error)
	DeleteDomainRecord(*alidns.DeleteDomainRecordRequest) (*alidns.DeleteDomainRecordResponse, error)
}
