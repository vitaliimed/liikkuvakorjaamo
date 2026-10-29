package main

import "net/http"

func newRouter(cfg Config, renderer *TemplateRenderer) *http.ServeMux {
	mux := http.NewServeMux()

	pages := NewPageHandler(cfg, renderer)
	leads := NewLeadHandler(cfg)
	admin := NewAdminHandler(cfg, renderer)
	reviews := NewReviewHandler(cfg, renderer)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		if cfg.DefaultLanguage == "en" {
			pages.LandingEN(w, r)
			return
		}

		pages.LandingFI(w, r)
	})

	mux.HandleFunc("/fi", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/en", pages.LandingEN)
	mux.HandleFunc("/thank-you-fi", pages.ThankYouFI)
	mux.HandleFunc("/thank-you-en", pages.ThankYouEN)
	mux.HandleFunc("/privacy", pages.Privacy)
	mux.HandleFunc("/robots.txt", pages.Robots)
	mux.HandleFunc("/sitemap.xml", pages.Sitemap)
	mux.HandleFunc("/lead", leads.Submit)
	mux.HandleFunc("/review", reviews.Handle)
	mux.HandleFunc("/review/", func(w http.ResponseWriter, r *http.Request) {
		target := "/review"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
	mux.HandleFunc("/reviews", func(w http.ResponseWriter, r *http.Request) {
		target := "/review"
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
	mux.HandleFunc("/internal/process-lead", leads.ProcessLead)
	mux.HandleFunc("/admin", admin.Show)
	mux.HandleFunc(standaloneEmailRoute, admin.StandaloneEmail)
	mux.HandleFunc("/admin/login", admin.Login)
	mux.HandleFunc("/admin/access", admin.Access)
	mux.HandleFunc("/admin/recovery", admin.Recovery)
	mux.HandleFunc("/admin/reset-password", admin.ResetPassword)
	mux.HandleFunc("/admin/confirm-recovery", admin.ConfirmRecovery)
	mux.HandleFunc("/admin/logout", admin.Logout)
	mux.HandleFunc("/admin/save", admin.Save)
	mux.HandleFunc("/admin/leads/retry", admin.RetryLead)
	mux.HandleFunc("/admin/leads/update-status", admin.UpdateClientStatus)
	mux.HandleFunc("/admin/leads/send-confirmation-email", admin.SendConfirmationEmail)
	mux.HandleFunc("/admin/leads/send-ready-email", admin.SendReadyEmail)
	mux.HandleFunc("/admin/leads/send-custom-email", admin.SendCustomEmail)
	mux.HandleFunc("/admin/leads/send-review-email", admin.SendReviewEmail)
	mux.HandleFunc("/admin/reviews/moderate", admin.ModerateReview)

	staticHandler := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", withCache(staticHandler)))

	return mux
}
