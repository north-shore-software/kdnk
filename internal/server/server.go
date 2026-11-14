// Package server builds a PocketBase application that serves a plugin.Plugin as a chat API.
package server

import (
	"io/fs"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/north-shore-software/agent-smith/internal/llm"
	_ "github.com/north-shore-software/agent-smith/internal/migrations"
	"github.com/north-shore-software/agent-smith/internal/plugin"
)

// Config configures New.
type Config struct {
	// Plugin is the application to serve. It is required.
	Plugin plugin.Plugin
	// Web, if non-nil, is served for GET requests no other route matches, falling back to
	// index.html for paths it does not contain.
	Web fs.FS
	// HideStartBanner suppresses PocketBase's startup banner.
	HideStartBanner bool
}

// New returns a PocketBase app that serves cfg.Plugin. It exits the process with log.Fatal if
// cfg.Plugin is nil.
//
// When the app starts serving, it creates any of the plugin's collections that do not exist,
// enables a default set of rate limits if rate limiting is off, schedules a daily 03:00 job that
// deletes guest users created over 30 days ago with no message in the last 30 days, and registers
// these routes:
//
//   - POST /api/guest creates a guest user and returns its auth token and generated credentials.
//   - POST /api/guest/upgrade gives the authenticated guest an email and password, making it a regular user.
//   - POST /api/chat stores the request's new messages and streams the model's response as server-sent events, running the plugin's tools for up to 4 model calls.
func New(cfg Config) *pocketbase.PocketBase {
	if cfg.Plugin == nil {
		log.Fatal("server.New: Config.Plugin is required")
	}

	app := pocketbase.NewWithConfig(pocketbase.Config{HideStartBanner: cfg.HideStartBanner})

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: false,
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := ensureCollections(app, cfg.Plugin); err != nil {
			return err
		}
		if err := ensureRateLimits(app); err != nil {
			return err
		}

		registerGuestPrune(app, guestTTL)

		se.Router.POST("/api/guest", handleGuestCreate(app))
		se.Router.POST("/api/guest/upgrade", handleGuestUpgrade(app)).Bind(apis.RequireAuth("users"))
		se.Router.POST("/api/chat", handleChat(app, cfg.Plugin)).Bind(apis.RequireAuth("users"))

		if cfg.Web != nil {
			se.Router.GET("/{path...}", apis.Static(cfg.Web, true))
		}

		return se.Next()
	})

	return app
}

func ensureRateLimits(app core.App) error {
	settings := app.Settings()
	if settings.RateLimits.Enabled {
		return nil
	}

	settings.RateLimits.Enabled = true
	settings.RateLimits.Rules = []core.RateLimitRule{
		{Label: "POST /api/guest", MaxRequests: 5, Duration: 300},
		{Label: "POST /api/chat", MaxRequests: 30, Duration: 60},
		{Label: "*:auth", MaxRequests: 5, Duration: 10},
		{Label: "*:create", MaxRequests: 20, Duration: 5},
	}

	return app.Save(settings)
}

func ensureCollections(app core.App, p plugin.Plugin) error {
	for _, col := range p.Collections() {
		if _, err := app.FindCollectionByNameOrId(col.Name); err == nil {
			continue
		}
		if err := app.Save(col); err != nil {
			return err
		}
		app.Logger().Info("created plugin collection", "plugin", p.Name(), "collection", col.Name)
	}
	return nil
}

// EnsureReady exits the process with log.Fatal if no provider factory has been registered with
// llm.SetProviderFactory.
func EnsureReady() {
	if !llm.Ready() {
		log.Fatal("server.EnsureReady: no LLM provider registered; call llm.SetProviderFactory before EnsureReady")
	}
}
