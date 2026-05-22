package httpapi

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"
)

//go:embed web/templates/emergency.html web/static/styles.css web/static/scripts.js
var emergencyWebFS embed.FS

// emergencyPageTemplate loads the standalone HTML file with presentation-only
// helpers. Incident data still comes from the token-checked handlers.
var emergencyPageTemplate = template.Must(template.New("emergency.html").Funcs(template.FuncMap{
	"humanTime":    humanTime,
	"relativeTime": relativeTime,
	"formatBytes":  formatBytes,
}).ParseFS(emergencyWebFS, "web/templates/emergency.html"))

// emergencyStaticHandler serves embedded CSS and JavaScript through the
// standard file server. Static assets contain no incident data or raw tokens.
func emergencyStaticHandler() http.Handler {
	staticFiles, err := fs.Sub(emergencyWebFS, "web/static")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles)))
}

// humanTime formats absolute timestamps for the emergency page template.
func humanTime(value time.Time) string {
	if value.IsZero() {
		return "Unknown"
	}
	return value.Local().Format("2 Jan 2006, 3:04 pm")
}

// relativeTime formats recent activity for quick scanning in the emergency
// page template.
func relativeTime(value time.Time) string {
	if value.IsZero() {
		return "Unknown"
	}

	now := time.Now()
	if value.After(now) {
		return "in " + humanDuration(value.Sub(now))
	}

	elapsed := now.Sub(value)
	if elapsed < time.Minute {
		return "just now"
	}
	return humanDuration(elapsed) + " ago"
}

func humanDuration(value time.Duration) string {
	if value < time.Minute {
		return "less than a minute"
	}
	minutes := int(value.Minutes())
	if minutes < 60 {
		return plural(minutes, "minute")
	}
	hours := int(value.Hours())
	if hours < 24 {
		return plural(hours, "hour")
	}
	days := hours / 24
	if days < 7 {
		return plural(days, "day")
	}
	weeks := days / 7
	if weeks < 5 {
		return plural(weeks, "week")
	}
	months := days / 30
	if months < 12 {
		return plural(months, "month")
	}
	return plural(days/365, "year")
}

func plural(value int, unit string) string {
	if value == 1 {
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %ss", value, unit)
}

func formatBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor := int64(unit)
	unitName := "KiB"
	for value/divisor >= unit {
		divisor *= unit
		switch unitName {
		case "KiB":
			unitName = "MiB"
		case "MiB":
			unitName = "GiB"
		default:
			return fmt.Sprintf("%.1f %s", float64(value)/float64(divisor), unitName)
		}
	}
	return fmt.Sprintf("%.1f %s", float64(value)/float64(divisor), unitName)
}
