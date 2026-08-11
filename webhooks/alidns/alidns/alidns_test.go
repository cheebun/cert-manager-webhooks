// SPDX-License-Identifier: MIT

package alidns

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"testing"

	sdkalidns "github.com/alibabacloud-go/alidns-20150109/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/assert"
)

func newTestAliDNS(t *testing.T) *AliDNS {
	accessKeyId := os.Getenv("WEBHOOK_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("WEBHOOK_ACCESS_KEY_SECRET")
	if len(accessKeyId) == 0 || len(accessKeySecret) == 0 {
		t.Skipf("no accessKeyId or accessKeySecret set")
	}

	cli, err := sdkalidns.NewClient(&openapi.Config{
		Endpoint:        tea.String("dns.aliyuncs.com"),
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
	})
	if assert.NoError(t, err) {
		return &AliDNS{cli: cli}
	}

	return nil
}

func randomDNSRecord(t *testing.T) (string, string) {
	domainName := os.Getenv("WEBHOOK_DOMAIN_NAME")
	if len(domainName) == 0 {
		t.Skipf("no domain name set")
	}

	subDomain := make([]byte, 4)
	if _, err := io.ReadFull(rand.Reader, subDomain); assert.NoError(t, err) {
		return hex.EncodeToString(subDomain) + "." + domainName + ".", domainName + "."
	}
	return "", ""
}

func TestAliDNS_AddRecord(t *testing.T) {
	if c := newTestAliDNS(t); c != nil {
		t.Run("not exists", func(t *testing.T) {
			if fqdn, zone := randomDNSRecord(t); len(fqdn) != 0 {
				err := c.AddRecord(fqdn, zone, "foobar")
				if assert.NoError(t, err) {
					_ = c.DeleteRecord(fqdn, zone, "foobar")
				}
			}
		})

		t.Run("same value", func(t *testing.T) {
			if fqdn, zone := randomDNSRecord(t); len(fqdn) != 0 {
				err := c.AddRecord(fqdn, zone, "foobar")
				if assert.NoError(t, err) {
					err = c.AddRecord(fqdn, zone, "foobar")
					if assert.NoError(t, err) {
						_ = c.DeleteRecord(fqdn, zone, "foobar")
					}
				}
			}
		})

		t.Run("updated", func(t *testing.T) {
			if fqdn, zone := randomDNSRecord(t); len(fqdn) != 0 {
				err := c.AddRecord(fqdn, zone, "foobar")
				if assert.NoError(t, err) {
					err = c.AddRecord(fqdn, zone, "updated")
					if assert.NoError(t, err) {
						_ = c.DeleteRecord(fqdn, zone, "updated")
					}
				}
			}
		})
	}
}

func TestAliDNS_DeleteRecord(t *testing.T) {
	if c := newTestAliDNS(t); c != nil {
		t.Run("not exists", func(t *testing.T) {
			if fqdn, zone := randomDNSRecord(t); len(fqdn) != 0 {
				err := c.DeleteRecord(fqdn, zone, "nonexistent")
				assert.NoError(t, err)
			}
		})

		t.Run("exists", func(t *testing.T) {
			if fqdn, zone := randomDNSRecord(t); len(fqdn) != 0 {
				err := c.AddRecord(fqdn, zone, "foobar")
				if assert.NoError(t, err) {
					_ = c.DeleteRecord(fqdn, zone, "foobar")
				}
			}
		})
	}
}
