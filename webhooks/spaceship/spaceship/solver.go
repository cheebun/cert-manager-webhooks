// SPDX-License-Identifier: MIT

package spaceship

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	cmacme "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

const SolverName = "spaceship"

// Config holds the per-Issuer configuration decoded from the ChallengeRequest.
type Config struct {
	APIKeyRef    cmmeta.SecretKeySelector `json:"apiKeyRef"`
	APISecretRef cmmeta.SecretKeySelector `json:"apiSecretRef"`
}

// SolverClient is the interface for Spaceship DNS operations.
// Defined on the consumer side to allow test stubs.
type SolverClient interface {
	ListTXTRecords(domain, name string) ([]DNSTXTRecord, error)
	AddTXTRecord(domain, name, value string, ttl int) error
	RemoveTXTRecord(domain, name string) error
}

// Solver implements the cert-manager webhook solver interface for Spaceship DNS.
type Solver struct {
	kube     *kubernetes.Clientset
	client   SolverClient // non-nil only in tests
	log      *slog.Logger
	logLevel *slog.LevelVar
}

// NewSolver returns a new Solver ready for use with cmd.RunWebhookServer.
func NewSolver() *Solver {
	log, logLevel := newLogger()
	return &Solver{
		log:      log,
		logLevel: logLevel,
	}
}

// SetLogLevel updates the log level at runtime.
func (s *Solver) SetLogLevel(level string) error {
	if err := s.logLevel.UnmarshalText([]byte(level)); err != nil {
		s.log.Error("failed to parse log level, valid values are: debug, info, warn, error",
			"level", level, "error", err)
		return err
	}
	return nil
}

func (s *Solver) Name() string {
	return SolverName
}

// Present creates (or confirms) the ACME DNS-01 TXT record.
func (s *Solver) Present(ch *cmacme.ChallengeRequest) error {
	s.log.Info("present ACME DNS01 challenge", "fqdn", ch.ResolvedFQDN, "zone", ch.ResolvedZone)

	client, err := s.loadClient(ch)
	if err != nil {
		return err
	}

	domain := strings.TrimSuffix(ch.ResolvedZone, ".")
	name := recordName(domain, ch.ResolvedFQDN)

	records, err := client.ListTXTRecords(domain, name)
	if err != nil {
		s.log.Error("failed to list TXT records", "domain", domain, "name", name, "error", err)
		return err
	}
	for _, r := range records {
		if r.Value == ch.Key {
			s.log.Info("TXT record already exists, skipping", "name", name, "domain", domain)
			return nil
		}
	}

	s.log.Info("adding TXT record", "name", name, "domain", domain)
	if err := client.AddTXTRecord(domain, name, ch.Key, 60); err != nil {
		s.log.Error("failed to add TXT record", "domain", domain, "name", name, "error", err)
		return err
	}
	return nil
}

// CleanUp removes all TXT records for the ACME challenge host.
func (s *Solver) CleanUp(ch *cmacme.ChallengeRequest) error {
	s.log.Info("clean up relevant TXT record of ACME DNS01 challenge", "fqdn", ch.ResolvedFQDN, "zone", ch.ResolvedZone)

	client, err := s.loadClient(ch)
	if err != nil {
		return err
	}

	domain := strings.TrimSuffix(ch.ResolvedZone, ".")
	name := recordName(domain, ch.ResolvedFQDN)

	s.log.Debug("removing TXT record", "name", name, "domain", domain)
	if err := client.RemoveTXTRecord(domain, name); err != nil {
		s.log.Error("failed to remove TXT record", "domain", domain, "name", name, "error", err)
		return err
	}
	return nil
}

// Initialize stores the Kubernetes client for later Secret lookups.
func (s *Solver) Initialize(kubeClientConfig *restclient.Config, _ <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		s.log.Error("failed to create kubernetes client", "error", err)
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	s.kube = cl
	return nil
}

// loadClient returns the SolverClient for this challenge.
// In tests, s.client is pre-injected and returned directly.
// In production, credentials are read from Kubernetes Secrets.
func (s *Solver) loadClient(ch *cmacme.ChallengeRequest) (SolverClient, error) {
	if s.client != nil {
		return s.client, nil
	}

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, err
	}

	apiKey, err := s.loadSecretData(ch.ResourceNamespace, cfg.APIKeyRef)
	if err != nil {
		return nil, err
	}

	apiSecret, err := s.loadSecretData(ch.ResourceNamespace, cfg.APISecretRef)
	if err != nil {
		return nil, err
	}

	return NewClient(apiKey, apiSecret, nil), nil
}

// loadSecretData retrieves a single key from a Kubernetes Secret.
func (s *Solver) loadSecretData(ns string, ref cmmeta.SecretKeySelector) (string, error) {
	secret, err := s.kube.CoreV1().Secrets(ns).Get(context.TODO(), ref.Name, metav1.GetOptions{})
	if err != nil {
		s.log.Error("failed to get secret", "namespace", ns, "name", ref.Name, "error", err)
		return "", fmt.Errorf("failed to get secret %s/%s: %w", ns, ref.Name, err)
	}
	data, ok := secret.Data[ref.Key]
	if !ok {
		s.log.Error("key not found in secret", "key", ref.Key, "namespace", ns, "name", ref.Name)
		return "", fmt.Errorf("key %q not found in secret %s/%s", ref.Key, ns, ref.Name)
	}
	return strings.TrimSpace(string(data)), nil
}

// loadConfig decodes the JSON configuration from the ChallengeRequest.
func loadConfig(cfgJSON *extapi.JSON) (*Config, error) {
	cfg := &Config{}
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, cfg); err != nil {
		return nil, fmt.Errorf("error decoding solver config: %w", err)
	}
	return cfg, nil
}

func recordName(zone, fqdn string) string {
	zone = strings.TrimSuffix(zone, ".")
	fqdn = strings.TrimSuffix(fqdn, ".")

	if zone == "" {
		return fqdn
	}

	suffix := "." + zone
	return strings.TrimSuffix(fqdn, suffix)
}
