// SPDX-License-Identifier: MIT

package dnspod

import (
	"strings"

	"github.com/cert-manager/cert-manager/pkg/issuer/acme/dns/util"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
)

var txtRecordType = "TXT"

func (s *Solver) ensureTxtRecordsDeleted(client dnspodAPI, zone, fqdn, key string) error {
	recordName := getRecordName(fqdn, zone)
	req := dnspod.NewDescribeRecordListRequest()
	req.Domain = &zone
	req.Subdomain = &recordName
	req.RecordType = &txtRecordType
	resp, err := client.DescribeRecordList(req)
	s.log.Debug(
		"dnspod api request",
		"api", "DescribeRecordList",
		"request", req,
		"response", resp,
	)
	if err != nil {
		if isRecordNotFound(err) {
			s.log.Warn(
				"TXT record not found, skipping deletion",
				"recordName", recordName,
				"zone", zone,
				"request", req,
				"response", resp,
				"error", err,
			)
			return nil
		}
		s.log.Warn(
			"failed to list txt records",
			"recordName", recordName,
			"zone", zone,
			"request", req,
			"response", resp,
			"error", err,
		)
		return err
	}
	for _, record := range resp.Response.RecordList {
		if *record.Name != recordName {
			continue
		}
		if *record.Value != key {
			s.log.Warn("record value does not match, delete anyway", "expect", key, "actual", *record.Value, "record", *record)
		}
		req := dnspod.NewDeleteRecordRequest()
		req.Domain = &zone
		req.RecordId = record.RecordId
		resp, err := client.DeleteRecord(req)
		s.log.Debug("dnspod api request", "api", "DeleteRecord", "request", req, "response", resp)
		if err != nil {
			s.log.Error(
				"failed to delete TXT record",
				"recordValue", *record.Value,
				"recordId", *record.RecordId,
				"zone", zone,
				"request", req,
				"response", resp,
				"error", err,
			)
			return err
		}
	}
	return nil
}

func getRecordName(fqdn, domain string) string {
	name := util.UnFqdn(fqdn)
	if idx := strings.LastIndex(name, "."+domain); idx != -1 {
		return name[:idx]
	}
	return name
}

func (s *Solver) createTxtRecord(client dnspodAPI, zone, fqdn, key, recordLine string, ttl *uint64) error {
	if recordLine == "" {
		recordLine = "默认"
	}
	recordName := getRecordName(fqdn, zone)

	// 1. List existing TXT records for this subdomain.
	listReq := dnspod.NewDescribeRecordListRequest()
	listReq.Domain = common.StringPtr(zone)
	listReq.Subdomain = &recordName
	listReq.RecordType = &txtRecordType
	listResp, err := client.DescribeRecordList(listReq)
	s.log.Debug("dnspod api request", "api", "DescribeRecordList", "request", listReq, "response", listResp)
	if err != nil && !isRecordNotFound(err) {
		s.logErr(err, "dnspod api request failed", "request", listReq, "response", listResp)
		return err
	}

	// 2. Find the first matching TXT record.
	if listResp != nil {
		for _, record := range listResp.Response.RecordList {
			if *record.Name != recordName {
				continue
			}
			// Same value — already correct, nothing to do.
			if *record.Value == key {
				s.log.Info("TXT record already exists, skipping", "recordName", recordName, "zone", zone)
				return nil
			}
			// Different value — update the first record in place.
			s.log.Info("Updating existing TXT record", "recordId", *record.RecordId, "oldValue", *record.Value, "newValue", key)
			modReq := dnspod.NewModifyRecordRequest()
			modReq.Domain = common.StringPtr(zone)
			modReq.RecordId = record.RecordId
			modReq.RecordType = &txtRecordType
			modReq.RecordLine = &recordLine
			modReq.SubDomain = &recordName
			modReq.Value = &key
			modReq.TTL = ttl
			modResp, err := client.ModifyRecord(modReq)
			s.log.Debug("dnspod api request", "api", "ModifyRecord", "request", modReq, "response", modResp)
			if err != nil {
				s.logErr(err, "dnspod api request failed", "request", modReq, "response", modResp)
				return err
			}
			return nil
		}
	}

	// 3. No existing record — create a new one.
	createReq := dnspod.NewCreateRecordRequest()
	createReq.Domain = common.StringPtr(zone)
	createReq.TTL = ttl
	createReq.Value = &key
	createReq.RecordType = &txtRecordType
	createReq.RecordLine = &recordLine
	createReq.SubDomain = &recordName

	createResp, err := client.CreateRecord(createReq)
	s.log.Debug("dnspod api request", "api", "CreateRecord", "request", createReq, "response", createResp)
	if err != nil {
		s.logErr(err, "dnspod api request failed", "request", createReq, "response", createResp)
		return err
	}
	return nil
}
