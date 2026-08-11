// SPDX-License-Identifier: MIT

package namesilo

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HTTPClient is the production implementation of Client that delegates to Call.
type HTTPClient struct{}

func (h *HTTPClient) ListRecords(apiKey, domain string) (DNSRecordListResponse, error) {
	return Call[DNSRecordListResponse](apiKey, "dnsListRecords", map[string]string{
		"domain": domain,
	})
}

func (h *HTTPClient) AddRecord(apiKey, domain, rrhost, rrvalue string) (Response, error) {
	return Call[Response](apiKey, "dnsAddRecord", map[string]string{
		"domain":  domain,
		"rrtype":  "TXT",
		"rrhost":  rrhost,
		"rrvalue": rrvalue,
	})
}

func (h *HTTPClient) UpdateRecord(apiKey, domain, rrid, rrhost, rrvalue string) (Response, error) {
	return Call[Response](apiKey, "dnsUpdateRecord", map[string]string{
		"domain":  domain,
		"rrid":    rrid,
		"rrtype":  "TXT",
		"rrhost":  rrhost,
		"rrvalue": rrvalue,
	})
}

func (h *HTTPClient) DeleteRecord(apiKey, domain, rrid string) (Response, error) {
	return Call[Response](apiKey, "dnsDeleteRecord", map[string]string{
		"domain": domain,
		"rrid":   rrid,
	})
}

// LoadAPIKey loads the Namesilo API key from a Kubernetes Secret.
func LoadAPIKey(ctx context.Context, kube *kubernetes.Clientset, cfg Config, namespace string) (string, error) {
	secret, err := kube.CoreV1().Secrets(namespace).Get(ctx, cfg.APIKey.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting api key: %w", err)
	}

	keyBytes, ok := secret.Data[cfg.APIKey.Key]
	if !ok {
		return "", fmt.Errorf("secret key not found in %s/%s (key: %s): %w",
			namespace, cfg.APIKey.Name, cfg.APIKey.Key, ErrAPIKeyDecode)
	}
	return strings.TrimSpace(string(keyBytes)), nil
}

