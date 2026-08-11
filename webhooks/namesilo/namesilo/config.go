// SPDX-License-Identifier: MIT

package namesilo

import (
	"encoding/json"
	"fmt"

	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Config is a structure that is used to decode into when solving a DNS01
// challenge.
type Config struct {
	APIKey cmmetav1.SecretKeySelector `json:"apiKey"`
}

// LoadConfig decodes JSON configuration into the typed config struct.
func LoadConfig(cfgJSON *extapi.JSON) (Config, error) {
	cfg := Config{}
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %w", err)
	}
	if cfg.APIKey.Name == "" {
		return cfg, fmt.Errorf("apiKey.name must not be empty")
	}
	if cfg.APIKey.Key == "" {
		return cfg, fmt.Errorf("apiKey.key must not be empty")
	}
	return cfg, nil
}
