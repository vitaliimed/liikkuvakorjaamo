package main

import (
	"fmt"
	"html"
	"strings"
)

// Theme preferences are content settings, not runtime credentials. Every value
// is normalized and escaped before it becomes an HTML attribute.
func liikkuvaThemeHTML(body string, settings SiteSettings) string {
	settings = liikkuvaProDefaults(settings)
	initial := "light"
	if settings.ProTheme == "dark" {
		initial = "dark"
	}
	attributes := fmt.Sprintf(` data-theme="%s" data-theme-policy="%s" data-night-start="%s" data-night-end="%s"`,
		initial, html.EscapeString(settings.ProTheme), html.EscapeString(settings.ProNightStart), html.EscapeString(settings.ProNightEnd))
	body = strings.Replace(body, "<html", "<html"+attributes, 1)
	assets := `<link rel="stylesheet" href="/static/css/theme.css?v=20261031"><script src="/static/js/theme.js?v=20261031"></script>`
	return strings.Replace(body, "</head>", assets+"</head>", 1)
}
