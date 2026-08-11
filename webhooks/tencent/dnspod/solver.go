// SPDX-License-Identifier: MIT

package dnspod

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/issuer/acme/dns/util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Solver implements the logic needed to 'present' an ACME challenge TXT record
// for dnspod.
type Solver struct {
	client    *kubernetes.Clientset
	dnsCache  sync.Map // key: "secretId\x00secretKey" → dnspodAPI
	log       *slog.Logger
	logLevel  *slog.LevelVar
}

func NewSolver() *Solver {
	log, logLevel := newLogger()
	return &Solver{
		log:      log,
		logLevel: logLevel,
	}
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
func (s *Solver) Name() string {
	return "dnspod"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
func (s *Solver) Present(ch *v1alpha1.ChallengeRequest) error {
	s.log.Info("present ACME DNS01 challenge", "fqdn", ch.ResolvedFQDN, "zone", ch.ResolvedZone)
	cfg, dnspodClient, err := s.getConfigAndClient(ch)
	if err != nil {
		return err
	}

	return s.createTxtRecord(
		dnspodClient,
		util.UnFqdn(ch.ResolvedZone),
		ch.ResolvedFQDN,
		ch.Key,
		cfg.RecordLine,
		cfg.TTL,
	)
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
func (s *Solver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	s.log.Info("clean up relevant TXT record of ACME DNS01 challenge", "fqdn", ch.ResolvedFQDN, "zone", ch.ResolvedZone)
	_, dnspodClient, err := s.getConfigAndClient(ch)
	if err != nil {
		return err
	}

	return s.ensureTxtRecordsDeleted(dnspodClient, util.UnFqdn(ch.ResolvedZone), ch.ResolvedFQDN, ch.Key)
}

// Initialize will be called when the webhook first starts.
func (s *Solver) Initialize(kubeClientConfig *rest.Config, _ <-chan struct{}) error {
	client, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		s.logErr(err, "Failed to create kubernetes client")
		return fmt.Errorf("creating kubernetes client: %w", err)
	}
	s.client = client
	return nil
}
