// SPDX-License-Identifier: MIT

package main

import (
	"os"
	_ "time/tzdata"

	"github.com/cheebun/cert-manager-webhooks/webhooks/alidns/alidns"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	_ "golang.org/x/crypto/x509roots/fallback"
)

var (
	GroupName = os.Getenv("GROUP_NAME")
	LogLevel  = os.Getenv("LOG_LEVEL")
)

func main() {
	groupName := GroupName
	if groupName == "" {
		panic("GROUP_NAME must be specified")
	}
	solver := alidns.NewSolver()
	if LogLevel != "" {
		if err := solver.SetLogLevel(LogLevel); err != nil {
			panic("invalid LOG_LEVEL: " + err.Error())
		}
	}
	cmd.RunWebhookServer(groupName, solver)
}
