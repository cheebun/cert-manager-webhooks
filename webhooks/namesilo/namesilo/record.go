// SPDX-License-Identifier: MIT

package namesilo

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// RRHost computes the relative record host for fqdn within zone.
func RRHost(fqdn, zone string) string {
	return strings.TrimSuffix(fqdn, "."+strings.ToLower(zone))
}

// Present ensures the TXT record exists for the ACME challenge.
// It creates, updates, or skips the record as needed.
func Present(ctx context.Context, log *slog.Logger, nsClient Client, apiKey, fqdn, zone, key string) error {
	domain := GetDomainFromZone(zone)
	rrhost := RRHost(fqdn, zone)

	listResp, err := nsClient.ListRecords(apiKey, domain)
	if err != nil {
		return fmt.Errorf("listing DNS records for %s: %w", domain, err)
	}
	if listResp.Reply.Code != "300" {
		return fmt.Errorf("error listing DNS records: %s, %w", listResp.Reply.Detail, ErrTXTRecordFetch)
	}

	for _, r := range listResp.Reply.ResourceRecord {
		if r.Host != rrhost || r.Type != "TXT" {
			continue
		}
		if r.Value == key {
			log.InfoContext(ctx, "TXT record already exists, skipping",
				"key", key, "fqdn", fqdn)
			return nil
		}
		log.InfoContext(ctx, "Updating existing TXT record",
			"recordID", r.ResourceID, "fqdn", fqdn, "oldValue", r.Value)
		updateResp, updateErr := nsClient.UpdateRecord(apiKey, domain, r.ResourceID, rrhost, key)
		if updateErr != nil {
			return fmt.Errorf("updating TXT record %s for %s: %w", r.ResourceID, fqdn, updateErr)
		}
		if updateResp.Reply.Code != "300" {
			log.ErrorContext(ctx, "Error updating TXT record",
				"key", key, "fqdn", fqdn, "detail", updateResp.Reply.Detail)
			return fmt.Errorf("error updating TXT record: %s, %w", updateResp.Reply.Detail, ErrTXTRecordCreate)
		}
		log.InfoContext(ctx, "Updated TXT record", "key", key, "fqdn", fqdn)
		return nil
	}

	resp, err := nsClient.AddRecord(apiKey, domain, rrhost, key)
	if err != nil {
		return fmt.Errorf("adding TXT record for %s: %w", fqdn, err)
	}
	if resp.Reply.Code != "300" {
		log.ErrorContext(ctx, "Error adding TXT record",
			"key", key, "fqdn", fqdn, "zone", zone, "detail", resp.Reply.Detail)
		return fmt.Errorf("error adding TXT record: %s, %w", resp.Reply.Detail, ErrTXTRecordCreate)
	}
	log.InfoContext(ctx, "Added TXT record", "key", key, "fqdn", fqdn, "zone", zone)
	return nil
}

// CleanUp deletes all TXT records matching the host from the DNS provider.
func CleanUp(ctx context.Context, log *slog.Logger, nsClient Client, apiKey, fqdn, zone string) error {
	domain := GetDomainFromZone(zone)

	listResp, err := nsClient.ListRecords(apiKey, domain)
	if err != nil {
		log.ErrorContext(ctx, "Error listing TXT records",
			"fqdn", fqdn, "zone", zone, "err", err)
		return fmt.Errorf("listing DNS records for %s: %w", domain, err)
	}
	if listResp.Reply.Code != "300" {
		return fmt.Errorf("error fetching txt record: %s, %w", listResp.Reply.Detail, ErrTXTRecordFetch)
	}

	expectedHost := RRHost(fqdn, zone)
	found := false

	for _, r := range listResp.Reply.ResourceRecord {
		if r.Host != expectedHost || r.Type != "TXT" {
			continue
		}
		found = true
		log.InfoContext(ctx, "Deleting TXT record",
			"recordID", r.ResourceID, "fqdn", fqdn, "value", r.Value)
		deleteResp, deleteErr := nsClient.DeleteRecord(apiKey, domain, r.ResourceID)
		if deleteErr != nil {
			return fmt.Errorf("deleting TXT record %s for %s: %w", r.ResourceID, fqdn, deleteErr)
		}
		if deleteResp.Reply.Code != "300" {
			log.ErrorContext(ctx, "Error deleting TXT record",
				"recordID", r.ResourceID, "fqdn", fqdn, "detail", deleteResp.Reply.Detail)
			return fmt.Errorf("error deleting TXT record: %s, %w", deleteResp.Reply.Detail, ErrTXTRecordDelete)
		}
	}

	if !found {
		log.InfoContext(ctx, "TXT record not found, skipping cleanup", "fqdn", fqdn)
	}
	return nil
}
