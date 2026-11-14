package server

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"

	"github.com/north-shore-software/kdnk/internal/chat"
)

const (
	guestIDLength       = 24
	guestPasswordLength = 40
	guestTTL            = 30 * 24 * time.Hour
	guestPruneSchedule  = "0 3 * * *"
)

type guestCredentials struct {
	GuestID  string `json:"guestId"`
	Password string `json:"password"`
}

type upgradeRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleGuestCreate(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		col, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return e.InternalServerError("users collection unavailable", err)
		}

		creds := guestCredentials{
			GuestID:  security.RandomString(guestIDLength),
			Password: security.RandomString(guestPasswordLength),
		}

		rec := core.NewRecord(col)
		rec.Set("guestId", creds.GuestID)
		rec.Set("isGuest", true)
		rec.SetPassword(creds.Password)

		if err := app.Save(rec); err != nil {
			return e.InternalServerError("guest creation failed", err)
		}

		return apis.RecordAuthResponse(e, rec, "guest", creds)
	}
}

func handleGuestUpgrade(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.UnauthorizedError("authentication required", nil)
		}
		if !e.Auth.GetBool("isGuest") {
			return e.BadRequestError("account is not a guest", nil)
		}

		req := upgradeRequest{}
		if err := e.BindBody(&req); err != nil {
			return e.BadRequestError("invalid upgrade request body", err)
		}
		if req.Email == "" || req.Password == "" {
			return e.BadRequestError("email and password required", nil)
		}

		e.Auth.SetEmail(req.Email)
		e.Auth.SetPassword(req.Password)
		e.Auth.Set("isGuest", false)
		e.Auth.Set("guestId", "")

		if err := app.Save(e.Auth); err != nil {
			return e.BadRequestError("upgrade failed", err)
		}

		return apis.RecordAuthResponse(e, e.Auth, "password", nil)
	}
}

func lastActivity(app core.App, ownerID string) (time.Time, error) {
	recs, err := app.FindRecordsByFilter(
		chat.CollectionMessage,
		"conversation.owner = {:owner}",
		"-created", 1, 0,
		dbx.Params{"owner": ownerID},
	)
	if err != nil {
		return time.Time{}, err
	}
	if len(recs) == 0 {
		return time.Time{}, nil
	}
	return recs[0].GetDateTime("created").Time(), nil
}

func pruneGuests(app core.App, ttl time.Duration) (int, error) {
	cutoff := time.Now().Add(-ttl)

	stale, err := app.FindRecordsByFilter(
		"users",
		"isGuest = true && created < {:cutoff}",
		"created", 0, 0,
		dbx.Params{"cutoff": cutoff.UTC().Format("2006-01-02 15:04:05.000Z")},
	)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, rec := range stale {
		seen, err := lastActivity(app, rec.Id)
		if err != nil {
			return deleted, err
		}
		if !seen.IsZero() && seen.After(cutoff) {
			continue
		}
		if err := app.Delete(rec); err != nil {
			return deleted, err
		}
		deleted++
	}

	return deleted, nil
}

func registerGuestPrune(app core.App, ttl time.Duration) {
	app.Cron().MustAdd("prune_guests", guestPruneSchedule, func() {
		deleted, err := pruneGuests(app, ttl)
		if err != nil {
			app.Logger().Error("guest prune failed", "error", err)
			return
		}
		if deleted > 0 {
			app.Logger().Info("pruned stale guests", "count", deleted)
		}
	})
}
