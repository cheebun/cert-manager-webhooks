// SPDX-License-Identifier: MIT

package alidns

import (
	"strings"

	sdkalidns "github.com/alibabacloud-go/alidns-20150109/v4/client"
	"github.com/cert-manager/cert-manager/pkg/issuer/acme/dns/util"
)

const (
	DNSRecordType = "TXT"
	ExactSearch   = "EXACT"
)

// AliDNS is a client for manipulating Aliyun DNS records through the OpenAPI.
type AliDNS struct {
	cli API
}

// rrhost returns the relative record name (RR) for an FQDN within a zone.
func rrhost(fqdn, zone string) string {
	return strings.TrimSuffix(util.UnFqdn(fqdn), "."+util.UnFqdn(zone))
}

// AddRecord adds a TXT record for fqdn in zone with the given value.
func (dns *AliDNS) AddRecord(fqdn, zone, value string) error {
	rr := rrhost(fqdn, zone)

	queryReq := new(sdkalidns.DescribeDomainRecordsRequest)
	queryReq.SetDomainName(util.UnFqdn(zone))
	queryReq.SetTypeKeyWord(DNSRecordType)
	queryReq.SetKeyWord(rr)
	queryReq.SetSearchMode(ExactSearch)
	queryResp, err := dns.cli.DescribeDomainRecords(queryReq)
	if err != nil {
		return err
	}

	if *queryResp.Body.TotalCount == 0 {
		req := new(sdkalidns.AddDomainRecordRequest)
		req.SetType(DNSRecordType)
		req.SetDomainName(util.UnFqdn(zone))
		req.SetRR(rr)
		req.SetValue(value)
		_, err = dns.cli.AddDomainRecord(req)
		return err
	}

	record := *queryResp.Body.DomainRecords.Record[0]
	if *record.Value == value {
		return nil
	}

	req := new(sdkalidns.UpdateDomainRecordRequest)
	req.SetRecordId(*record.RecordId)
	req.SetType(DNSRecordType)
	req.SetRR(rr)
	req.SetValue(value)
	_, err = dns.cli.UpdateDomainRecord(req)
	return err
}

// DeleteRecord removes the TXT record for fqdn in zone whose value matches value.
// Other TXT records for the same host (e.g. from a concurrent challenge) are left intact.
func (dns *AliDNS) DeleteRecord(fqdn, zone, value string) error {
	rr := rrhost(fqdn, zone)

	queryReq := new(sdkalidns.DescribeDomainRecordsRequest)
	queryReq.SetDomainName(util.UnFqdn(zone))
	queryReq.SetTypeKeyWord(DNSRecordType)
	queryReq.SetKeyWord(rr)
	queryReq.SetSearchMode(ExactSearch)
	queryResp, err := dns.cli.DescribeDomainRecords(queryReq)
	if err != nil {
		return err
	}

	for _, record := range queryResp.Body.DomainRecords.Record {
		if *record.Value != value {
			continue
		}
		req := new(sdkalidns.DeleteDomainRecordRequest)
		req.SetRecordId(*record.RecordId)
		_, err = dns.cli.DeleteDomainRecord(req)
		return err
	}
	return nil // record not found — already cleaned up
}
