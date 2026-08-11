package main

import (
	"os"
	_ "time/tzdata"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/cheebun/cert-manager-webhooks/webhooks/namesilo/namesilo"
	_ "golang.org/x/crypto/x509roots/fallback"
)

var (
	GroupName = os.Getenv("GROUP_NAME")
	LogLevel  = os.Getenv("LOG_LEVEL")
)

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}
	solver := namesilo.NewSolver()
	if LogLevel != "" {
		if err := solver.SetLogLevel(LogLevel); err != nil {
			panic("invalid LOG_LEVEL: " + err.Error())
		}
	}
	cmd.RunWebhookServer(GroupName, solver)
}
