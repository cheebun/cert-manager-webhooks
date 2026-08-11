//go:build integration

package main

import (
	"os"
	"testing"

	acmetest "github.com/cert-manager/cert-manager/test/acme"
	"github.com/cheebun/cert-manager-webhooks/webhooks/namesilo/namesilo"
)

var (
	zone = os.Getenv("TEST_ZONE_NAME")
)

func TestRunsSuite(t *testing.T) { //nolint:paralleltest,wsl
	// The manifest path should contain a file named config.json that is a
	// snippet of valid configuration that should be included on the
	// ChallengeRequest passed as part of the test cases.

	fixture := acmetest.NewFixture(namesilo.NewSolver(),
		acmetest.SetResolvedZone(zone),
		acmetest.SetAllowAmbientCredentials(false),
		acmetest.SetManifestPath("testdata/namesilo"),
		acmetest.SetDNSServer("202.46.34.75:53"),
	)
	fixture.RunBasic(t)
	fixture.RunExtended(t)
}
