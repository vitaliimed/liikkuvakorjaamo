package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

type TemplateRenderer struct {
	landing  *template.Template
	thankyou *template.Template
	privacy  *template.Template
	review   *template.Template
	admin    *template.Template
}

func newTemplateRenderer() (*TemplateRenderer, error) {
	landingPath, err := resolveTemplatePath("landing.html")
	if err != nil {
		return nil, err
	}
	landing, err := template.ParseFiles(landingPath)
	if err != nil {
		return nil, err
	}

	thankyouPath, err := resolveTemplatePath("thankyou.html")
	if err != nil {
		return nil, err
	}
	thankyou, err := template.ParseFiles(thankyouPath)
	if err != nil {
		return nil, err
	}

	privacyPath, err := resolveTemplatePath("privacy.html")
	if err != nil {
		return nil, err
	}
	privacy, err := template.ParseFiles(privacyPath)
	if err != nil {
		return nil, err
	}

	reviewPath, err := resolveTemplatePath("review.html")
	if err != nil {
		return nil, err
	}
	review, err := template.ParseFiles(reviewPath)
	if err != nil {
		return nil, err
	}

	adminPath, err := resolveTemplatePath("admin.html")
	if err != nil {
		return nil, err
	}
	admin, err := template.New("admin.html").Funcs(template.FuncMap{"statusLabel": adminStatusLabel}).ParseFiles(adminPath)
	if err != nil {
		return nil, err
	}

	return &TemplateRenderer{
		landing:  landing,
		thankyou: thankyou,
		privacy:  privacy,
		review:   review,
		admin:    admin,
	}, nil
}

func (tr *TemplateRenderer) renderReview(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tr.review.Execute(w, data)
}

func (tr *TemplateRenderer) renderLanding(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tr.landing.Execute(w, data)
}

func (tr *TemplateRenderer) renderThankYou(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tr.thankyou.Execute(w, data)
}

func (tr *TemplateRenderer) renderPrivacy(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tr.privacy.Execute(w, data)
}

func (tr *TemplateRenderer) renderAdmin(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tr.admin.Execute(w, data)
}

func resolveTemplatePath(name string) (string, error) {
	paths := []string{
		filepath.Clean(filepath.Join("templates", name)),
		filepath.Clean(name),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("template %s not found in templates/ or project root", name)
}
