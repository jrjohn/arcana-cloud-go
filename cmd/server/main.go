package main

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/di"
)

func main() {
	app := fx.New(
		// Load all application modules via DI
		di.AppModule,

		// Print startup banner
		fx.Invoke(di.PrintBanner),

		// Configure fx logger to use zap
		fx.WithLogger(func(logger *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: logger}
		}),
	)

	app.Run()
}
