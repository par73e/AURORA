// Export reads public astronomy data from the local AURORA database. It never
// migrates, synchronizes or writes to that database, and exports no credentials.
package main

import (
	"aurora/backend/internal/astronomyevent"
	"aurora/backend/internal/dailyimage"
	"aurora/backend/internal/database"
	"aurora/backend/internal/mars"
	"aurora/backend/internal/moon"
	"aurora/backend/internal/orbit"
	"aurora/backend/internal/voyage"
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	out := flag.String("out", "../data", "snapshot directory")
	db := flag.String("database", "postgres://localhost:5432/aurora?sslmode=disable", "local database URL (prefer DATABASE_URL env)")
	flag.Parse()
	if env := os.Getenv("DATABASE_URL"); env != "" {
		*db = env
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, *db)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	must(os.MkdirAll(*out, 0755))
	write := func(name string, value any) {
		b, e := json.MarshalIndent(value, "", "  ")
		must(e)
		must(os.WriteFile(filepath.Join(*out, name+".json"), b, 0644))
	}
	overview, err := orbit.NewRepository(pool).Overview(ctx)
	must(err)
	write("overview", overview)
	mr := moon.NewRepository(pool)
	ms, err := mr.ListSpacecraft(ctx)
	must(err)
	mt, err := mr.LastSyncTime(ctx)
	must(err)
	write("moon-spacecraft", map[string]any{"spacecraft": ms, "syncedAt": mt})
	ml, err := mr.ListLandingSites(ctx)
	must(err)
	write("moon-sites", map[string]any{"landingSites": ml})
	ar := mars.NewRepository(pool)
	as, err := ar.ListSpacecraft(ctx)
	must(err)
	at, err := ar.LastSyncTime(ctx)
	must(err)
	write("mars-spacecraft", map[string]any{"spacecraft": as, "syncedAt": at})
	al, err := ar.ListLandingSites(ctx)
	must(err)
	write("mars-sites", map[string]any{"landingSites": al})
	probes, err := voyage.NewRepository(pool).ListProbes(ctx)
	must(err)
	write("probes", map[string]any{"probes": probes})
	now := time.Now().UTC()
	events, err := astronomyevent.NewRepository(pool).List(ctx, astronomyevent.ListQuery{From: now.AddDate(-1, 0, 0), To: now.AddDate(2, 0, 0)})
	must(err)
	write("events", events)
	// The image preparation script subsequently validates and downloads every file.
	wall, ok, err := dailyimage.NewRepository(pool).LatestWallCache(ctx)
	must(err)
	if ok {
		write("image-wall-remote", wall)
	}
	write("manifest", map[string]any{"exportedAt": now.Format(time.RFC3339), "mode": "hybrid-demo", "weatherMode": "live", "locationMode": "live", "spacecraft": len(overview.Spacecraft), "probes": len(probes), "events": len(events)})
	log.Println("Public snapshots exported to", *out)
}
func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
