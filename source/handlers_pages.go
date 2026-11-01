package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

type PageHandler struct {
	cfg      Config
	renderer *TemplateRenderer
}

func NewPageHandler(cfg Config, renderer *TemplateRenderer) *PageHandler {
	return &PageHandler{
		cfg:      cfg,
		renderer: renderer,
	}
}

func (h *PageHandler) LandingFI(w http.ResponseWriter, r *http.Request) {
	h.serveLanding(w, r, "fi")
}

func (h *PageHandler) LandingEN(w http.ResponseWriter, r *http.Request) {
	h.serveLanding(w, r, "en")
}

func (h *PageHandler) ThankYouFI(w http.ResponseWriter, r *http.Request) {
	h.serveThankYou(w, r, "fi")
}

func (h *PageHandler) ThankYouEN(w http.ResponseWriter, r *http.Request) {
	h.serveThankYou(w, r, "en")
}

func (h *PageHandler) Privacy(w http.ResponseWriter, r *http.Request) {
	lang := detectPrivacyLang(r, h.cfg.DefaultLanguage)
	h.servePrivacy(w, r, lang)
}

func (h *PageHandler) Robots(w http.ResponseWriter, r *http.Request) {
	settings, err := loadSiteSettings(r.Context(), h.cfg)
	settings = liikkuvaSettings(settings)
	if err == nil && !settings.SearchIndexingEnabled {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /internal/\nDisallow: /lead\n"))
		return
	}
	if err == nil && strings.TrimSpace(settings.RobotsTXT) != "" {
		content := strings.ReplaceAll(strings.TrimSpace(settings.RobotsTXT), "{{SITE_URL}}", strings.TrimRight(h.cfg.SiteURL, "/"))
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(content))
		return
	}

	fileBytes, err := os.ReadFile("static/robots.txt")
	if err != nil {
		content := fallbackRobotsTXT(h.cfg.SiteURL)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(content))
		return
	}

	content := strings.ReplaceAll(string(fileBytes), "{{SITE_URL}}", strings.TrimRight(h.cfg.SiteURL, "/"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(content))
}

func (h *PageHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	settings, settingsErr := loadSiteSettings(r.Context(), h.cfg)
	if settingsErr == nil && !settings.SearchIndexingEnabled {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`))
		return
	}
	fileBytes, err := os.ReadFile("static/sitemap.xml")
	if err != nil {
		content := fallbackSitemapXML(h.cfg.SiteURL)
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(content))
		return
	}

	content := strings.ReplaceAll(string(fileBytes), "{{SITE_URL}}", strings.TrimRight(h.cfg.SiteURL, "/"))
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(content))
}

func assetOrFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}

func faviconURLs(settings SiteSettings) (string, string, string, string, string) {
	return assetOrFallback(settings.FaviconSVGURL, "/static/img/favicon.svg?v=20260327b"),
		assetOrFallback(settings.Favicon32URL, "/static/img/favicon-32x32.png?v=20260327b"),
		assetOrFallback(settings.Favicon16URL, "/static/img/favicon-16x16.png?v=20260327b"),
		assetOrFallback(settings.FaviconICOURL, "/static/img/favicon.ico?v=20260327b"),
		assetOrFallback(settings.AppleTouchIconURL, "/static/img/apple-touch-icon.png?v=20260327b")
}

func consentModeEnabled(settings SiteSettings) bool {
	return ga4ID(settings) != "" ||
		strings.TrimSpace(settings.GoogleAdsTagID) != "" ||
		strings.TrimSpace(settings.MetaPixelID) != "" ||
		strings.TrimSpace(settings.LinkedInPartnerID) != "" ||
		strings.TrimSpace(settings.TikTokPixelID) != ""
}

func primaryGoogleTagID(settings SiteSettings) string {
	if value := ga4ID(settings); value != "" {
		return value
	}
	return strings.TrimSpace(settings.GoogleAdsTagID)
}

func detectPrivacyLang(r *http.Request, fallback string) string {
	lang := strings.TrimSpace(r.URL.Query().Get("lang"))
	if lang == "fi" || lang == "en" {
		return lang
	}
	return fallback
}

func landingPathByLang(lang string) string {
	if lang == "en" {
		return "/en"
	}
	return "/"
}

func fallbackRobotsTXT(siteURL string) string {
	baseURL := strings.TrimRight(siteURL, "/")
	return "User-agent: *\nAllow: /\n\nSitemap: " + baseURL + "/sitemap.xml\n"
}

func fallbackSitemapXML(siteURL string) string {
	baseURL := strings.TrimRight(siteURL, "/")
	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n" +
		"  <url><loc>" + baseURL + "/</loc></url>\n" +
		"  <url><loc>" + baseURL + "/en</loc></url>\n" +
		"  <url><loc>" + baseURL + "/privacy?lang=fi</loc></url>\n" +
		"  <url><loc>" + baseURL + "/privacy?lang=en</loc></url>\n" +
		"  <url><loc>" + baseURL + "/thank-you-fi</loc></url>\n" +
		"  <url><loc>" + baseURL + "/thank-you-en</loc></url>\n" +
		"</urlset>\n"
}

func (h *PageHandler) serveLanding(w http.ResponseWriter, r *http.Request, lang string) {
	texts := getLandingTexts(lang)
	content := getLandingContent(lang)

	seoDefaults := liikkuvaSettings(SiteSettings{})
	pageTitle, metaDescription := seoDefaults.LandingSEOTitleEN, seoDefaults.LandingSEODescriptionEN
	if lang == "fi" {
		pageTitle = seoDefaults.LandingSEOTitleFI
		metaDescription = seoDefaults.LandingSEODescriptionFI
	}
	robotsMeta := "index, follow"

	settings, err := loadSiteSettings(r.Context(), h.cfg)
	settings = liikkuvaSettings(settings)
	if err != nil {
		log.Printf("loadSiteSettings warning: %v", err)
	} else {
		content = applySiteSettingsToContent(content, settings, lang)
		pageTitle, metaDescription, robotsMeta = applyLandingPageSEO(pageTitle, metaDescription, robotsMeta, settings, lang)
	}

	if strings.TrimSpace(h.cfg.FirestoreProjectID) != "" {
		reviewRecords, reviewErr := listApprovedReviewRecords(r.Context(), h.cfg, lang, 12)
		if reviewErr != nil {
			log.Printf("load approved reviews warning: %v", reviewErr)
		} else if reviewCards := approvedReviewCards(reviewRecords, lang, 12); len(reviewCards) > 0 {
			content.Testimonials.Items = append(reviewCards, content.Testimonials.Items...)
		}
	}

	content = withoutLanguageClaims(content)
	pageURL := strings.TrimRight(h.cfg.SiteURL, "/") + landingPathByLang(lang)
	alternateFI := strings.TrimRight(h.cfg.SiteURL, "/") + landingPathByLang("fi")
	alternateEN := strings.TrimRight(h.cfg.SiteURL, "/") + landingPathByLang("en")
	faviconSVGURL, favicon32URL, favicon16URL, faviconICOURL, appleTouchIconURL := faviconURLs(settings)

	data := LandingPageData{
		Lang:                         texts["LANG"],
		PageTitle:                    pageTitle,
		MetaDescription:              metaDescription,
		RobotsMeta:                   robotsMeta,
		CanonicalURL:                 pageURL,
		HrefLangFI:                   alternateFI,
		HrefLangEN:                   alternateEN,
		OGTitle:                      pageTitle,
		OGDescription:                metaDescription,
		OGURL:                        pageURL,
		FaviconSVGURL:                faviconSVGURL,
		Favicon32URL:                 favicon32URL,
		Favicon16URL:                 favicon16URL,
		FaviconICOURL:                faviconICOURL,
		AppleTouchIconURL:            appleTouchIconURL,
		EnableConsentMode:            consentModeEnabled(settings),
		PrimaryGoogleTagID:           primaryGoogleTagID(settings),
		GA4MeasurementID:             ga4ID(settings),
		GoogleAdsTagID:               strings.TrimSpace(settings.GoogleAdsTagID),
		GoogleAdsLeadConversionLabel: strings.TrimSpace(settings.GoogleAdsLeadConversionLabel),
		MetaPixelID:                  strings.TrimSpace(settings.MetaPixelID),
		LinkedInPartnerID:            strings.TrimSpace(settings.LinkedInPartnerID),
		LinkedInLeadConversionID:     strings.TrimSpace(settings.LinkedInLeadConversionID),
		TikTokPixelID:                strings.TrimSpace(settings.TikTokPixelID),
		HeroStyleVars:                template.CSS(buildHeroStyleVars(settings)),
		HeroSceneImageStyle:          template.CSS(buildHeroSceneImageStyle(settings)),
		HeroPersonLayerStyle:         template.CSS(buildHeroPersonLayerStyle(settings)),
		HeroPersonImageStyle:         template.CSS(buildHeroPersonImageStyle(settings)),
		AreasStyleVars:               template.CSS(buildAreasStyleVars(settings)),
		AreasMapImageStyle:           template.CSS(buildAreasMapImageStyle(settings)),
		ServiceImageStyle:            template.CSS(buildServiceImageStyle(settings)),

		HeroTitle:    content.Hero.Title,
		HeroSubtitle: content.Hero.Subtitle,
		FormTitle:    content.Form.Title,

		LabelName:        content.Form.LabelName,
		LabelPhone:       content.Form.LabelPhone,
		LabelEmail:       content.Form.LabelEmail,
		LabelAddress:     content.Form.LabelAddress,
		LabelAddressHint: content.Form.LabelAddressHint,
		LabelCar:         content.Form.LabelCar,
		LabelMessage:     content.Form.LabelMessage,

		PlaceholderName:        content.Form.PlaceholderName,
		PlaceholderPhone:       content.Form.PlaceholderPhone,
		PlaceholderEmail:       content.Form.PlaceholderEmail,
		PlaceholderAddress:     content.Form.PlaceholderAddress,
		PlaceholderAddressHint: content.Form.PlaceholderAddressHint,
		PlaceholderCar:         content.Form.PlaceholderCar,
		PlaceholderMessage:     content.Form.PlaceholderMessage,

		ButtonSubmit:   content.Form.SubmitButton,
		SuccessMessage: content.Form.SuccessMessage,
		ErrorMessage:   content.Form.ErrorMessage,
		LanguageValue:  lang,

		Content: content,
	}

	if err := h.renderer.renderLanding(w, data); err != nil {
		http.Error(w, "failed to render landing", http.StatusInternalServerError)
		return
	}
}

func (h *PageHandler) serveThankYou(w http.ResponseWriter, r *http.Request, lang string) {
	texts := getThankYouTexts(lang)
	landingContent := getLandingContent(lang)
	footerContent := landingContent.Footer
	siteName := landingContent.SiteName

	settings, err := loadSiteSettings(r.Context(), h.cfg)
	settings = liikkuvaSettings(settings)
	if err != nil {
		log.Printf("loadSiteSettings warning: %v", err)
	} else {
		texts = applyThankYouSettingsToTexts(texts, settings, lang)
		landingContent = applySiteSettingsToContent(landingContent, settings, lang)
		footerContent = landingContent.Footer
		siteName = landingContent.SiteName
	}

	pageURL := strings.TrimRight(h.cfg.SiteURL, "/") + "/thank-you-" + lang
	alternateFI := strings.TrimRight(h.cfg.SiteURL, "/") + "/thank-you-fi"
	alternateEN := strings.TrimRight(h.cfg.SiteURL, "/") + "/thank-you-en"
	faviconSVGURL, favicon32URL, favicon16URL, faviconICOURL, appleTouchIconURL := faviconURLs(settings)

	data := ThankYouPageData{
		Lang:                         texts["LANG"],
		PageTitle:                    texts["PAGE_TITLE"],
		MetaDescription:              texts["META_DESCRIPTION"],
		RobotsMeta:                   texts["ROBOTS_META"],
		CanonicalURL:                 pageURL,
		SiteName:                     siteName,
		Footer:                       footerContent,
		HrefLangFI:                   alternateFI,
		HrefLangEN:                   alternateEN,
		OGTitle:                      texts["PAGE_TITLE"],
		OGDescription:                texts["META_DESCRIPTION"],
		OGURL:                        pageURL,
		LogoURL:                      assetOrFallback(settings.LogoURL, "/static/img/logo_liikkuva.svg?v=20260327b"),
		FaviconSVGURL:                faviconSVGURL,
		Favicon32URL:                 favicon32URL,
		Favicon16URL:                 favicon16URL,
		FaviconICOURL:                faviconICOURL,
		AppleTouchIconURL:            appleTouchIconURL,
		EnableConsentMode:            consentModeEnabled(settings),
		PrimaryGoogleTagID:           primaryGoogleTagID(settings),
		GA4MeasurementID:             ga4ID(settings),
		GoogleAdsTagID:               strings.TrimSpace(settings.GoogleAdsTagID),
		GoogleAdsLeadConversionLabel: strings.TrimSpace(settings.GoogleAdsLeadConversionLabel),
		MetaPixelID:                  strings.TrimSpace(settings.MetaPixelID),
		LinkedInPartnerID:            strings.TrimSpace(settings.LinkedInPartnerID),
		LinkedInLeadConversionID:     strings.TrimSpace(settings.LinkedInLeadConversionID),
		TikTokPixelID:                strings.TrimSpace(settings.TikTokPixelID),
		ThankYouTitle:                texts["THANKYOU_TITLE"],
		ThankYouText:                 texts["THANKYOU_TEXT"],
		ThankYouButton:               texts["THANKYOU_BUTTON"],
		ThankYouButtonURL:            texts["THANKYOU_BUTTON_URL"],
		VisitPriceEUR:                visitPriceEUR,
	}

	if err := h.renderer.renderThankYou(w, data); err != nil {
		http.Error(w, "failed to render thankyou", http.StatusInternalServerError)
		return
	}
}

func (h *PageHandler) servePrivacy(w http.ResponseWriter, r *http.Request, lang string) {
	texts := getPrivacyTexts(lang)

	settings, err := loadSiteSettings(r.Context(), h.cfg)
	settings = liikkuvaSettings(settings)
	if err != nil {
		log.Printf("loadSiteSettings warning: %v", err)
	} else {
		texts = applyPrivacySettingsToTexts(texts, settings, lang)
	}

	pageURL := strings.TrimRight(h.cfg.SiteURL, "/") + "/privacy"
	alternateFI := strings.TrimRight(h.cfg.SiteURL, "/") + "/privacy?lang=fi"
	alternateEN := strings.TrimRight(h.cfg.SiteURL, "/") + "/privacy?lang=en"
	faviconSVGURL, favicon32URL, favicon16URL, faviconICOURL, appleTouchIconURL := faviconURLs(settings)

	data := PrivacyPageData{
		Lang:                         texts["LANG"],
		PageTitle:                    texts["PAGE_TITLE"],
		MetaDescription:              texts["META_DESCRIPTION"],
		RobotsMeta:                   texts["ROBOTS_META"],
		CanonicalURL:                 pageURL,
		HrefLangFI:                   alternateFI,
		HrefLangEN:                   alternateEN,
		OGTitle:                      texts["PAGE_TITLE"],
		OGDescription:                texts["META_DESCRIPTION"],
		OGURL:                        pageURL,
		LogoURL:                      assetOrFallback(settings.LogoURL, "/static/img/logo_liikkuva.svg?v=20260327b"),
		FaviconSVGURL:                faviconSVGURL,
		Favicon32URL:                 favicon32URL,
		Favicon16URL:                 favicon16URL,
		FaviconICOURL:                faviconICOURL,
		AppleTouchIconURL:            appleTouchIconURL,
		EnableConsentMode:            consentModeEnabled(settings),
		PrimaryGoogleTagID:           primaryGoogleTagID(settings),
		GA4MeasurementID:             ga4ID(settings),
		GoogleAdsTagID:               strings.TrimSpace(settings.GoogleAdsTagID),
		GoogleAdsLeadConversionLabel: strings.TrimSpace(settings.GoogleAdsLeadConversionLabel),
		MetaPixelID:                  strings.TrimSpace(settings.MetaPixelID),
		LinkedInPartnerID:            strings.TrimSpace(settings.LinkedInPartnerID),
		LinkedInLeadConversionID:     strings.TrimSpace(settings.LinkedInLeadConversionID),
		TikTokPixelID:                strings.TrimSpace(settings.TikTokPixelID),
		PrivacyTitle:                 texts["PRIVACY_TITLE"],
		PrivacyIntro:                 texts["PRIVACY_INTRO"],
		Section1Title:                texts["SECTION_1_TITLE"],
		Section1Text:                 texts["SECTION_1_TEXT"],
		Section2Title:                texts["SECTION_2_TITLE"],
		Section2Text:                 texts["SECTION_2_TEXT"],
		Section3Title:                texts["SECTION_3_TITLE"],
		Section3Text:                 texts["SECTION_3_TEXT"],
		Section4Title:                texts["SECTION_4_TITLE"],
		Section4Text:                 texts["SECTION_4_TEXT"],
		Section5Title:                texts["SECTION_5_TITLE"],
		Section5Text:                 texts["SECTION_5_TEXT"],
	}

	if err := h.renderer.renderPrivacy(w, data); err != nil {
		http.Error(w, "failed to render privacy", http.StatusInternalServerError)
		return
	}
}
