// SPDX-License-Identifier: MIT

package namesilo

import (
	"context"
	"fmt"
	"log/slog"

	v1alpha1 "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Solver implements the cert-manager DNS01 webhook solver for Namesilo.
type Solver struct {
	kube     *kubernetes.Clientset
	nsClient Client
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
	return "namesilo"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
func (s *Solver) Present(ch *v1alpha1.ChallengeRequest) error {
	ctx := context.Background() // cert-manager v1 ChallengeRequest does not carry a context

	cfg, err := LoadConfig(ch.Config)
	if err != nil {
		return err
	}

	apiKey, err := LoadAPIKey(ctx, s.kube, cfg, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	s.log.InfoContext(ctx, "Presenting TXT record",
		"key", ch.Key,
		"fqdn", ch.ResolvedFQDN,
		"zone", ch.ResolvedZone,
	)

	return Present(ctx, s.log, s.nsClient, apiKey, ch.ResolvedFQDN, ch.ResolvedZone, ch.Key)
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
func (s *Solver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	ctx := context.Background() // cert-manager v1 ChallengeRequest does not carry a context

	cfg, err := LoadConfig(ch.Config)
	if err != nil {
		return err
	}

	apiKey, err := LoadAPIKey(ctx, s.kube, cfg, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	return CleanUp(ctx, s.log, s.nsClient, apiKey, ch.ResolvedFQDN, ch.ResolvedZone)
}

// Initialize will be called when the webhook first starts.
func (s *Solver) Initialize(kubeClientConfig *rest.Config, _ <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return fmt.Errorf("error getting client config: %w", err)
	}
	s.kube = cl
	s.nsClient = &HTTPClient{}
	return nil
}
