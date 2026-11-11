package main

import "strings"

// Mirror the same public defaults in the editor; saved overrides keep priority.
func adminContentDefaults(s SiteSettings) SiteSettings {
	fi, en := getLandingContent("fi"), getLandingContent("en")
	for _, field := range []*string{&s.HeroTrustCard3TitleFI, &s.HeroTrustCard3TextFI, &s.HeroTrustCard3TitleEN, &s.HeroTrustCard3TextEN} {
		if oldLanguageClaim(*field) {
			*field = ""
		}
	}
	adminDefaultString(&s.LogoURL, fi.Header.LogoURL)
	adminDefaultString(&s.HeroPersonDesktopURL, fi.Hero.PersonDesktopURL)
	adminDefaultString(&s.HeroBackgroundURL, fi.Hero.BackgroundURL)
	adminDefaultString(&s.AreasMapImageURL, fi.Areas.MapImageURL)
	adminDefaultString(&s.HeroPersonAltFI, fi.Hero.PersonAlt)
	adminDefaultString(&s.TrustKickerFI, fi.TrustPanel.Kicker)
	adminDefaultString(&s.TrustTitleFI, fi.TrustPanel.Title)
	adminDefaultString(&s.TrustTextFI, fi.TrustPanel.Text)
	adminDefaultString(&s.CapabilityKickerFI, fi.Capability.Heading.Kicker)
	adminDefaultString(&s.CapabilityTitleFI, fi.Capability.Heading.Title)
	adminDefaultString(&s.CapabilityTextFI, fi.Capability.Heading.Text)
	adminDefaultString(&s.CapabilityCanDoTitleFI, fi.Capability.CanDo.Title)
	adminDefaultString(&s.CapabilityNeedsWorkshopTitleFI, fi.Capability.NeedsWorkshop.Title)
	adminDefaultString(&s.HeaderCTAFI, fi.Header.CTA)
	adminDefaultString(&s.HeroTitleFI, fi.Hero.Title)
	adminDefaultString(&s.HeroSubtitleFI, fi.Hero.Subtitle)
	adminDefaultString(&s.PrimaryCTAFI, fi.Hero.PrimaryCTA)
	adminDefaultString(&s.SecondaryCTAFI, fi.Hero.SecondaryCTA)
	adminDefaultString(&s.ServicesKickerFI, fi.Services.Heading.Kicker)
	adminDefaultString(&s.ServicesTitleFI, fi.Services.Heading.Title)
	adminDefaultString(&s.ServicesTextFI, fi.Services.Heading.Text)
	adminDefaultString(&s.ProcessKickerFI, fi.Process.Heading.Kicker)
	adminDefaultString(&s.ProcessTitleFI, fi.Process.Heading.Title)
	adminDefaultString(&s.ProcessTextFI, fi.Process.Heading.Text)
	adminDefaultString(&s.FinalCTAKickerFI, fi.FinalCTA.Kicker)
	adminDefaultString(&s.FinalCTATitleFI, fi.FinalCTA.Title)
	adminDefaultString(&s.FinalCTATextFI, fi.FinalCTA.Text)
	adminDefaultString(&s.FinalCTAButtonFI, fi.FinalCTA.Button)
	adminDefaultString(&s.FormTitleFI, fi.Form.Title)
	adminDefaultString(&s.FormIntroFI, fi.Form.Intro)
	adminDefaultString(&s.FormLabelNameFI, fi.Form.LabelName)
	adminDefaultString(&s.FormLabelPhoneFI, fi.Form.LabelPhone)
	adminDefaultString(&s.FormLabelEmailFI, fi.Form.LabelEmail)
	adminDefaultString(&s.FormLabelAddressFI, fi.Form.LabelAddress)
	adminDefaultString(&s.FormLabelCarFI, fi.Form.LabelCar)
	adminDefaultString(&s.FormLabelPhotosFI, fi.Form.LabelPhotos)
	adminDefaultString(&s.FormLabelMessageFI, fi.Form.LabelMessage)
	adminDefaultString(&s.FormPlaceholderNameFI, fi.Form.PlaceholderName)
	adminDefaultString(&s.FormPlaceholderPhoneFI, fi.Form.PlaceholderPhone)
	adminDefaultString(&s.FormPlaceholderEmailFI, fi.Form.PlaceholderEmail)
	adminDefaultString(&s.FormPlaceholderAddressFI, fi.Form.PlaceholderAddress)
	adminDefaultString(&s.FormPlaceholderCarFI, fi.Form.PlaceholderCar)
	adminDefaultString(&s.FormPlaceholderMessageFI, fi.Form.PlaceholderMessage)
	adminDefaultString(&s.FormUseLocationButtonFI, fi.Form.UseLocationButton)
	adminDefaultString(&s.FormPickMapButtonFI, fi.Form.PickMapButton)
	adminDefaultString(&s.FormUploadButtonTextFI, fi.Form.UploadButtonText)
	adminDefaultString(&s.FormPhotosHintFI, fi.Form.PhotosHint)
	adminDefaultString(&s.FormPhoneHelpFI, fi.Form.PhoneHelp)
	adminDefaultString(&s.FormAddressHelpFI, fi.Form.AddressHelp)
	adminDefaultString(&s.FormCarHelpFI, fi.Form.CarHelp)
	adminDefaultString(&s.FormSubmitButtonFI, fi.Form.SubmitButton)
	adminDefaultString(&s.FormSubmitLoadingTextFI, fi.Form.SubmitLoadingText)
	adminDefaultString(&s.FormSuccessMessageFI, fi.Form.SuccessMessage)
	adminDefaultString(&s.FormErrorMessageFI, fi.Form.ErrorMessage)
	adminDefaultString(&s.FormNameRequiredFI, fi.Form.NameRequiredMessage)
	adminDefaultString(&s.FormContactRequiredFI, fi.Form.ContactRequiredMessage)
	adminDefaultString(&s.FormPhoneInvalidFI, fi.Form.PhoneInvalidMessage)
	adminDefaultString(&s.FormEmailInvalidFI, fi.Form.EmailInvalidMessage)
	adminDefaultString(&s.FormRegistrationInvalidFI, fi.Form.RegistrationInvalidMessage)
	adminDefaultString(&s.FormPhotosTooManyFI, fi.Form.PhotosTooManyMessage)
	adminDefaultString(&s.FormPhotosTypeFI, fi.Form.PhotosTypeMessage)
	adminDefaultString(&s.FormPhotosSizeFI, fi.Form.PhotosSizeMessage)
	adminDefaultString(&s.FormGeoUnsupportedFI, fi.Form.GeoUnsupportedMessage)
	adminDefaultString(&s.FormGeoLoadingFI, fi.Form.GeoLoadingText)
	adminDefaultString(&s.FormGeoAddedFI, fi.Form.GeoAddedText)
	adminDefaultString(&s.FormGeoFailedFI, fi.Form.GeoFailedMessage)
	adminDefaultString(&s.FormMapPickFirstFI, fi.Form.MapPickFirstMessage)
	adminDefaultString(&s.FormMapPointAddedFI, fi.Form.MapPointAddedText)
	adminDefaultString(&s.FormReverseGeocodeFailedFI, fi.Form.ReverseGeocodeFailedMessage)
	adminDefaultString(&s.FormNetworkErrorFI, fi.Form.NetworkErrorMessage)
	adminDefaultString(&s.PromoBadgeFI, fi.Promo.Badge)
	adminDefaultString(&s.PromoTitleFI, fi.Promo.Title)
	adminDefaultString(&s.PromoTextFI, fi.Promo.Text)
	adminDefaultString(&s.PromoButtonFI, fi.Promo.Button)
	adminDefaultString(&s.HeroPersonAltEN, en.Hero.PersonAlt)
	adminDefaultString(&s.TrustKickerEN, en.TrustPanel.Kicker)
	adminDefaultString(&s.TrustTitleEN, en.TrustPanel.Title)
	adminDefaultString(&s.TrustTextEN, en.TrustPanel.Text)
	adminDefaultString(&s.CapabilityKickerEN, en.Capability.Heading.Kicker)
	adminDefaultString(&s.CapabilityTitleEN, en.Capability.Heading.Title)
	adminDefaultString(&s.CapabilityTextEN, en.Capability.Heading.Text)
	adminDefaultString(&s.CapabilityCanDoTitleEN, en.Capability.CanDo.Title)
	adminDefaultString(&s.CapabilityNeedsWorkshopTitleEN, en.Capability.NeedsWorkshop.Title)
	adminDefaultString(&s.HeaderCTAEN, en.Header.CTA)
	adminDefaultString(&s.HeroTitleEN, en.Hero.Title)
	adminDefaultString(&s.HeroSubtitleEN, en.Hero.Subtitle)
	adminDefaultString(&s.PrimaryCTAEN, en.Hero.PrimaryCTA)
	adminDefaultString(&s.SecondaryCTAEN, en.Hero.SecondaryCTA)
	adminDefaultString(&s.ServicesKickerEN, en.Services.Heading.Kicker)
	adminDefaultString(&s.ServicesTitleEN, en.Services.Heading.Title)
	adminDefaultString(&s.ServicesTextEN, en.Services.Heading.Text)
	adminDefaultString(&s.ProcessKickerEN, en.Process.Heading.Kicker)
	adminDefaultString(&s.ProcessTitleEN, en.Process.Heading.Title)
	adminDefaultString(&s.ProcessTextEN, en.Process.Heading.Text)
	adminDefaultString(&s.FinalCTAKickerEN, en.FinalCTA.Kicker)
	adminDefaultString(&s.FinalCTATitleEN, en.FinalCTA.Title)
	adminDefaultString(&s.FinalCTATextEN, en.FinalCTA.Text)
	adminDefaultString(&s.FinalCTAButtonEN, en.FinalCTA.Button)
	adminDefaultString(&s.FormTitleEN, en.Form.Title)
	adminDefaultString(&s.FormIntroEN, en.Form.Intro)
	adminDefaultString(&s.FormLabelNameEN, en.Form.LabelName)
	adminDefaultString(&s.FormLabelPhoneEN, en.Form.LabelPhone)
	adminDefaultString(&s.FormLabelEmailEN, en.Form.LabelEmail)
	adminDefaultString(&s.FormLabelAddressEN, en.Form.LabelAddress)
	adminDefaultString(&s.FormLabelCarEN, en.Form.LabelCar)
	adminDefaultString(&s.FormLabelPhotosEN, en.Form.LabelPhotos)
	adminDefaultString(&s.FormLabelMessageEN, en.Form.LabelMessage)
	adminDefaultString(&s.FormPlaceholderNameEN, en.Form.PlaceholderName)
	adminDefaultString(&s.FormPlaceholderPhoneEN, en.Form.PlaceholderPhone)
	adminDefaultString(&s.FormPlaceholderEmailEN, en.Form.PlaceholderEmail)
	adminDefaultString(&s.FormPlaceholderAddressEN, en.Form.PlaceholderAddress)
	adminDefaultString(&s.FormPlaceholderCarEN, en.Form.PlaceholderCar)
	adminDefaultString(&s.FormPlaceholderMessageEN, en.Form.PlaceholderMessage)
	adminDefaultString(&s.FormUseLocationButtonEN, en.Form.UseLocationButton)
	adminDefaultString(&s.FormPickMapButtonEN, en.Form.PickMapButton)
	adminDefaultString(&s.FormUploadButtonTextEN, en.Form.UploadButtonText)
	adminDefaultString(&s.FormPhotosHintEN, en.Form.PhotosHint)
	adminDefaultString(&s.FormPhoneHelpEN, en.Form.PhoneHelp)
	adminDefaultString(&s.FormAddressHelpEN, en.Form.AddressHelp)
	adminDefaultString(&s.FormCarHelpEN, en.Form.CarHelp)
	adminDefaultString(&s.FormSubmitButtonEN, en.Form.SubmitButton)
	adminDefaultString(&s.FormSubmitLoadingTextEN, en.Form.SubmitLoadingText)
	adminDefaultString(&s.FormSuccessMessageEN, en.Form.SuccessMessage)
	adminDefaultString(&s.FormErrorMessageEN, en.Form.ErrorMessage)
	adminDefaultString(&s.FormNameRequiredEN, en.Form.NameRequiredMessage)
	adminDefaultString(&s.FormContactRequiredEN, en.Form.ContactRequiredMessage)
	adminDefaultString(&s.FormPhoneInvalidEN, en.Form.PhoneInvalidMessage)
	adminDefaultString(&s.FormEmailInvalidEN, en.Form.EmailInvalidMessage)
	adminDefaultString(&s.FormRegistrationInvalidEN, en.Form.RegistrationInvalidMessage)
	adminDefaultString(&s.FormPhotosTooManyEN, en.Form.PhotosTooManyMessage)
	adminDefaultString(&s.FormPhotosTypeEN, en.Form.PhotosTypeMessage)
	adminDefaultString(&s.FormPhotosSizeEN, en.Form.PhotosSizeMessage)
	adminDefaultString(&s.FormGeoUnsupportedEN, en.Form.GeoUnsupportedMessage)
	adminDefaultString(&s.FormGeoLoadingEN, en.Form.GeoLoadingText)
	adminDefaultString(&s.FormGeoAddedEN, en.Form.GeoAddedText)
	adminDefaultString(&s.FormGeoFailedEN, en.Form.GeoFailedMessage)
	adminDefaultString(&s.FormMapPickFirstEN, en.Form.MapPickFirstMessage)
	adminDefaultString(&s.FormMapPointAddedEN, en.Form.MapPointAddedText)
	adminDefaultString(&s.FormReverseGeocodeFailedEN, en.Form.ReverseGeocodeFailedMessage)
	adminDefaultString(&s.FormNetworkErrorEN, en.Form.NetworkErrorMessage)
	adminDefaultString(&s.PromoBadgeEN, en.Promo.Badge)
	adminDefaultString(&s.PromoTitleEN, en.Promo.Title)
	adminDefaultString(&s.PromoTextEN, en.Promo.Text)
	adminDefaultString(&s.PromoButtonEN, en.Promo.Button)
	adminDefaultString(&s.HeroBadgeFI, fi.Hero.Badge)
	adminDefaultString(&s.HeroPointsFI, strings.Join(fi.Hero.Points, "\n"))
	adminDefaultString(&s.CoverageCities, strings.Join(fi.Areas.Cities, "\n"))
	if len(fi.Hero.TrustCards) > 0 {
		adminDefaultString(&s.HeroTrustCard1TitleFI, fi.Hero.TrustCards[0].Title)
		adminDefaultString(&s.HeroTrustCard1TextFI, fi.Hero.TrustCards[0].Text)
	}
	if len(fi.Hero.TrustCards) > 1 {
		adminDefaultString(&s.HeroTrustCard2TitleFI, fi.Hero.TrustCards[1].Title)
		adminDefaultString(&s.HeroTrustCard2TextFI, fi.Hero.TrustCards[1].Text)
	}
	if len(fi.Hero.TrustCards) > 2 {
		adminDefaultString(&s.HeroTrustCard3TitleFI, fi.Hero.TrustCards[2].Title)
		adminDefaultString(&s.HeroTrustCard3TextFI, fi.Hero.TrustCards[2].Text)
	}
	adminDefaultString(&s.HeroBadgeEN, en.Hero.Badge)
	adminDefaultString(&s.HeroPointsEN, strings.Join(en.Hero.Points, "\n"))
	adminDefaultString(&s.CoverageCities, strings.Join(en.Areas.Cities, "\n"))
	if len(en.Hero.TrustCards) > 0 {
		adminDefaultString(&s.HeroTrustCard1TitleEN, en.Hero.TrustCards[0].Title)
		adminDefaultString(&s.HeroTrustCard1TextEN, en.Hero.TrustCards[0].Text)
	}
	if len(en.Hero.TrustCards) > 1 {
		adminDefaultString(&s.HeroTrustCard2TitleEN, en.Hero.TrustCards[1].Title)
		adminDefaultString(&s.HeroTrustCard2TextEN, en.Hero.TrustCards[1].Text)
	}
	if len(en.Hero.TrustCards) > 2 {
		adminDefaultString(&s.HeroTrustCard3TitleEN, en.Hero.TrustCards[2].Title)
		adminDefaultString(&s.HeroTrustCard3TextEN, en.Hero.TrustCards[2].Text)
	}
	if len(s.ServiceItems) == 0 {
		for i, a := range fi.Services.Items {
			b := a
			if i < len(en.Services.Items) {
				b = en.Services.Items[i]
			}
			s.ServiceItems = append(s.ServiceItems, AdminServiceItem{BadgeFI: a.Badge, TitleFI: a.Title, TextFI: a.Text, PriceFI: a.Price, BadgeEN: b.Badge, TitleEN: b.Title, TextEN: b.Text, PriceEN: b.Price, ImageURL: a.ImageURL})
		}
	}
	if len(s.CaseItems) == 0 {
		for i, a := range fi.Cases.Items {
			b := a
			if i < len(en.Cases.Items) {
				b = en.Cases.Items[i]
			}
			s.CaseItems = append(s.CaseItems, AdminCaseItem{TagFI: a.Tag, TitleFI: a.Title, TextFI: a.Text, PointsFI: a.Items, TagEN: b.Tag, TitleEN: b.Title, TextEN: b.Text, PointsEN: b.Items})
		}
	}
	return s
}
func adminDefaultString(target *string, value string) {
	if strings.TrimSpace(*target) == "" {
		*target = strings.NewReplacer("70 €", "{visit} €", "€70", "€{visit}", "50 €", "{diagnostic} €", "€50", "€{diagnostic}").Replace(value)
	}
}
