// SPDX-License-Identifier: MIT

package dnspod

import (
	"context"
	"fmt"
	"strings"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Solver) loadSecretData(namespace string, ref cmmeta.SecretKeySelector) (string, error) {
	secret, err := s.client.CoreV1().Secrets(namespace).Get(context.TODO(), ref.Name, metav1.GetOptions{})
	if err != nil {
		s.logErr(
			err, "failed to get secret",
			"namespace", namespace,
			"name", ref.Name,
		)
		return "", err
	}
	data, ok := secret.Data[ref.Key]
	if !ok {
		s.log.Error(
			"no data found in secret",
			"key", ref.Key,
			"namespace", namespace,
			"name", ref.Name,
		)
		return "", fmt.Errorf("no data found for %q in secret '%s/%s'", ref.Key, namespace, ref.Name)
	}
	return strings.TrimSpace(string(data)), nil
}

func (s *Solver) getConfigAndClient(ch *v1alpha1.ChallengeRequest) (*Config, dnspodAPI, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		s.logErr(err, "failed to load config from challenge request")
		return nil, nil, err
	}

	secretId, err := s.loadSecretData(ch.ResourceNamespace, cfg.SecretIdRef)
	if err != nil {
		s.logErr(err, "failed to load secret id from secret")
		return nil, nil, err
	}

	secretKey, err := s.loadSecretData(ch.ResourceNamespace, cfg.SecretKeyRef)
	if err != nil {
		s.logErr(err, "failed to load secret key from secret")
		return nil, nil, err
	}

	// Use a NUL byte as separator to avoid key collisions between
	// credentials whose concatenation would otherwise be identical.
	cacheKey := secretId + "\x00" + secretKey
	if v, ok := s.dnsCache.Load(cacheKey); ok {
		return cfg, v.(dnspodAPI), nil
	}

	credential := common.NewCredential(secretId, secretKey)
	dnspodClient, err := dnspod.NewClient(credential, "", profile.NewClientProfile())
	if err != nil {
		s.logErr(err, "failed to create dnspod client")
		return nil, nil, fmt.Errorf("creating dnspod client: %w", err)
	}

	s.dnsCache.Store(cacheKey, dnspodClient)
	return cfg, dnspodClient, nil
}
