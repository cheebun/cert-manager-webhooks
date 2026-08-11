// SPDX-License-Identifier: MIT

package alidns

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	sdkalidns "github.com/alibabacloud-go/alidns-20150109/v4/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	acme "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// NewAliDNSClient creates an AliDNS client from the given config and k8s client.
func NewAliDNSClient(kube *kubernetes.Clientset, cfg *Config, challenge *acme.ChallengeRequest) (*AliDNS, error) {
	accessKeyId, err := loadSecretData(kube, cfg.AccessKeyIdRef, challenge.ResourceNamespace)
	if err != nil {
		return nil, err
	}
	if trimmed := bytes.TrimSpace(accessKeyId); len(trimmed) != len(accessKeyId) {
		slog.WarnContext(context.TODO(), "Blank characters in accessKeyId have been trimmed")
		accessKeyId = trimmed
	}
	if !validSecretData(accessKeyId) {
		return nil, errors.New("The accessKeyId contains invalid characters")
	}

	accessKeySecret, err := loadSecretData(kube, cfg.AccessKeySecretRef, challenge.ResourceNamespace)
	if err != nil {
		return nil, err
	}
	if trimmed := bytes.TrimSpace(accessKeySecret); len(trimmed) != len(accessKeySecret) {
		slog.WarnContext(context.TODO(), "Blank characters in accessKeySecret have been trimmed")
		accessKeySecret = trimmed
	}
	if !validSecretData(accessKeySecret) {
		return nil, errors.New("The accessKeySecret contains invalid characters")
	}

	endpoint := "dns.aliyuncs.com"
	if len(cfg.Region) != 0 {
		endpoint = "alidns." + cfg.Region + ".aliyuncs.com"
	}

	const timeoutMs = 30000 // 30 seconds in milliseconds
	cli, err := sdkalidns.NewClient(&openapi.Config{
		Endpoint:        &endpoint,
		AccessKeyId:     tea.String(string(accessKeyId)),
		AccessKeySecret: tea.String(string(accessKeySecret)),
		ReadTimeout:     tea.Int(timeoutMs),
		ConnectTimeout:  tea.Int(timeoutMs),
	})
	if err != nil {
		return nil, err
	}

	return &AliDNS{cli: cli}, nil
}

// loadSecretData loads the specified secret from Kubernetes.
func loadSecretData(kube *kubernetes.Clientset, selector cmmeta.SecretKeySelector, ns string) ([]byte, error) {
	secret, err := kube.CoreV1().Secrets(ns).Get(context.TODO(), selector.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading secret %q", ns+"/"+selector.Name)
	}

	if data, ok := secret.Data[selector.Key]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("couldn't find key %q in secret %q", selector.Key, ns+"/"+selector.Name)
}

// validSecretData reports whether data contains no control characters.
func validSecretData(data []byte) bool {
	for _, b := range data {
		if b <= ' ' || b == 0x7f {
			return false
		}
	}
	return true
}
