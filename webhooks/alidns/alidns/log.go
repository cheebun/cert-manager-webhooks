// SPDX-License-Identifier: MIT

package alidns

import (
	"log/slog"
	"os"
)

func newLogger() (*slog.Logger, *slog.LevelVar) {
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	return log, logLevel
}
