package main

import (
	"fmt"
	"html"
	"regexp"
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
	light := settings.LogoURL
	if strings.TrimSpace(light) == "" {
		light = "/static/img/logo_liikkuva.svg"
	}
	// Brand marks occur in the public header and footer. Text remains CSS-colored.
	logoPattern := regexp.MustCompile(`(<a\b[^>]*class="brand"[^>]*>\s*)<img\b[^>]*>`)
	body = logoPattern.ReplaceAllStringFunc(body, func(block string) string {
		start := strings.Index(block, "<img")
		return block[:start] + `<img src="` + html.EscapeString(light) + `" data-logo-light="` + html.EscapeString(light) + `" data-logo-dark="` + html.EscapeString(settings.LogoDarkURL) + `" alt="" width="46" height="46">`
	})
	assets := `<link rel="stylesheet" href="/static/css/theme.css?v=20261031-compact2"><script src="/static/js/theme.js?v=20261031-client"></script>`
	return strings.Replace(body, "</head>", assets+"</head>", 1)
}
