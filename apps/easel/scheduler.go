package main

import (
	"context"
	"time"
)

func (a *App) todayInTZ() string {
	return time.Now().In(a.TZ).Format("2006-01-02")
}

func (a *App) pastNDates(n int) []string {
	today := time.Now().In(a.TZ).Truncate(24 * time.Hour)
	out := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, today.AddDate(0, 0, -i).Format("2006-01-02"))
	}
	return out
}

func parseDate(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}

func (a *App) ensureDay(ctx context.Context, date string) error {
	if d, err := getDaily(a.DB, date); err != nil {
		return err
	} else if d != nil {
		return nil
	}
	raw, err := pickUnique(ctx, a.HTTP, a.DB, a.Classifications, a.ExcludeTerms, a.MaxDedupRetries)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	daily := rawToDaily(raw, date, now)
	if daily == nil {
		return errMissingImage
	}
	if _, err := insertDaily(a.DB, daily); err != nil {
		return err
	}
	a.Log.Info("stored artwork", "artwork_id", daily.ArtworkID, "date", date, "image_id", daily.ImageID)
	return nil
}

var errMissingImage = &simpleError{"missing image_id on selected artwork"}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

func (a *App) runScheduler(ctx context.Context) {
	if err := a.ensureDay(ctx, a.todayInTZ()); err != nil {
		a.Log.Warn("startup ensure_day failed", "err", err)
	}
	if a.BackfillDays > 0 {
		dates := a.pastNDates(a.BackfillDays)
		missing, err := missingDates(a.DB, dates)
		if err != nil {
			a.Log.Error("backfill missing_dates failed", "err", err)
		} else {
			a.Log.Info("backfill", "missing", len(missing), "window_days", a.BackfillDays)
			for _, d := range missing {
				if err := a.ensureDay(ctx, d); err != nil {
					a.Log.Warn("backfill day failed", "date", d, "err", err)
				}
			}
		}
	}
	for {
		dur := a.durationUntilNextMidnight()
		a.Log.Info("scheduler sleeping", "seconds", dur.Seconds(), "tz", a.TZName)
		select {
		case <-ctx.Done():
			return
		case <-time.After(dur):
		}
		if err := a.ensureDay(ctx, a.todayInTZ()); err != nil {
			a.Log.Warn("scheduled ensure_day failed", "err", err)
		}
	}
}

func (a *App) durationUntilNextMidnight() time.Duration {
	now := time.Now().In(a.TZ)
	nextDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 1, 0, a.TZ).AddDate(0, 0, 1)
	delta := nextDay.Sub(now)
	if delta < time.Second {
		return time.Hour
	}
	return delta
}
