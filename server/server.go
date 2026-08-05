package server

import (
	"io/fs"
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/north-shore-software/agent-smith/llm"
	_ "github.com/north-shore-software/agent-smith/migrations"
	"github.com/north-shore-software/agent-smith/plugin"
)

type Config struct {
	Plugin          plugin.Plugin
	Web             fs.FS
	HideStartBanner bool
}

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

func EnsureReady() {
	if !llm.Ready() {
		log.Fatal("server.EnsureReady: no LLM provider registered; call llm.SetProviderFactory before EnsureReady")
	}
}
