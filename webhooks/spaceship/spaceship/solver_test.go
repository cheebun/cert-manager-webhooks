// SPDX-License-Identifier: MIT

package spaceship

import (
	"errors"
	"testing"

	cmacme "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	restclient "k8s.io/client-go/rest"
)

type solverClientStub struct {
	addArgs     []string
	removeArgs  []string
	listRecords []DNSTXTRecord
	addErr      error
	removeErr   error
	listErr     error
	addTTL      int
}

func (s *solverClientStub) ListTXTRecords(domain, name string) ([]DNSTXTRecord, error) {
	return s.listRecords, s.listErr
}

func (s *solverClientStub) AddTXTRecord(domain, name, value string, ttl int) error {
	s.addArgs = []string{domain, name, value}
	s.addTTL = ttl
	return s.addErr
}

func (s *solverClientStub) RemoveTXTRecord(domain, name string) error {
	s.removeArgs = []string{domain, name}
	return s.removeErr
}

func newTestSolver(stub SolverClient) *Solver {
	s := NewSolver()
	s.client = stub
	return s
}

func TestSolverPresentNoRecord(t *testing.T) {
	stub := &solverClientStub{}
	solver := newTestSolver(stub)

	ch := &cmacme.ChallengeRequest{
		ResolvedFQDN: "_acme-challenge.example.com.",
		ResolvedZone: "example.com.",
		Key:          "token",
	}

	if err := solver.Present(ch); err != nil {
		t.Fatalf("Present error: %v", err)
	}
	if len(stub.addArgs) != 3 {
		t.Fatalf("addArgs len = %d", len(stub.addArgs))
	}
	if stub.addArgs[0] != "example.com" || stub.addArgs[1] != "_acme-challenge" || stub.addArgs[2] != "token" {
		t.Fatalf("addArgs = %#v", stub.addArgs)
	}
	if stub.addTTL != 60 {
		t.Fatalf("addTTL = %d", stub.addTTL)
	}
}

func TestSolverPresentRecordAlreadyExists(t *testing.T) {
	stub := &solverClientStub{
		listRecords: []DNSTXTRecord{
			{Type: "TXT", Name: "_acme-challenge", Value: "token"},
		},
	}
	solver := newTestSolver(stub)

	ch := &cmacme.ChallengeRequest{
		ResolvedFQDN: "_acme-challenge.example.com.",
		ResolvedZone: "example.com.",
		Key:          "token",
	}

	if err := solver.Present(ch); err != nil {
		t.Fatalf("Present error: %v", err)
	}
	if len(stub.addArgs) != 0 {
		t.Fatalf("AddTXTRecord should not be called when record already exists, addArgs = %#v", stub.addArgs)
	}
}

func TestSolverPresentRecordDifferentValue(t *testing.T) {
	stub := &solverClientStub{
		listRecords: []DNSTXTRecord{
			{Type: "TXT", Name: "_acme-challenge", Value: "old-token"},
		},
	}
	solver := newTestSolver(stub)

	ch := &cmacme.ChallengeRequest{
		ResolvedFQDN: "_acme-challenge.example.com.",
		ResolvedZone: "example.com.",
		Key:          "new-token",
	}

	if err := solver.Present(ch); err != nil {
		t.Fatalf("Present error: %v", err)
	}
	if len(stub.addArgs) != 3 || stub.addArgs[2] != "new-token" {
		t.Fatalf("addArgs = %#v", stub.addArgs)
	}
}

func TestSolverCleanUp(t *testing.T) {
	stub := &solverClientStub{}
	solver := newTestSolver(stub)

	ch := &cmacme.ChallengeRequest{
		ResolvedFQDN: "_acme-challenge.example.com.",
		ResolvedZone: "example.com.",
		Key:          "token",
	}

	if err := solver.CleanUp(ch); err != nil {
		t.Fatalf("CleanUp error: %v", err)
	}
	if len(stub.removeArgs) != 2 {
		t.Fatalf("removeArgs len = %d", len(stub.removeArgs))
	}
	if stub.removeArgs[0] != "example.com" || stub.removeArgs[1] != "_acme-challenge" {
		t.Fatalf("removeArgs = %#v", stub.removeArgs)
	}
}

func TestSolverInitializeBuildsKubeClient(t *testing.T) {
	solver := NewSolver()
	// An empty rest.Config will fail TLS setup — we just verify Initialize
	// attempts to build a kube client (returns an error rather than silently
	// doing nothing as before).
	err := solver.Initialize(&restclient.Config{Host: "https://127.0.0.1:0"}, make(chan struct{}))
	// kubernetes.NewForConfig does not dial; it only validates the config.
	// With a valid (if unreachable) host it should succeed.
	if err != nil {
		t.Fatalf("Initialize error: %v", err)
	}
	if solver.kube == nil {
		t.Fatal("expected kube clientset to be set after Initialize")
	}
}

func TestSolverPresentListError(t *testing.T) {
	wantErr := errors.New("list error")
	stub := &solverClientStub{listErr: wantErr}
	solver := newTestSolver(stub)

	ch := &cmacme.ChallengeRequest{
		ResolvedFQDN: "_acme-challenge.example.com.",
		ResolvedZone: "example.com.",
		Key:          "token",
	}

	err := solver.Present(ch)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Present error = %v, want %v", err, wantErr)
	}
	if len(stub.addArgs) != 0 {
		t.Fatalf("AddTXTRecord should not be called when ListTXTRecords returns error, addArgs = %#v", stub.addArgs)
	}
}

func TestSolverCleanUpError(t *testing.T) {
	wantErr := errors.New("remove error")
	stub := &solverClientStub{removeErr: wantErr}
	solver := newTestSolver(stub)

	ch := &cmacme.ChallengeRequest{
		ResolvedFQDN: "_acme-challenge.example.com.",
		ResolvedZone: "example.com.",
		Key:          "token",
	}

	err := solver.CleanUp(ch)
	if !errors.Is(err, wantErr) {
		t.Fatalf("CleanUp error = %v, want %v", err, wantErr)
	}
}
