package main

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const adminCookieName = "__session"

type AdminHandler struct {
	cfg      Config
	renderer *TemplateRenderer
}

func NewAdminHandler(cfg Config, renderer *TemplateRenderer) *AdminHandler {
	return &AdminHandler{
		cfg:      cfg,
		renderer: renderer,
	}
}

func normalizeTestimonialRating(value string) int {
	rating, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 5
	}

	if rating < 1 {
		return 1
	}

	if rating > 5 {
		return 5
	}

	return rating
}

func parseAdminTestimonials(r *http.Request) []AdminTestimonialItem {
	names := r.Form["testimonial_name[]"]
	quotesFI := r.Form["testimonial_quote_fi[]"]
	quotesEN := r.Form["testimonial_quote_en[]"]
	photos := r.Form["testimonial_photo_url[]"]
	ratings := r.Form["testimonial_rating[]"]

	maxLen := 0
	for _, n := range []int{len(names), len(quotesFI), len(quotesEN), len(photos), len(ratings)} {
		if n > maxLen {
			maxLen = n
		}
	}

	items := make([]AdminTestimonialItem, 0, maxLen)

	for i := 0; i < maxLen; i++ {
		var name, quoteFI, quoteEN, photoURL, ratingValue string

		if i < len(names) {
			name = strings.TrimSpace(names[i])
		}
		if i < len(quotesFI) {
			quoteFI = strings.TrimSpace(quotesFI[i])
		}
		if i < len(quotesEN) {
			quoteEN = strings.TrimSpace(quotesEN[i])
		}
		if i < len(photos) {
			photoURL = strings.TrimSpace(photos[i])
		}
		if i < len(ratings) {
			ratingValue = strings.TrimSpace(ratings[i])
		}

		if name == "" && quoteFI == "" && quoteEN == "" && photoURL == "" {
			continue
		}
		if name == "" {
			continue
		}
		if quoteFI == "" && quoteEN == "" {
			continue
		}

		items = append(items, AdminTestimonialItem{
			Name:     name,
			QuoteFI:  quoteFI,
			QuoteEN:  quoteEN,
			PhotoURL: photoURL,
			Rating:   normalizeTestimonialRating(ratingValue),
		})
	}

	return items
}

func parseAdminServices(r *http.Request) []AdminServiceItem {
	badgesFI := r.Form["service_badge_fi[]"]
	pricesFI := r.Form["service_price_fi[]"]
	titlesFI := r.Form["service_title_fi[]"]
	textsFI := r.Form["service_text_fi[]"]
	badgesEN := r.Form["service_badge_en[]"]
	pricesEN := r.Form["service_price_en[]"]
	titlesEN := r.Form["service_title_en[]"]
	textsEN := r.Form["service_text_en[]"]
	images := r.Form["service_image_url[]"]

	maxLen := 0
	for _, n := range []int{len(badgesFI), len(pricesFI), len(titlesFI), len(textsFI), len(badgesEN), len(pricesEN), len(titlesEN), len(textsEN), len(images)} {
		if n > maxLen {
			maxLen = n
		}
	}

	items := make([]AdminServiceItem, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		item := AdminServiceItem{}
		if i < len(badgesFI) {
			item.BadgeFI = strings.TrimSpace(badgesFI[i])
		}
		if i < len(pricesFI) {
			item.PriceFI = strings.TrimSpace(pricesFI[i])
		}
		if i < len(titlesFI) {
			item.TitleFI = strings.TrimSpace(titlesFI[i])
		}
		if i < len(textsFI) {
			item.TextFI = strings.TrimSpace(textsFI[i])
		}
		if i < len(badgesEN) {
			item.BadgeEN = strings.TrimSpace(badgesEN[i])
		}
		if i < len(pricesEN) {
			item.PriceEN = strings.TrimSpace(pricesEN[i])
		}
		if i < len(titlesEN) {
			item.TitleEN = strings.TrimSpace(titlesEN[i])
		}
		if i < len(textsEN) {
			item.TextEN = strings.TrimSpace(textsEN[i])
		}
		if i < len(images) {
			item.ImageURL = strings.TrimSpace(images[i])
		}

		if item.BadgeFI == "" && item.PriceFI == "" && item.TitleFI == "" && item.TextFI == "" && item.BadgeEN == "" && item.PriceEN == "" && item.TitleEN == "" && item.TextEN == "" && item.ImageURL == "" {
			continue
		}

		hasFI := item.TitleFI != "" && item.TextFI != ""
		hasEN := item.TitleEN != "" && item.TextEN != ""
		if !hasFI && !hasEN {
			continue
		}

		items = append(items, item)
	}
	return items
}

func parseAdminCases(r *http.Request) []AdminCaseItem {
	tagsFI := r.Form["case_tag_fi[]"]
	titlesFI := r.Form["case_title_fi[]"]
	textsFI := r.Form["case_text_fi[]"]
	p1fi := r.Form["case_point_1_fi[]"]
	p2fi := r.Form["case_point_2_fi[]"]
	p3fi := r.Form["case_point_3_fi[]"]
	p4fi := r.Form["case_point_4_fi[]"]
	tagsEN := r.Form["case_tag_en[]"]
	titlesEN := r.Form["case_title_en[]"]
	textsEN := r.Form["case_text_en[]"]
	p1en := r.Form["case_point_1_en[]"]
	p2en := r.Form["case_point_2_en[]"]
	p3en := r.Form["case_point_3_en[]"]
	p4en := r.Form["case_point_4_en[]"]

	maxLen := 0
	for _, n := range []int{len(tagsFI), len(titlesFI), len(textsFI), len(p1fi), len(p2fi), len(p3fi), len(p4fi), len(tagsEN), len(titlesEN), len(textsEN), len(p1en), len(p2en), len(p3en), len(p4en)} {
		if n > maxLen {
			maxLen = n
		}
	}

	items := make([]AdminCaseItem, 0, maxLen)
	for i := 0; i < maxLen; i++ {
		item := AdminCaseItem{}
		if i < len(tagsFI) {
			item.TagFI = strings.TrimSpace(tagsFI[i])
		}
		if i < len(titlesFI) {
			item.TitleFI = strings.TrimSpace(titlesFI[i])
		}
		if i < len(textsFI) {
			item.TextFI = strings.TrimSpace(textsFI[i])
		}
		item.PointsFI = filterNonEmptyStrings([]string{
			valueAt(p1fi, i), valueAt(p2fi, i), valueAt(p3fi, i), valueAt(p4fi, i),
		})
		if i < len(tagsEN) {
			item.TagEN = strings.TrimSpace(tagsEN[i])
		}
		if i < len(titlesEN) {
			item.TitleEN = strings.TrimSpace(titlesEN[i])
		}
		if i < len(textsEN) {
			item.TextEN = strings.TrimSpace(textsEN[i])
		}
		item.PointsEN = filterNonEmptyStrings([]string{
			valueAt(p1en, i), valueAt(p2en, i), valueAt(p3en, i), valueAt(p4en, i),
		})

		if item.TagFI == "" && item.TitleFI == "" && item.TextFI == "" && len(item.PointsFI) == 0 && item.TagEN == "" && item.TitleEN == "" && item.TextEN == "" && len(item.PointsEN) == 0 {
			continue
		}

		hasFI := item.TitleFI != "" && item.TextFI != ""
		hasEN := item.TitleEN != "" && item.TextEN != ""
		if !hasFI && !hasEN {
			continue
		}

		items = append(items, item)
	}
	return items
}

func padAdminServices(items []AdminServiceItem, size int) []AdminServiceItem {
	if size <= 0 {
		return items
	}

	out := make([]AdminServiceItem, size)
	copy(out, items)
	return out
}

func padAdminCases(items []AdminCaseItem, size int) []AdminCaseItem {
	if size <= 0 {
		return items
	}

	out := make([]AdminCaseItem, size)
	copy(out, items)
	return out
}

func normalizeAdminSettingsForTemplate(settings SiteSettings) SiteSettings {
	settings = adminContentDefaults(liikkuvaEmailDefaults(liikkuvaSettings(liikkuvaProDefaults(settings))))
	content := getLandingContent("fi")
	settings.LogoURL = assetOrFallback(settings.LogoURL, content.Header.LogoURL)
	settings.HeroBackgroundURL = assetOrFallback(settings.HeroBackgroundURL, content.Hero.BackgroundURL)
	settings.AreasMapImageURL = assetOrFallback(settings.AreasMapImageURL, content.Areas.MapImageURL)
	settings.FaviconSVGURL, settings.Favicon32URL, settings.Favicon16URL, settings.FaviconICOURL, settings.AppleTouchIconURL = faviconURLs(settings)
	if len(settings.ServiceItems) == 0 {
		settings.ServiceItems = []AdminServiceItem{{}}
	}
	if len(settings.CaseItems) == 0 {
		settings.CaseItems = []AdminCaseItem{{}}
	}
	settings.CaseItems = padAdminCasePoints(settings.CaseItems)

	if !settings.SectionVisibilityConfigured {
		settings.StatsEnabled = false
		settings.ServicesEnabled = true
		settings.ProcessEnabled = true
		settings.CasesEnabled = true
		settings.CapabilityEnabled = true
		settings.AreasEnabled = true
		settings.TestimonialsEnabled = true
		settings.FAQEnabled = true
		settings.FinalCTAEnabled = true
	}
	return settings
}

func (h *AdminHandler) Show(w http.ResponseWriter, r *http.Request) {
	if !isAdminAuthenticated(h.cfg, r) {
		h.accessLogin(w, r)
		return
	}

	settings, err := loadSiteSettings(r.Context(), h.cfg)
	if err != nil {
		renderAccess(w, http.StatusServiceUnavailable, accessPageData{Mode: "unavailable", Error: "Не удалось загрузить настройки сайта. Повторите позже. Форма редактирования откроется после восстановления связи с хранилищем."})
		return
	}

	settings = normalizeAdminSettingsForTemplate(settings)

	statusFilter := normalizeAdminStatusFilter(r.URL.Query().Get("status"))
	clientStatusFilter := normalizeClientStatusFilter(r.URL.Query().Get("client_status"))

	records, err := listLeadRecords(r.Context(), h.cfg, 50, statusFilter)
	if err != nil {
		if renderErr := h.renderAdmin(w, r, AdminPageData{
			PageTitle: "Кабинет Liikkuva",
			Error:     err.Error(),
			Settings:  settings,
			ActiveTab: normalizeAdminTab(r.URL.Query().Get("tab"), false),
		}); renderErr != nil {
			http.Error(w, "failed to render admin lead error page: "+renderErr.Error(), http.StatusInternalServerError)
		}
		return
	}
	records = filterLeadRecordsByClientStatus(records, clientStatusFilter)

	reviewStatusFilter := normalizeReviewStatusFilter(r.URL.Query().Get("review_status"))
	allReviews, reviewsErr := listReviewRecords(r.Context(), h.cfg, 100)
	reviewError := ""
	if reviewsErr != nil {
		reviewError = "load reviews failed: " + reviewsErr.Error()
		allReviews = nil
	}
	visibleReviews := filterReviewRecords(allReviews, reviewStatusFilter)

	var selectedLead *AdminLeadDetails
	selectedLeadID := strings.TrimSpace(r.URL.Query().Get("lead_id"))
	if selectedLeadID != "" {
		record, err := getLeadRecord(r.Context(), h.cfg, selectedLeadID)
		if err != nil {
			if renderErr := h.renderAdmin(w, r, AdminPageData{
				PageTitle:    "Кабинет Liikkuva",
				Error:        err.Error(),
				Settings:     settings,
				ActiveTab:    normalizeAdminTab(r.URL.Query().Get("tab"), true),
				StatusFilter: statusFilter,
				Leads:        buildAdminLeadList(records),
			}); renderErr != nil {
				http.Error(w, "failed to render admin selected lead error page: "+renderErr.Error(), http.StatusInternalServerError)
			}
			return
		}

		details := buildAdminLeadDetails(record)
		selectedLead = &details
	}

	overviewTotalLeads := len(records)
	overviewFailedLeads := 0
	overviewPendingEmail := 0
	for _, record := range records {
		if strings.EqualFold(strings.TrimSpace(record.ProcessingStatus), "failed") {
			overviewFailedLeads++
		}
		if strings.TrimSpace(record.Lead.Email) != "" && !strings.EqualFold(strings.TrimSpace(record.ReadyEmailStatus), "sent") {
			overviewPendingEmail++
		}
	}
	overviewMissingAssets := countAdminMissingAssets(settings)
	overviewPendingReviews := 0
	for _, review := range allReviews {
		if review.Status == reviewStatusPending {
			overviewPendingReviews++
		}
	}

	activeTab := normalizeAdminTab(r.URL.Query().Get("tab"), selectedLead != nil)

	queryError := strings.TrimSpace(r.URL.Query().Get("error"))
	if reviewError != "" {
		if queryError != "" {
			queryError += " | "
		}
		queryError += reviewError
	}

	if err := h.renderAdmin(w, r, AdminPageData{
		PageTitle:              "Кабинет Liikkuva",
		Saved:                  r.URL.Query().Get("saved") == "1",
		MailEnabled:            mailNotificationsEnabled(h.cfg),
		Settings:               settings,
		ActiveTab:              activeTab,
		OverviewTotalLeads:     overviewTotalLeads,
		OverviewFailedLeads:    overviewFailedLeads,
		OverviewPendingEmail:   overviewPendingEmail,
		OverviewMissingAssets:  overviewMissingAssets,
		OverviewPendingReviews: overviewPendingReviews,
		StatusFilter:           statusFilter,
		ClientStatusFilter:     clientStatusFilter,
		Leads:                  buildAdminLeadList(records),
		SelectedLead:           selectedLead,
		RetryQueued:            r.URL.Query().Get("retry") == "1",
		ReadyEmailSent:         r.URL.Query().Get("ready_email") == "1",
		ConfirmationEmailSent:  r.URL.Query().Get("confirmation_email") == "1",
		ReviewEmailSent:        r.URL.Query().Get("review_email") == "1",
		StatusUpdated:          r.URL.Query().Get("status_updated") == "1",
		ReviewStatusFilter:     reviewStatusFilter,
		Reviews:                buildAdminReviewList(visibleReviews),
		Error:                  queryError,
	}); err != nil {
		http.Error(w, "failed to render admin page: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.accessLogin(w, r)
}
func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.accessLogout(w, r)
}

func (h *AdminHandler) Save(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	activeTab := normalizeAdminTab(r.FormValue("active_tab"), false)

	revision, revisionErr := strconv.ParseInt(r.FormValue("settings_revision"), 10, 64)
	if revisionErr != nil || revision < 0 {
		http.Error(w, "Обновите страницу кабинета перед сохранением.", http.StatusBadRequest)
		return
	}
	settings := SiteSettings{
		Revision:                  revision,
		ProVisitEUR:               liikkuvaInt(r.FormValue("ProVisitEUR"), 70, 1, 2000),
		ProDiagnosticEUR:          liikkuvaInt(r.FormValue("ProDiagnosticEUR"), 50, 1, 2000),
		ProFontSize:               liikkuvaInt(r.FormValue("ProFontSize"), 18, 16, 22),
		ProAccent:                 strings.TrimSpace(r.FormValue("ProAccent")),
		SearchIndexingEnabled:     r.FormValue("search_indexing_enabled") == "1",
		ProTheme:                  strings.TrimSpace(r.FormValue("ProTheme")),
		ProNightStart:             strings.TrimSpace(r.FormValue("ProNightStart")),
		ProNightEnd:               strings.TrimSpace(r.FormValue("ProNightEnd")),
		ProPricingText:            strings.TrimSpace(r.FormValue("ProPricingText")),
		ProAreasText:              strings.TrimSpace(r.FormValue("ProAreasText")),
		ProDiagnosticsText:        strings.TrimSpace(r.FormValue("ProDiagnosticsText")),
		ProPricingTitle:           strings.TrimSpace(r.FormValue("ProPricingTitle")),
		ProPricingDescription:     strings.TrimSpace(r.FormValue("ProPricingDescription")),
		ProPricingHeading:         strings.TrimSpace(r.FormValue("ProPricingHeading")),
		ProAreasTitle:             strings.TrimSpace(r.FormValue("ProAreasTitle")),
		ProAreasDescription:       strings.TrimSpace(r.FormValue("ProAreasDescription")),
		ProAreasHeading:           strings.TrimSpace(r.FormValue("ProAreasHeading")),
		ProDiagnosticsTitle:       strings.TrimSpace(r.FormValue("ProDiagnosticsTitle")),
		ProDiagnosticsDescription: strings.TrimSpace(r.FormValue("ProDiagnosticsDescription")),
		ProDiagnosticsHeading:     strings.TrimSpace(r.FormValue("ProDiagnosticsHeading")),

		Phone:                        strings.TrimSpace(r.FormValue("phone")),
		WhatsAppNumber:               strings.TrimSpace(r.FormValue("whatsapp_number")),
		ServiceEmail:                 strings.TrimSpace(r.FormValue("service_email")),
		BusinessName:                 strings.TrimSpace(r.FormValue("business_name")),
		YTunnus:                      strings.TrimSpace(r.FormValue("y_tunnus")),
		BaseCity:                     strings.TrimSpace(r.FormValue("base_city")),
		WorkingHoursFI:               strings.TrimSpace(r.FormValue("working_hours_fi")),
		WorkingHoursEN:               strings.TrimSpace(r.FormValue("working_hours_en")),
		LogoURL:                      strings.TrimSpace(r.FormValue("logo_url")),
		LogoDarkURL:                  strings.TrimSpace(r.FormValue("logo_dark_url")),
		EmailLogoURL:                 strings.TrimSpace(r.FormValue("email_logo_url")),
		EmailConfirmationSubjectFI:   strings.TrimSpace(r.FormValue("email_confirmation_subject_fi")),
		EmailConfirmationSubjectEN:   strings.TrimSpace(r.FormValue("email_confirmation_subject_en")),
		EmailConfirmationTextFI:      strings.TrimSpace(r.FormValue("email_confirmation_text_fi")),
		EmailConfirmationTextEN:      strings.TrimSpace(r.FormValue("email_confirmation_text_en")),
		EmailReadySubjectFI:          strings.TrimSpace(r.FormValue("email_ready_subject_fi")),
		EmailReadySubjectEN:          strings.TrimSpace(r.FormValue("email_ready_subject_en")),
		EmailReadyTextFI:             strings.TrimSpace(r.FormValue("email_ready_text_fi")),
		EmailReadyTextEN:             strings.TrimSpace(r.FormValue("email_ready_text_en")),
		EmailSignatureFI:             strings.TrimSpace(r.FormValue("email_signature_fi")),
		EmailSignatureEN:             strings.TrimSpace(r.FormValue("email_signature_en")),
		AnalyticsDisabled:            r.FormValue("analytics_disabled") == "1",
		AnalyticsEventsConfigured:    true,
		AnalyticsDisabledEvents:      disabledAnalyticsEvents(r.Form["analytics_event"]),
		InstagramURL:                 strings.TrimSpace(r.FormValue("instagram_url")),
		TikTokURL:                    strings.TrimSpace(r.FormValue("tiktok_url")),
		FacebookGroupURL:             strings.TrimSpace(r.FormValue("facebook_group_url")),
		GA4MeasurementID:             strings.TrimSpace(r.FormValue("ga4_measurement_id")),
		GoogleAdsTagID:               strings.TrimSpace(r.FormValue("google_ads_tag_id")),
		GoogleAdsLeadConversionLabel: strings.TrimSpace(r.FormValue("google_ads_lead_conversion_label")),
		MetaPixelID:                  strings.TrimSpace(r.FormValue("meta_pixel_id")),
		LinkedInPartnerID:            strings.TrimSpace(r.FormValue("linkedin_partner_id")),
		LinkedInLeadConversionID:     strings.TrimSpace(r.FormValue("linkedin_lead_conversion_id")),
		TikTokPixelID:                strings.TrimSpace(r.FormValue("tiktok_pixel_id")),
		Testimonials:                 parseAdminTestimonials(r),
		ServiceItems:                 parseAdminServices(r),
		CaseItems:                    parseAdminCases(r),

		FaviconSVGURL:        strings.TrimSpace(r.FormValue("favicon_svg_url")),
		Favicon32URL:         strings.TrimSpace(r.FormValue("favicon_32_url")),
		Favicon16URL:         strings.TrimSpace(r.FormValue("favicon_16_url")),
		FaviconICOURL:        strings.TrimSpace(r.FormValue("favicon_ico_url")),
		AppleTouchIconURL:    strings.TrimSpace(r.FormValue("apple_touch_icon_url")),
		HeroPersonDesktopURL: strings.TrimSpace(r.FormValue("hero_person_desktop_url")),
		HeroPersonAltFI:      strings.TrimSpace(r.FormValue("hero_person_alt_fi")),
		HeroPersonAltEN:      strings.TrimSpace(r.FormValue("hero_person_alt_en")),
		HeroBackgroundURL:    strings.TrimSpace(r.FormValue("hero_background_url")),
		AreasMapImageURL:     strings.TrimSpace(r.FormValue("areas_map_image_url")),
		ServicesImage1URL:    strings.TrimSpace(r.FormValue("services_image_1_url")),
		ServicesImage2URL:    strings.TrimSpace(r.FormValue("services_image_2_url")),
		ServicesImage3URL:    strings.TrimSpace(r.FormValue("services_image_3_url")),
		ServicesImage4URL:    strings.TrimSpace(r.FormValue("services_image_4_url")),
		ServicesImage5URL:    strings.TrimSpace(r.FormValue("services_image_5_url")),
		ServicesImage6URL:    strings.TrimSpace(r.FormValue("services_image_6_url")),

		HeroScenePreset:     strings.TrimSpace(r.FormValue("hero_scene_preset")),
		HeroSceneBrightness: strings.TrimSpace(r.FormValue("hero_scene_brightness")),
		HeroSceneContrast:   strings.TrimSpace(r.FormValue("hero_scene_contrast")),
		HeroSceneSaturation: strings.TrimSpace(r.FormValue("hero_scene_saturation")),
		HeroSceneBlur:       strings.TrimSpace(r.FormValue("hero_scene_blur")),
		HeroSceneOpacity:    strings.TrimSpace(r.FormValue("hero_scene_opacity")),
		HeroSceneScale:      strings.TrimSpace(r.FormValue("hero_scene_scale")),
		HeroOverlayOpacity:  strings.TrimSpace(r.FormValue("hero_overlay_opacity")),

		HeroPersonPreset:     strings.TrimSpace(r.FormValue("hero_person_preset")),
		HeroPersonBrightness: strings.TrimSpace(r.FormValue("hero_person_brightness")),
		HeroPersonContrast:   strings.TrimSpace(r.FormValue("hero_person_contrast")),
		HeroPersonSaturation: strings.TrimSpace(r.FormValue("hero_person_saturation")),
		HeroPersonOpacity:    strings.TrimSpace(r.FormValue("hero_person_opacity")),
		HeroPersonScale:      strings.TrimSpace(r.FormValue("hero_person_scale")),
		HeroPersonPositionX:  strings.TrimSpace(r.FormValue("hero_person_position_x")),
		HeroPersonPositionY:  strings.TrimSpace(r.FormValue("hero_person_position_y")),
		HeroPersonShadow:     strings.TrimSpace(r.FormValue("hero_person_shadow")),

		AreasMapPreset:      strings.TrimSpace(r.FormValue("areas_map_preset")),
		AreasMapBrightness:  strings.TrimSpace(r.FormValue("areas_map_brightness")),
		AreasMapContrast:    strings.TrimSpace(r.FormValue("areas_map_contrast")),
		AreasMapSaturation:  strings.TrimSpace(r.FormValue("areas_map_saturation")),
		AreasMapBlur:        strings.TrimSpace(r.FormValue("areas_map_blur")),
		AreasMapOpacity:     strings.TrimSpace(r.FormValue("areas_map_opacity")),
		AreasMapScale:       strings.TrimSpace(r.FormValue("areas_map_scale")),
		AreasOverlayOpacity: strings.TrimSpace(r.FormValue("areas_overlay_opacity")),

		ServiceImagePreset:     strings.TrimSpace(r.FormValue("service_image_preset")),
		ServiceImageBrightness: strings.TrimSpace(r.FormValue("service_image_brightness")),
		ServiceImageContrast:   strings.TrimSpace(r.FormValue("service_image_contrast")),
		ServiceImageSaturation: strings.TrimSpace(r.FormValue("service_image_saturation")),
		ServiceImageBlur:       strings.TrimSpace(r.FormValue("service_image_blur")),
		ServiceImageOpacity:    strings.TrimSpace(r.FormValue("service_image_opacity")),
		ServiceImageScale:      strings.TrimSpace(r.FormValue("service_image_scale")),

		HeaderCTAFI: strings.TrimSpace(r.FormValue("header_cta_fi")),
		HeaderCTAEN: strings.TrimSpace(r.FormValue("header_cta_en")),

		CoverageCities: strings.TrimSpace(r.FormValue("coverage_cities")),

		HeroBadgeFI:  strings.TrimSpace(r.FormValue("hero_badge_fi")),
		HeroBadgeEN:  strings.TrimSpace(r.FormValue("hero_badge_en")),
		HeroPointsFI: strings.TrimSpace(r.FormValue("hero_points_fi")),
		HeroPointsEN: strings.TrimSpace(r.FormValue("hero_points_en")),

		HeroTrustCard1TitleFI: strings.TrimSpace(r.FormValue("hero_trust_card_1_title_fi")),
		HeroTrustCard1TextFI:  strings.TrimSpace(r.FormValue("hero_trust_card_1_text_fi")),
		HeroTrustCard2TitleFI: strings.TrimSpace(r.FormValue("hero_trust_card_2_title_fi")),
		HeroTrustCard2TextFI:  strings.TrimSpace(r.FormValue("hero_trust_card_2_text_fi")),
		HeroTrustCard3TitleFI: strings.TrimSpace(r.FormValue("hero_trust_card_3_title_fi")),
		HeroTrustCard3TextFI:  strings.TrimSpace(r.FormValue("hero_trust_card_3_text_fi")),
		HeroTrustCard1TitleEN: strings.TrimSpace(r.FormValue("hero_trust_card_1_title_en")),
		HeroTrustCard1TextEN:  strings.TrimSpace(r.FormValue("hero_trust_card_1_text_en")),
		HeroTrustCard2TitleEN: strings.TrimSpace(r.FormValue("hero_trust_card_2_title_en")),
		HeroTrustCard2TextEN:  strings.TrimSpace(r.FormValue("hero_trust_card_2_text_en")),
		HeroTrustCard3TitleEN: strings.TrimSpace(r.FormValue("hero_trust_card_3_title_en")),
		HeroTrustCard3TextEN:  strings.TrimSpace(r.FormValue("hero_trust_card_3_text_en")),

		TrustKickerFI: strings.TrimSpace(r.FormValue("trust_kicker_fi")),
		TrustKickerEN: strings.TrimSpace(r.FormValue("trust_kicker_en")),
		TrustTitleFI:  strings.TrimSpace(r.FormValue("trust_title_fi")),
		TrustTitleEN:  strings.TrimSpace(r.FormValue("trust_title_en")),
		TrustTextFI:   strings.TrimSpace(r.FormValue("trust_text_fi")),
		TrustTextEN:   strings.TrimSpace(r.FormValue("trust_text_en")),
		TrustItemsFI:  strings.TrimSpace(r.FormValue("trust_items_fi")),
		TrustItemsEN:  strings.TrimSpace(r.FormValue("trust_items_en")),

		CapabilityKickerFI:             strings.TrimSpace(r.FormValue("capability_kicker_fi")),
		CapabilityKickerEN:             strings.TrimSpace(r.FormValue("capability_kicker_en")),
		CapabilityTitleFI:              strings.TrimSpace(r.FormValue("capability_title_fi")),
		CapabilityTitleEN:              strings.TrimSpace(r.FormValue("capability_title_en")),
		CapabilityTextFI:               strings.TrimSpace(r.FormValue("capability_text_fi")),
		CapabilityTextEN:               strings.TrimSpace(r.FormValue("capability_text_en")),
		CapabilityCanDoTitleFI:         strings.TrimSpace(r.FormValue("capability_can_do_title_fi")),
		CapabilityCanDoTitleEN:         strings.TrimSpace(r.FormValue("capability_can_do_title_en")),
		CapabilityCanDoItemsFI:         strings.TrimSpace(r.FormValue("capability_can_do_items_fi")),
		CapabilityCanDoItemsEN:         strings.TrimSpace(r.FormValue("capability_can_do_items_en")),
		CapabilityNeedsWorkshopTitleFI: strings.TrimSpace(r.FormValue("capability_needs_workshop_title_fi")),
		CapabilityNeedsWorkshopTitleEN: strings.TrimSpace(r.FormValue("capability_needs_workshop_title_en")),
		CapabilityNeedsWorkshopItemsFI: strings.TrimSpace(r.FormValue("capability_needs_workshop_items_fi")),
		CapabilityNeedsWorkshopItemsEN: strings.TrimSpace(r.FormValue("capability_needs_workshop_items_en")),

		HeroTitleFI:    strings.TrimSpace(r.FormValue("hero_title_fi")),
		HeroTitleEN:    strings.TrimSpace(r.FormValue("hero_title_en")),
		HeroSubtitleFI: strings.TrimSpace(r.FormValue("hero_subtitle_fi")),
		HeroSubtitleEN: strings.TrimSpace(r.FormValue("hero_subtitle_en")),

		PrimaryCTAFI:   strings.TrimSpace(r.FormValue("primary_cta_fi")),
		PrimaryCTAEN:   strings.TrimSpace(r.FormValue("primary_cta_en")),
		SecondaryCTAFI: strings.TrimSpace(r.FormValue("secondary_cta_fi")),
		SecondaryCTAEN: strings.TrimSpace(r.FormValue("secondary_cta_en")),

		ServicesKickerFI: strings.TrimSpace(r.FormValue("services_kicker_fi")),
		ServicesKickerEN: strings.TrimSpace(r.FormValue("services_kicker_en")),
		ServicesTitleFI:  strings.TrimSpace(r.FormValue("services_title_fi")),
		ServicesTitleEN:  strings.TrimSpace(r.FormValue("services_title_en")),
		ServicesTextFI:   strings.TrimSpace(r.FormValue("services_text_fi")),
		ServicesTextEN:   strings.TrimSpace(r.FormValue("services_text_en")),

		ProcessKickerFI: strings.TrimSpace(r.FormValue("process_kicker_fi")),
		ProcessKickerEN: strings.TrimSpace(r.FormValue("process_kicker_en")),
		ProcessTitleFI:  strings.TrimSpace(r.FormValue("process_title_fi")),
		ProcessTitleEN:  strings.TrimSpace(r.FormValue("process_title_en")),
		ProcessTextFI:   strings.TrimSpace(r.FormValue("process_text_fi")),
		ProcessTextEN:   strings.TrimSpace(r.FormValue("process_text_en")),

		FinalCTAKickerFI: strings.TrimSpace(r.FormValue("final_cta_kicker_fi")),
		FinalCTAKickerEN: strings.TrimSpace(r.FormValue("final_cta_kicker_en")),
		FinalCTATitleFI:  strings.TrimSpace(r.FormValue("final_cta_title_fi")),
		FinalCTATitleEN:  strings.TrimSpace(r.FormValue("final_cta_title_en")),
		FinalCTATextFI:   strings.TrimSpace(r.FormValue("final_cta_text_fi")),
		FinalCTATextEN:   strings.TrimSpace(r.FormValue("final_cta_text_en")),
		FinalCTAButtonFI: strings.TrimSpace(r.FormValue("final_cta_button_fi")),
		FinalCTAButtonEN: strings.TrimSpace(r.FormValue("final_cta_button_en")),

		FormTitleFI:                strings.TrimSpace(r.FormValue("form_title_fi")),
		FormIntroFI:                strings.TrimSpace(r.FormValue("form_intro_fi")),
		FormLabelNameFI:            strings.TrimSpace(r.FormValue("form_label_name_fi")),
		FormLabelPhoneFI:           strings.TrimSpace(r.FormValue("form_label_phone_fi")),
		FormLabelEmailFI:           strings.TrimSpace(r.FormValue("form_label_email_fi")),
		FormLabelAddressFI:         strings.TrimSpace(r.FormValue("form_label_address_fi")),
		FormLabelCarFI:             strings.TrimSpace(r.FormValue("form_label_car_fi")),
		FormLabelPhotosFI:          strings.TrimSpace(r.FormValue("form_label_photos_fi")),
		FormLabelMessageFI:         strings.TrimSpace(r.FormValue("form_label_message_fi")),
		FormPlaceholderNameFI:      strings.TrimSpace(r.FormValue("form_placeholder_name_fi")),
		FormPlaceholderPhoneFI:     strings.TrimSpace(r.FormValue("form_placeholder_phone_fi")),
		FormPlaceholderEmailFI:     strings.TrimSpace(r.FormValue("form_placeholder_email_fi")),
		FormPlaceholderAddressFI:   strings.TrimSpace(r.FormValue("form_placeholder_address_fi")),
		FormPlaceholderCarFI:       strings.TrimSpace(r.FormValue("form_placeholder_car_fi")),
		FormPlaceholderMessageFI:   strings.TrimSpace(r.FormValue("form_placeholder_message_fi")),
		FormUseLocationButtonFI:    strings.TrimSpace(r.FormValue("form_use_location_button_fi")),
		FormPickMapButtonFI:        strings.TrimSpace(r.FormValue("form_pick_map_button_fi")),
		FormUploadButtonTextFI:     strings.TrimSpace(r.FormValue("form_upload_button_text_fi")),
		FormPhotosHintFI:           strings.TrimSpace(r.FormValue("form_photos_hint_fi")),
		FormPhoneHelpFI:            strings.TrimSpace(r.FormValue("form_phone_help_fi")),
		FormAddressHelpFI:          strings.TrimSpace(r.FormValue("form_address_help_fi")),
		FormCarHelpFI:              strings.TrimSpace(r.FormValue("form_car_help_fi")),
		FormSubmitButtonFI:         strings.TrimSpace(r.FormValue("form_submit_button_fi")),
		FormSubmitLoadingTextFI:    strings.TrimSpace(r.FormValue("form_submit_loading_text_fi")),
		FormSuccessMessageFI:       strings.TrimSpace(r.FormValue("form_success_message_fi")),
		FormErrorMessageFI:         strings.TrimSpace(r.FormValue("form_error_message_fi")),
		FormNameRequiredFI:         strings.TrimSpace(r.FormValue("form_name_required_fi")),
		FormContactRequiredFI:      strings.TrimSpace(r.FormValue("form_contact_required_fi")),
		FormPhoneInvalidFI:         strings.TrimSpace(r.FormValue("form_phone_invalid_fi")),
		FormEmailInvalidFI:         strings.TrimSpace(r.FormValue("form_email_invalid_fi")),
		FormRegistrationInvalidFI:  strings.TrimSpace(r.FormValue("form_registration_invalid_fi")),
		FormPhotosTooManyFI:        strings.TrimSpace(r.FormValue("form_photos_too_many_fi")),
		FormPhotosTypeFI:           strings.TrimSpace(r.FormValue("form_photos_type_fi")),
		FormPhotosSizeFI:           strings.TrimSpace(r.FormValue("form_photos_size_fi")),
		FormGeoUnsupportedFI:       strings.TrimSpace(r.FormValue("form_geo_unsupported_fi")),
		FormGeoLoadingFI:           strings.TrimSpace(r.FormValue("form_geo_loading_fi")),
		FormGeoAddedFI:             strings.TrimSpace(r.FormValue("form_geo_added_fi")),
		FormGeoFailedFI:            strings.TrimSpace(r.FormValue("form_geo_failed_fi")),
		FormMapPickFirstFI:         strings.TrimSpace(r.FormValue("form_map_pick_first_fi")),
		FormMapPointAddedFI:        strings.TrimSpace(r.FormValue("form_map_point_added_fi")),
		FormReverseGeocodeFailedFI: strings.TrimSpace(r.FormValue("form_reverse_geocode_failed_fi")),
		FormNetworkErrorFI:         strings.TrimSpace(r.FormValue("form_network_error_fi")),

		FormTitleEN:                strings.TrimSpace(r.FormValue("form_title_en")),
		FormIntroEN:                strings.TrimSpace(r.FormValue("form_intro_en")),
		FormLabelNameEN:            strings.TrimSpace(r.FormValue("form_label_name_en")),
		FormLabelPhoneEN:           strings.TrimSpace(r.FormValue("form_label_phone_en")),
		FormLabelEmailEN:           strings.TrimSpace(r.FormValue("form_label_email_en")),
		FormLabelAddressEN:         strings.TrimSpace(r.FormValue("form_label_address_en")),
		FormLabelCarEN:             strings.TrimSpace(r.FormValue("form_label_car_en")),
		FormLabelPhotosEN:          strings.TrimSpace(r.FormValue("form_label_photos_en")),
		FormLabelMessageEN:         strings.TrimSpace(r.FormValue("form_label_message_en")),
		FormPlaceholderNameEN:      strings.TrimSpace(r.FormValue("form_placeholder_name_en")),
		FormPlaceholderPhoneEN:     strings.TrimSpace(r.FormValue("form_placeholder_phone_en")),
		FormPlaceholderEmailEN:     strings.TrimSpace(r.FormValue("form_placeholder_email_en")),
		FormPlaceholderAddressEN:   strings.TrimSpace(r.FormValue("form_placeholder_address_en")),
		FormPlaceholderCarEN:       strings.TrimSpace(r.FormValue("form_placeholder_car_en")),
		FormPlaceholderMessageEN:   strings.TrimSpace(r.FormValue("form_placeholder_message_en")),
		FormUseLocationButtonEN:    strings.TrimSpace(r.FormValue("form_use_location_button_en")),
		FormPickMapButtonEN:        strings.TrimSpace(r.FormValue("form_pick_map_button_en")),
		FormUploadButtonTextEN:     strings.TrimSpace(r.FormValue("form_upload_button_text_en")),
		FormPhotosHintEN:           strings.TrimSpace(r.FormValue("form_photos_hint_en")),
		FormPhoneHelpEN:            strings.TrimSpace(r.FormValue("form_phone_help_en")),
		FormAddressHelpEN:          strings.TrimSpace(r.FormValue("form_address_help_en")),
		FormCarHelpEN:              strings.TrimSpace(r.FormValue("form_car_help_en")),
		FormSubmitButtonEN:         strings.TrimSpace(r.FormValue("form_submit_button_en")),
		FormSubmitLoadingTextEN:    strings.TrimSpace(r.FormValue("form_submit_loading_text_en")),
		FormSuccessMessageEN:       strings.TrimSpace(r.FormValue("form_success_message_en")),
		FormErrorMessageEN:         strings.TrimSpace(r.FormValue("form_error_message_en")),
		FormNameRequiredEN:         strings.TrimSpace(r.FormValue("form_name_required_en")),
		FormContactRequiredEN:      strings.TrimSpace(r.FormValue("form_contact_required_en")),
		FormPhoneInvalidEN:         strings.TrimSpace(r.FormValue("form_phone_invalid_en")),
		FormEmailInvalidEN:         strings.TrimSpace(r.FormValue("form_email_invalid_en")),
		FormRegistrationInvalidEN:  strings.TrimSpace(r.FormValue("form_registration_invalid_en")),
		FormPhotosTooManyEN:        strings.TrimSpace(r.FormValue("form_photos_too_many_en")),
		FormPhotosTypeEN:           strings.TrimSpace(r.FormValue("form_photos_type_en")),
		FormPhotosSizeEN:           strings.TrimSpace(r.FormValue("form_photos_size_en")),
		FormGeoUnsupportedEN:       strings.TrimSpace(r.FormValue("form_geo_unsupported_en")),
		FormGeoLoadingEN:           strings.TrimSpace(r.FormValue("form_geo_loading_en")),
		FormGeoAddedEN:             strings.TrimSpace(r.FormValue("form_geo_added_en")),
		FormGeoFailedEN:            strings.TrimSpace(r.FormValue("form_geo_failed_en")),
		FormMapPickFirstEN:         strings.TrimSpace(r.FormValue("form_map_pick_first_en")),
		FormMapPointAddedEN:        strings.TrimSpace(r.FormValue("form_map_point_added_en")),
		FormReverseGeocodeFailedEN: strings.TrimSpace(r.FormValue("form_reverse_geocode_failed_en")),
		FormNetworkErrorEN:         strings.TrimSpace(r.FormValue("form_network_error_en")),

		SectionVisibilityConfigured: true,
		StatsEnabled:                strings.TrimSpace(r.FormValue("stats_enabled")) == "1",
		ServicesEnabled:             strings.TrimSpace(r.FormValue("services_enabled")) == "1",
		ProcessEnabled:              strings.TrimSpace(r.FormValue("process_enabled")) == "1",
		CasesEnabled:                strings.TrimSpace(r.FormValue("cases_enabled")) == "1",
		CapabilityEnabled:           strings.TrimSpace(r.FormValue("capability_enabled")) == "1",
		AreasEnabled:                strings.TrimSpace(r.FormValue("areas_enabled")) == "1",
		TestimonialsEnabled:         strings.TrimSpace(r.FormValue("testimonials_enabled")) == "1",
		FAQEnabled:                  strings.TrimSpace(r.FormValue("faq_enabled")) == "1",
		FinalCTAEnabled:             strings.TrimSpace(r.FormValue("final_cta_enabled")) == "1",

		PromoEnabled:  strings.TrimSpace(r.FormValue("promo_enabled")) == "1",
		PromoBadgeFI:  strings.TrimSpace(r.FormValue("promo_badge_fi")),
		PromoBadgeEN:  strings.TrimSpace(r.FormValue("promo_badge_en")),
		PromoTitleFI:  strings.TrimSpace(r.FormValue("promo_title_fi")),
		PromoTitleEN:  strings.TrimSpace(r.FormValue("promo_title_en")),
		PromoTextFI:   strings.TrimSpace(r.FormValue("promo_text_fi")),
		PromoTextEN:   strings.TrimSpace(r.FormValue("promo_text_en")),
		PromoButtonFI: strings.TrimSpace(r.FormValue("promo_button_fi")),
		PromoButtonEN: strings.TrimSpace(r.FormValue("promo_button_en")),
		PromoURL:      strings.TrimSpace(r.FormValue("promo_url")),

		LandingSEOTitleFI:       strings.TrimSpace(r.FormValue("landing_seo_title_fi")),
		LandingSEOTitleEN:       strings.TrimSpace(r.FormValue("landing_seo_title_en")),
		LandingSEODescriptionFI: strings.TrimSpace(r.FormValue("landing_seo_description_fi")),
		LandingSEODescriptionEN: strings.TrimSpace(r.FormValue("landing_seo_description_en")),

		ThankYouSEOTitleFI:       strings.TrimSpace(r.FormValue("thankyou_seo_title_fi")),
		ThankYouSEOTitleEN:       strings.TrimSpace(r.FormValue("thankyou_seo_title_en")),
		ThankYouSEODescriptionFI: strings.TrimSpace(r.FormValue("thankyou_seo_description_fi")),
		ThankYouSEODescriptionEN: strings.TrimSpace(r.FormValue("thankyou_seo_description_en")),
		ThankYouTitleFI:          strings.TrimSpace(r.FormValue("thankyou_title_fi")),
		ThankYouTitleEN:          strings.TrimSpace(r.FormValue("thankyou_title_en")),
		ThankYouTextFI:           strings.TrimSpace(r.FormValue("thankyou_text_fi")),
		ThankYouTextEN:           strings.TrimSpace(r.FormValue("thankyou_text_en")),
		ThankYouButtonFI:         strings.TrimSpace(r.FormValue("thankyou_button_fi")),
		ThankYouButtonEN:         strings.TrimSpace(r.FormValue("thankyou_button_en")),
		ThankYouButtonURLFI:      strings.TrimSpace(r.FormValue("thankyou_button_url_fi")),
		ThankYouButtonURLEN:      strings.TrimSpace(r.FormValue("thankyou_button_url_en")),

		PrivacySEOTitleFI:       strings.TrimSpace(r.FormValue("privacy_seo_title_fi")),
		PrivacySEOTitleEN:       strings.TrimSpace(r.FormValue("privacy_seo_title_en")),
		PrivacySEODescriptionFI: strings.TrimSpace(r.FormValue("privacy_seo_description_fi")),
		PrivacySEODescriptionEN: strings.TrimSpace(r.FormValue("privacy_seo_description_en")),
		PrivacyTitleFI:          strings.TrimSpace(r.FormValue("privacy_title_fi")),
		PrivacyTitleEN:          strings.TrimSpace(r.FormValue("privacy_title_en")),
		PrivacyIntroFI:          strings.TrimSpace(r.FormValue("privacy_intro_fi")),
		PrivacyIntroEN:          strings.TrimSpace(r.FormValue("privacy_intro_en")),
		PrivacySection1TitleFI:  strings.TrimSpace(r.FormValue("privacy_section_1_title_fi")),
		PrivacySection1TitleEN:  strings.TrimSpace(r.FormValue("privacy_section_1_title_en")),
		PrivacySection1TextFI:   strings.TrimSpace(r.FormValue("privacy_section_1_text_fi")),
		PrivacySection1TextEN:   strings.TrimSpace(r.FormValue("privacy_section_1_text_en")),
		PrivacySection2TitleFI:  strings.TrimSpace(r.FormValue("privacy_section_2_title_fi")),
		PrivacySection2TitleEN:  strings.TrimSpace(r.FormValue("privacy_section_2_title_en")),
		PrivacySection2TextFI:   strings.TrimSpace(r.FormValue("privacy_section_2_text_fi")),
		PrivacySection2TextEN:   strings.TrimSpace(r.FormValue("privacy_section_2_text_en")),
		PrivacySection3TitleFI:  strings.TrimSpace(r.FormValue("privacy_section_3_title_fi")),
		PrivacySection3TitleEN:  strings.TrimSpace(r.FormValue("privacy_section_3_title_en")),
		PrivacySection3TextFI:   strings.TrimSpace(r.FormValue("privacy_section_3_text_fi")),
		PrivacySection3TextEN:   strings.TrimSpace(r.FormValue("privacy_section_3_text_en")),
		PrivacySection4TitleFI:  strings.TrimSpace(r.FormValue("privacy_section_4_title_fi")),
		PrivacySection4TitleEN:  strings.TrimSpace(r.FormValue("privacy_section_4_title_en")),
		PrivacySection4TextFI:   strings.TrimSpace(r.FormValue("privacy_section_4_text_fi")),
		PrivacySection4TextEN:   strings.TrimSpace(r.FormValue("privacy_section_4_text_en")),
		PrivacySection5TitleFI:  strings.TrimSpace(r.FormValue("privacy_section_5_title_fi")),
		PrivacySection5TitleEN:  strings.TrimSpace(r.FormValue("privacy_section_5_title_en")),
		PrivacySection5TextFI:   strings.TrimSpace(r.FormValue("privacy_section_5_text_fi")),
		PrivacySection5TextEN:   strings.TrimSpace(r.FormValue("privacy_section_5_text_en")),

		LandingRobotsMeta:  strings.TrimSpace(r.FormValue("landing_robots_meta")),
		ThankYouRobotsMeta: strings.TrimSpace(r.FormValue("thankyou_robots_meta")),
		PrivacyRobotsMeta:  strings.TrimSpace(r.FormValue("privacy_robots_meta")),
		RobotsTXT:          strings.TrimSpace(r.FormValue("robots_txt")),
	}
	// An already-open form from an older release has no event controls.
	if r.FormValue("analytics_events_present") != "1" {
		current, err := loadSiteSettings(r.Context(), h.cfg)
		if err != nil {
			http.Error(w, "Не удалось проверить настройки аналитики. Обновите страницу.", http.StatusServiceUnavailable)
			return
		}
		settings.AnalyticsEventsConfigured = current.AnalyticsEventsConfigured
		settings.AnalyticsDisabledEvents = current.AnalyticsDisabledEvents
		settings.AnalyticsDisabled = current.AnalyticsDisabled
	}

	if err := saveSiteSettings(r.Context(), h.cfg, settings); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		code := http.StatusServiceUnavailable
		if errors.Is(err, errSiteSettingsConflict) {
			code = http.StatusConflict
		}
		w.WriteHeader(code)
		settings = normalizeAdminSettingsForTemplate(settings)

		if renderErr := h.renderAdmin(w, r, AdminPageData{
			PageTitle: "Кабинет Liikkuva",
			Error:     err.Error(),
			Settings:  settings,
			ActiveTab: activeTab,
		}); renderErr != nil {
			http.Error(w, "failed to render admin save error: "+renderErr.Error(), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, "/admin?saved=1&tab="+url.QueryEscape(activeTab), http.StatusSeeOther)
}

func (h *AdminHandler) RetryLead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	leadID := strings.TrimSpace(r.FormValue("lead_id"))
	if leadID == "" {
		http.Redirect(w, r, "/admin?tab=leads&error="+url.QueryEscape("lead_id is required"), http.StatusSeeOther)
		return
	}

	if err := enqueueLeadProcessingTask(r.Context(), h.cfg, leadID); err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("retry failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	if err := markLeadQueued(r.Context(), h.cfg, leadID); err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("lead re-queued, but status update failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&retry=1", http.StatusSeeOther)
}

func (h *AdminHandler) UpdateClientStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	leadID := strings.TrimSpace(r.FormValue("lead_id"))
	clientStatus := strings.ToLower(strings.TrimSpace(r.FormValue("client_status")))
	if leadID == "" || !isAllowedClientStatus(clientStatus) {
		http.Redirect(w, r, "/admin?tab=leads&error="+url.QueryEscape("select a valid client status"), http.StatusSeeOther)
		return
	}

	if err := markClientStatus(r.Context(), h.cfg, leadID, clientStatus); err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("status update failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&status_updated=1", http.StatusSeeOther)
}

func (h *AdminHandler) SendConfirmationEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	leadID := strings.TrimSpace(r.FormValue("lead_id"))
	if leadID == "" {
		http.Redirect(w, r, "/admin?tab=leads&error="+url.QueryEscape("lead_id is required"), http.StatusSeeOther)
		return
	}

	record, err := getLeadRecord(r.Context(), h.cfg, leadID)
	if err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("load lead failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	if strings.TrimSpace(record.Lead.Email) == "" {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("client email is empty"), http.StatusSeeOther)
		return
	}

	if err := sendLeadConfirmationEmail(h.cfg, record.Lead); err != nil {
		_ = markConfirmationEmailFailed(r.Context(), h.cfg, leadID, err.Error())
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("confirmation email send failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	if err := markConfirmationEmailSent(r.Context(), h.cfg, leadID); err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("confirmation email sent, but status update failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&confirmation_email=1", http.StatusSeeOther)
}

func (h *AdminHandler) SendReadyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	leadID := strings.TrimSpace(r.FormValue("lead_id"))
	if leadID == "" {
		http.Redirect(w, r, "/admin?tab=leads&error="+url.QueryEscape("lead_id is required"), http.StatusSeeOther)
		return
	}

	record, err := getLeadRecord(r.Context(), h.cfg, leadID)
	if err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("load lead failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	if strings.TrimSpace(record.Lead.Email) == "" {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("client email is empty"), http.StatusSeeOther)
		return
	}

	if err := sendReadyForPickupEmail(h.cfg, record.Lead); err != nil {
		_ = markReadyEmailFailed(r.Context(), h.cfg, leadID, err.Error())
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("email send failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	if err := markReadyEmailSent(r.Context(), h.cfg, leadID); err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("email sent, but status update failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&ready_email=1", http.StatusSeeOther)
}

func isAdminAuthenticated(cfg Config, r *http.Request) bool {
	return authenticateAdmin(cfg, r)
}

func normalizeAdminStatusFilter(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "queued", "processing", "done", "failed":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "all"
	}
}

func normalizeClientStatusFilter(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "all" || isAllowedClientStatus(value) {
		return value
	}
	return "all"
}

func isAllowedClientStatus(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "new", "contacted", "scheduled", "in_progress", "waiting_parts", "ready_for_pickup", "completed", "cancelled":
		return true
	default:
		return false
	}
}

func filterLeadRecordsByClientStatus(records []LeadRecord, statusFilter string) []LeadRecord {
	if statusFilter == "" || statusFilter == "all" {
		return records
	}
	filtered := make([]LeadRecord, 0, len(records))
	for _, record := range records {
		if strings.EqualFold(adminClientStatus(record.ClientStatus), statusFilter) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func normalizeAdminTab(value string, hasSelectedLead bool) string {
	value = strings.ToLower(strings.TrimSpace(value))

	switch value {
	case "overview", "help", "general", "assets", "analytics", "emails", "visual", "landing-fi", "landing-en", "form-fi", "form-en", "promo", "testimonials", "services", "cases", "thankyou", "privacy", "seo", "leads", "reviews":
		return value
	}

	if hasSelectedLead {
		return "leads"
	}

	return "overview"
}

func countAdminMissingAssets(settings SiteSettings) int {
	missing := 0
	if strings.TrimSpace(settings.LogoURL) == "" {
		missing++
	}
	if strings.TrimSpace(settings.HeroBackgroundURL) == "" {
		missing++
	}
	if strings.TrimSpace(settings.HeroPersonDesktopURL) == "" {
		missing++
	}
	if strings.TrimSpace(settings.AreasMapImageURL) == "" {
		missing++
	}
	serviceImages := []string{settings.ServicesImage1URL, settings.ServicesImage2URL, settings.ServicesImage3URL, settings.ServicesImage4URL, settings.ServicesImage5URL, settings.ServicesImage6URL}
	hasServiceImage := false
	for _, item := range serviceImages {
		if strings.TrimSpace(item) != "" {
			hasServiceImage = true
			break
		}
	}
	if !hasServiceImage {
		missing++
	}
	return missing
}

func buildAdminLeadList(records []LeadRecord) []AdminLeadListItem {
	items := make([]AdminLeadListItem, 0, len(records))
	for _, record := range records {
		items = append(items, AdminLeadListItem{
			ID:               record.ID,
			CreatedAt:        formatAdminTime(record.CreatedAt),
			Name:             adminValue(record.Lead.Name),
			Phone:            adminValue(record.Lead.Phone),
			Email:            adminValue(record.Lead.Email),
			Car:              adminValue(record.Lead.Car),
			MessagePreview:   adminExcerpt(record.Lead.Message, 110),
			Language:         adminValue(record.Lead.Language),
			ProcessingStatus: adminValue(record.ProcessingStatus),
			TelegramStatus:   adminValue(record.TelegramStatus),
			SheetsStatus:     adminValue(record.SheetsStatus),
			ClientStatus:     adminClientStatus(record.ClientStatus),
			ReadyEmailStatus: adminValue(record.ReadyEmailStatus),
			LastError:        strings.TrimSpace(record.LastError),
		})
	}
	return items
}

func adminExcerpt(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "—"
	}
	runes := []rune(value)
	if maxRunes <= 0 || len(runes) <= maxRunes {
		return value
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "…"
}

func adminClientStatus(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if !isAllowedClientStatus(value) {
		return "new"
	}
	return value
}

func buildAdminLeadDetails(record LeadRecord) AdminLeadDetails {
	return AdminLeadDetails{
		ID:                      record.ID,
		CreatedAt:               formatAdminTime(record.CreatedAt),
		UpdatedAt:               formatAdminTime(record.UpdatedAt),
		TaskEnqueuedAt:          formatAdminTime(record.TaskEnqueuedAt),
		ProcessedAt:             formatAdminTime(record.ProcessedAt),
		Name:                    adminValue(record.Lead.Name),
		Phone:                   adminValue(record.Lead.Phone),
		Email:                   adminValue(record.Lead.Email),
		Address:                 adminValue(record.Lead.Address),
		AddressHint:             adminValue(record.Lead.AddressHint),
		LocationSource:          adminValue(record.Lead.LocationSource),
		Latitude:                adminValue(record.Lead.Latitude),
		Longitude:               adminValue(record.Lead.Longitude),
		MapsURL:                 adminValue(record.Lead.MapsURL),
		Car:                     adminValue(record.Lead.Car),
		Message:                 adminValue(record.Lead.Message),
		VisitPriceEUR:           record.Lead.VisitPriceEUR,
		VisitPriceAccepted:      record.Lead.VisitPriceAccepted,
		Language:                adminValue(record.Lead.Language),
		Source:                  adminValue(record.Lead.Source),
		PageURL:                 adminValue(record.Lead.PageURL),
		Referer:                 adminValue(record.Lead.Referer),
		UTMSource:               adminValue(record.Lead.UTMSource),
		UTMMedium:               adminValue(record.Lead.UTMMedium),
		UTMCampaign:             adminValue(record.Lead.UTMCampaign),
		UTMTerm:                 adminValue(record.Lead.UTMTerm),
		UTMContent:              adminValue(record.Lead.UTMContent),
		IP:                      adminValue(record.Lead.IP),
		UserAgent:               adminValue(record.Lead.UserAgent),
		ProcessingStatus:        adminValue(record.ProcessingStatus),
		TelegramStatus:          adminValue(record.TelegramStatus),
		SheetsStatus:            adminValue(record.SheetsStatus),
		ClientStatus:            adminClientStatus(record.ClientStatus),
		ReadyEmailStatus:        adminValue(record.ReadyEmailStatus),
		ReadyEmailError:         adminValue(record.ReadyEmailError),
		ReadyEmailSentAt:        formatAdminTime(record.ReadyEmailSentAt),
		ConfirmationEmailStatus: adminValue(record.ConfirmationEmailStatus),
		ConfirmationEmailError:  adminValue(record.ConfirmationEmailError),
		ConfirmationEmailSentAt: formatAdminTime(record.ConfirmationEmailSentAt),
		ReviewInviteEmailStatus: adminValue(record.ReviewInviteEmailStatus),
		ReviewInviteEmailError:  adminValue(record.ReviewInviteEmailError),
		ReviewInviteEmailSentAt: formatAdminTime(record.ReviewInviteEmailSentAt),
		LastError:               adminValue(record.LastError),
		PhotoURLs:               record.Lead.PhotoURLs,
	}
}

func buildAdminReviewList(records []ReviewRecord) []AdminReviewListItem {
	items := make([]AdminReviewListItem, 0, len(records))
	for _, record := range records {
		items = append(items, AdminReviewListItem{
			ID:        record.ID,
			LeadID:    record.LeadID,
			CreatedAt: formatAdminTime(record.CreatedAt),
			Name:      adminValue(record.Name),
			Rating:    normalizeTestimonialRatingValue(record.Rating),
			Stars:     testimonialStars(record.Rating),
			Language:  adminValue(record.Language),
			Message:   adminValue(record.Message),
			Status:    adminValue(record.Status),
		})
	}
	return items
}

func adminValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "—"
	}
	return value
}

func formatAdminTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return liikkuvaTimestamp(t)
}

func valueAt(items []string, index int) string {
	if index < 0 || index >= len(items) {
		return ""
	}
	return strings.TrimSpace(items[index])
}

func padStrings(items []string, size int) []string {
	if size <= 0 {
		return items
	}

	out := make([]string, size)
	copy(out, items)
	return out
}

func (h *AdminHandler) SendCustomEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxEmailAttachmentBytes+(1<<20))
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			http.Error(w, "Слишком большой размер письма", 400)
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
	}
	attachments, attachmentErr := readEmailAttachments(r)
	if attachmentErr != nil {
		http.Error(w, attachmentErr.Error(), 400)
		return
	}
	leadID := strings.TrimSpace(r.FormValue("lead_id"))
	subject := strings.TrimSpace(r.FormValue("custom_email_subject"))
	message := strings.TrimSpace(r.FormValue("custom_email_message"))

	if leadID == "" {
		http.Redirect(w, r, "/admin?tab=leads&error="+url.QueryEscape("lead_id is required"), http.StatusSeeOther)
		return
	}
	if subject == "" {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("email subject is required"), http.StatusSeeOther)
		return
	}
	if message == "" {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("email message is required"), http.StatusSeeOther)
		return
	}

	record, err := getLeadRecord(r.Context(), h.cfg, leadID)
	if err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("load lead failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	if strings.TrimSpace(record.Lead.Email) == "" {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("client email is empty"), http.StatusSeeOther)
		return
	}

	if err := sendCustomClientEmail(h.cfg, record.Lead, subject, message, attachments...); err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("custom email send failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&saved=1", http.StatusSeeOther)
}

func (h *AdminHandler) SendReviewEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	leadID := strings.TrimSpace(r.FormValue("lead_id"))
	if leadID == "" {
		http.Redirect(w, r, "/admin?tab=leads&error="+url.QueryEscape("lead_id is required"), http.StatusSeeOther)
		return
	}

	record, err := getLeadRecord(r.Context(), h.cfg, leadID)
	if err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("load lead failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	if strings.TrimSpace(record.Lead.Email) == "" {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("client email is empty"), http.StatusSeeOther)
		return
	}

	reviewURL, err := buildReviewURLWithBase(h.cfg, publicBaseURLForRequest(h.cfg, r), leadID, record.Lead)
	if err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("build review link failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	if err := sendReviewInvitationEmail(h.cfg, record.Lead, reviewURL); err != nil {
		_ = markReviewInviteEmailFailed(r.Context(), h.cfg, leadID, err.Error())
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("review email send failed: "+err.Error()), http.StatusSeeOther)
		return
	}
	if err := markReviewInviteEmailSent(r.Context(), h.cfg, leadID); err != nil {
		http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&error="+url.QueryEscape("review email sent, but status update failed: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin?tab=leads&lead_id="+url.QueryEscape(leadID)+"&review_email=1", http.StatusSeeOther)
}

func (h *AdminHandler) ModerateReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin?tab=reviews", http.StatusSeeOther)
		return
	}
	if !isAdminAuthenticated(h.cfg, r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	reviewID := strings.TrimSpace(r.FormValue("review_id"))
	action := strings.TrimSpace(r.FormValue("action"))
	if reviewID == "" {
		http.Redirect(w, r, "/admin?tab=reviews&error="+url.QueryEscape("review_id is required"), http.StatusSeeOther)
		return
	}
	if err := moderateReviewRecord(r.Context(), h.cfg, reviewID, action); err != nil {
		http.Redirect(w, r, "/admin?tab=reviews&error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin?tab=reviews&review_status=pending&saved=1", http.StatusSeeOther)
}

func padAdminCasePoints(items []AdminCaseItem) []AdminCaseItem {
	out := make([]AdminCaseItem, len(items))
	copy(out, items)

	for i := range out {
		out[i].PointsFI = padStrings(out[i].PointsFI, 4)
		out[i].PointsEN = padStrings(out[i].PointsEN, 4)
	}

	return out
}

func adminStatusLabel(value string) string {
	labels := map[string]string{
		"new": "Новая", "contacted": "Связались", "scheduled": "Выезд назначен", "in_progress": "В работе", "waiting_parts": "Ожидаем запчасти", "ready_for_pickup": "Работа готова", "completed": "Завершено", "cancelled": "Отменено", "queued": "В очереди", "processing": "Обрабатывается", "done": "Обработано", "failed": "Ошибка", "pending": "Ожидает", "sent": "Отправлено", "not_sent": "Не отправлено", "not_applicable": "Не требуется", "success": "Доставлено", "approved": "Одобрено", "rejected": "Отклонено", "error": "Ошибка", "skipped": "Пропущено", "disabled": "Выключено",
	}
	if label, ok := labels[strings.ToLower(strings.TrimSpace(value))]; ok {
		return label
	}
	return value
}
