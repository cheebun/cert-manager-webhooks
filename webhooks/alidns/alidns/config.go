// SPDX-License-Identifier: MIT

package alidns

import (
	"encoding/json"
	"fmt"

	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/pkg/errors"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Config is a structure that is used to decode into when solving a DNS01
// challenge.
type Config struct {
	// Region can be used to select an access point close to the Webhook cluster node.
	Region string `json:"region"`

	// AccessKeyIdRef is a credential for accessing Aliyun OpenAPI.
	AccessKeyIdRef cmmeta.SecretKeySelector `json:"accessKeyIdRef"`

	// AccessKeySecretRef is the access credential secret that matches AccessKeyIdRef.
	AccessKeySecretRef cmmeta.SecretKeySelector `json:"accessKeySecretRef"`

	// SecretAccessKeyRef is the access credential secret (Amazon naming style).
	SecretAccessKeyRef cmmeta.SecretKeySelector `json:"secretAccessKeyRef"`
}

// Validate checks if the config of the webhook is valid.
func (cfg *Config) Validate() error {
	if len(cfg.AccessKeyIdRef.Name) == 0 {
		return errors.New("accessKeyIdRef may not be empty")
	}

	if len(cfg.AccessKeySecretRef.Name) == 0 {
		cfg.SecretAccessKeyRef.DeepCopyInto(&cfg.AccessKeySecretRef)
	}
	if len(cfg.AccessKeySecretRef.Name) == 0 {
		return errors.New("accessKeySecretRef may not be empty")
	}

	return nil
}

// LoadConfig decodes JSON configuration into the Config struct.
func LoadConfig(cfgJSON *extapi.JSON) (*Config, error) {
	var cfg Config

	if cfgJSON == nil {
		return &cfg, nil
	}

	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return nil, fmt.Errorf("error decoding solver config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate solver config: %v", err)
	}

	return &cfg, nil
}
