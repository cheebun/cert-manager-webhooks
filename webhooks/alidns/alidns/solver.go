// SPDX-License-Identifier: MIT

package alidns

import (
	"context"
	"log/slog"

	acme "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Solver implements the cert-manager DNS01 webhook solver for Alibaba Cloud DNS.
type Solver struct {
	kube     *kubernetes.Clientset
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

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
func (s *Solver) Name() string {
	return "alidns"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
func (s *Solver) Present(challenge *acme.ChallengeRequest) error {
	s.log.InfoContext(context.TODO(), "Presenting TXT record", "fqdn", challenge.ResolvedFQDN)

	cfg, err := LoadConfig(challenge.Config)
	if err != nil {
		return err
	}

	dns, err := NewAliDNSClient(s.kube, cfg, challenge)
	if err != nil {
		s.log.ErrorContext(context.TODO(), "Failed to load alidns", "error", err)
		return err
	}

	if err = dns.AddRecord(challenge.ResolvedFQDN, challenge.ResolvedZone, challenge.Key); err != nil {
		s.log.ErrorContext(context.TODO(), "Failed to add TXT record",
			"fqdn", challenge.ResolvedFQDN, "error", err)
	}
	return err
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
func (s *Solver) CleanUp(challenge *acme.ChallengeRequest) error {
	s.log.InfoContext(context.TODO(), "Cleaning up TXT record", "fqdn", challenge.ResolvedFQDN)

	cfg, err := LoadConfig(challenge.Config)
	if err != nil {
		return err
	}

	dns, err := NewAliDNSClient(s.kube, cfg, challenge)
	if err != nil {
		return err
	}

	if err = dns.DeleteRecord(challenge.ResolvedFQDN, challenge.ResolvedZone, challenge.Key); err != nil {
		s.log.ErrorContext(context.TODO(), "Failed to delete TXT record",
			"fqdn", challenge.ResolvedFQDN, "error", err)
	}
	return err
}

// Initialize will be called when the webhook first starts.
func (s *Solver) Initialize(kubeClientConfig *rest.Config, _ <-chan struct{}) (err error) {
	s.kube, err = kubernetes.NewForConfig(kubeClientConfig)
	return
}
