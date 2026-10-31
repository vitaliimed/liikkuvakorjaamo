package main

import "html/template"

type NavItem struct {
	Label string
	Href  string
}

type SectionHeading struct {
	Kicker string
	Title  string
	Text   string
}

type SimpleCard struct {
	Title    string
	Text     string
	ImageURL string
	Badge    string
	Price    string
}

type StepCard struct {
	Number string
	Title  string
	Text   string
}

type CaseCard struct {
	Tag   string
	Title string
	Text  string
	Items []string
}

type TestimonialCard struct {
	Quote       string
	Author      string
	ImageURL    string
	Rating      int
	RatingStars string
}

type AdminTestimonialItem struct {
	Name     string `json:"name" firestore:"name"`
	QuoteFI  string `json:"quote_fi" firestore:"quote_fi"`
	QuoteEN  string `json:"quote_en" firestore:"quote_en"`
	PhotoURL string `json:"photo_url" firestore:"photo_url"`
	Rating   int    `json:"rating" firestore:"rating"`
}

type AdminServiceItem struct {
	BadgeFI  string `json:"badge_fi" firestore:"badge_fi"`
	PriceFI  string `json:"price_fi" firestore:"price_fi"`
	TitleFI  string `json:"title_fi" firestore:"title_fi"`
	TextFI   string `json:"text_fi" firestore:"text_fi"`
	BadgeEN  string `json:"badge_en" firestore:"badge_en"`
	PriceEN  string `json:"price_en" firestore:"price_en"`
	TitleEN  string `json:"title_en" firestore:"title_en"`
	TextEN   string `json:"text_en" firestore:"text_en"`
	ImageURL string `json:"image_url" firestore:"image_url"`
}

type AdminCaseItem struct {
	TagFI    string   `json:"tag_fi" firestore:"tag_fi"`
	TitleFI  string   `json:"title_fi" firestore:"title_fi"`
	TextFI   string   `json:"text_fi" firestore:"text_fi"`
	PointsFI []string `json:"points_fi" firestore:"points_fi"`

	TagEN    string   `json:"tag_en" firestore:"tag_en"`
	TitleEN  string   `json:"title_en" firestore:"title_en"`
	TextEN   string   `json:"text_en" firestore:"text_en"`
	PointsEN []string `json:"points_en" firestore:"points_en"`
}

type FAQItem struct {
	Question string
	Answer   string
}

type HeaderContent struct {
	LogoText string
	LogoURL  string
	Nav      []NavItem
	LangFI   string
	LangEN   string
	CTA      string
}

type FooterContent struct {
	PrivacyLabel     string
	Phone            string
	PhoneHref        string
	WhatsAppHref     string
	Email            string
	BusinessName     string
	YTunnus          string
	BaseCity         string
	WorkingHours     string
	InstagramURL     string
	TikTokURL        string
	FacebookGroupURL string
}

type MapModalContent struct {
	Title         string
	ConfirmButton string
}

type HeroContent struct {
	Badge            string
	Title            string
	Subtitle         string
	Points           []string
	PrimaryCTA       string
	SecondaryCTA     string
	TrustCards       []SimpleCard
	PersonDesktopURL string
	PersonAlt        string
	BackgroundURL    string
}

type AreasContent struct {
	Heading     SectionHeading
	Cities      []string
	NoteTitle   string
	NoteText    string
	NoteButton  string
	MapImageURL string
}

type FAQContent struct {
	Heading SectionHeading
	Items   []FAQItem
}

type FinalCTAContent struct {
	Kicker string
	Title  string
	Text   string
	Button string
}

type PromoContent struct {
	Enabled bool
	Badge   string
	Title   string
	Text    string
	Button  string
	URL     string
}

type TrustPanelContent struct {
	Kicker string
	Title  string
	Text   string
	Items  []string
}

type CapabilityColumn struct {
	Title string
	Items []string
}

type CapabilityContent struct {
	Heading       SectionHeading
	CanDo         CapabilityColumn
	NeedsWorkshop CapabilityColumn
}

type FormContent struct {
	Title string
	Intro string

	LabelName        string
	LabelPhone       string
	LabelEmail       string
	LabelAddress     string
	LabelAddressHint string
	LabelCar         string
	LabelPhotos      string
	LabelMessage     string

	PlaceholderName        string
	PlaceholderPhone       string
	PlaceholderEmail       string
	PlaceholderAddress     string
	PlaceholderAddressHint string
	PlaceholderCar         string
	PlaceholderMessage     string

	UseLocationButton string
	PickMapButton     string
	UploadButtonText  string
	PhotosHint        string
	PhoneHelp         string
	AddressHelp       string
	CarHelp           string

	VisitPriceKicker          string
	VisitPriceTitle           string
	VisitPriceText            string
	VisitPriceItems           []string
	VisitPriceFootnote        string
	VisitPriceConsentLabel    string
	VisitPriceRequiredMessage string

	SubmitButton      string
	SubmitLoadingText string
	SuccessMessage    string
	ErrorMessage      string

	NameRequiredMessage         string
	ContactRequiredMessage      string
	PhoneInvalidMessage         string
	EmailInvalidMessage         string
	RegistrationInvalidMessage  string
	PhotosTooManyMessage        string
	PhotosTypeMessage           string
	PhotosSizeMessage           string
	GeoUnsupportedMessage       string
	GeoLoadingText              string
	GeoAddedText                string
	GeoFailedMessage            string
	MapPickFirstMessage         string
	MapPointAddedText           string
	ReverseGeocodeFailedMessage string
	NetworkErrorMessage         string
}

type LandingContent struct {
	SiteName string
	Header   HeaderContent
	Footer   FooterContent
	MapModal MapModalContent
	Hero     HeroContent
	Stats    []SimpleCard

	Services struct {
		Heading SectionHeading
		Items   []SimpleCard
	}

	Process struct {
		Heading SectionHeading
		Items   []StepCard
	}

	Cases struct {
		Heading SectionHeading
		Items   []CaseCard
	}

	Areas AreasContent

	Testimonials struct {
		Heading SectionHeading
		Items   []TestimonialCard
	}

	TrustPanel TrustPanelContent
	Capability CapabilityContent
	FAQ        FAQContent
	FinalCTA   FinalCTAContent
	Promo      PromoContent
	Form       FormContent

	ShowStats        bool
	ShowServices     bool
	ShowProcess      bool
	ShowCases        bool
	ShowCapability   bool
	ShowAreas        bool
	ShowTestimonials bool
	ShowFAQ          bool
	ShowFinalCTA     bool
}

type LandingPageData struct {
	Lang                         string
	PageTitle                    string
	MetaDescription              string
	RobotsMeta                   string
	CanonicalURL                 string
	HrefLangFI                   string
	HrefLangEN                   string
	OGTitle                      string
	OGDescription                string
	OGURL                        string
	FaviconSVGURL                string
	Favicon32URL                 string
	Favicon16URL                 string
	FaviconICOURL                string
	AppleTouchIconURL            string
	EnableConsentMode            bool
	PrimaryGoogleTagID           string
	GA4MeasurementID             string
	GoogleAdsTagID               string
	GoogleAdsLeadConversionLabel string
	MetaPixelID                  string
	LinkedInPartnerID            string
	LinkedInLeadConversionID     string
	TikTokPixelID                string
	HeroStyleVars                template.CSS
	HeroSceneImageStyle          template.CSS
	HeroPersonLayerStyle         template.CSS
	HeroPersonImageStyle         template.CSS
	AreasStyleVars               template.CSS
	AreasMapImageStyle           template.CSS
	ServiceImageStyle            template.CSS

	HeroTitle    string
	HeroSubtitle string
	FormTitle    string

	LabelName        string
	LabelPhone       string
	LabelEmail       string
	LabelAddress     string
	LabelAddressHint string
	LabelCar         string
	LabelMessage     string

	PlaceholderName        string
	PlaceholderPhone       string
	PlaceholderEmail       string
	PlaceholderAddress     string
	PlaceholderAddressHint string
	PlaceholderCar         string
	PlaceholderMessage     string

	ButtonSubmit   string
	SuccessMessage string
	ErrorMessage   string
	LanguageValue  string

	Content LandingContent
}

type ThankYouPageData struct {
	Lang            string
	PageTitle       string
	MetaDescription string
	RobotsMeta      string
	CanonicalURL    string
	HrefLangFI      string
	HrefLangEN      string
	OGTitle         string
	OGDescription   string
	OGURL           string
	LogoURL         string

	SiteName                     string
	Footer                       FooterContent
	FaviconSVGURL                string
	Favicon32URL                 string
	Favicon16URL                 string
	FaviconICOURL                string
	AppleTouchIconURL            string
	EnableConsentMode            bool
	PrimaryGoogleTagID           string
	GA4MeasurementID             string
	GoogleAdsTagID               string
	GoogleAdsLeadConversionLabel string
	MetaPixelID                  string
	LinkedInPartnerID            string
	LinkedInLeadConversionID     string
	TikTokPixelID                string

	ThankYouTitle     string
	ThankYouText      string
	ThankYouButton    string
	ThankYouButtonURL string
	VisitPriceEUR     int
}

type ReviewPageData struct {
	Lang              string
	PageTitle         string
	MetaDescription   string
	CanonicalURL      string
	LogoURL           string
	FaviconSVGURL     string
	Favicon32URL      string
	Favicon16URL      string
	FaviconICOURL     string
	AppleTouchIconURL string
	LeadID            string
	Token             string
	Name              string
	NameMode          string
	Alias             string
	Rating            int
	Message           string
	PublishConsent    bool
	Errors            map[string]string
	Submitted         bool
	AlreadySubmitted  bool
}

type PrivacyPageData struct {
	Lang                         string
	PageTitle                    string
	MetaDescription              string
	RobotsMeta                   string
	CanonicalURL                 string
	HrefLangFI                   string
	HrefLangEN                   string
	OGTitle                      string
	OGDescription                string
	OGURL                        string
	LogoURL                      string
	FaviconSVGURL                string
	Favicon32URL                 string
	Favicon16URL                 string
	FaviconICOURL                string
	AppleTouchIconURL            string
	EnableConsentMode            bool
	PrimaryGoogleTagID           string
	GA4MeasurementID             string
	GoogleAdsTagID               string
	GoogleAdsLeadConversionLabel string
	MetaPixelID                  string
	LinkedInPartnerID            string
	LinkedInLeadConversionID     string
	TikTokPixelID                string

	PrivacyTitle  string
	PrivacyIntro  string
	Section1Title string
	Section1Text  string
	Section2Title string
	Section2Text  string
	Section3Title string
	Section3Text  string
	Section4Title string
	Section4Text  string
	Section5Title string
	Section5Text  string
}

type SiteSettings struct {
	Revision                     int64 `firestore:"revision"`
	ProVisitEUR                  int
	ProDiagnosticEUR             int
	ProFontSize                  int
	ProAccent                    string
	ProTheme                     string
	ProNightStart                string
	ProNightEnd                  string
	ProPricingText               string
	ProAreasText                 string
	ProDiagnosticsText           string
	ProPricingTitle              string
	ProPricingDescription        string
	ProPricingHeading            string
	ProAreasTitle                string
	ProAreasDescription          string
	ProAreasHeading              string
	ProDiagnosticsTitle          string
	ProDiagnosticsDescription    string
	ProDiagnosticsHeading        string
	Phone                        string
	WhatsAppNumber               string
	ServiceEmail                 string
	BusinessName                 string
	YTunnus                      string
	BaseCity                     string
	WorkingHoursFI               string
	WorkingHoursEN               string
	LogoURL                      string
	LogoDarkURL                  string
	EmailLogoURL                 string
	EmailConfirmationSubjectFI   string
	EmailConfirmationSubjectEN   string
	EmailConfirmationTextFI      string
	EmailConfirmationTextEN      string
	EmailReadySubjectFI          string
	EmailReadySubjectEN          string
	EmailReadyTextFI             string
	EmailReadyTextEN             string
	EmailSignatureFI             string
	EmailSignatureEN             string
	AnalyticsDisabled            bool
	AnalyticsEventsConfigured    bool
	AnalyticsDisabledEvents      []string
	InstagramURL                 string
	TikTokURL                    string
	FacebookGroupURL             string
	GA4MeasurementID             string
	GoogleAdsTagID               string
	GoogleAdsLeadConversionLabel string
	MetaPixelID                  string
	LinkedInPartnerID            string
	LinkedInLeadConversionID     string
	TikTokPixelID                string
	Testimonials                 []AdminTestimonialItem `json:"testimonials" firestore:"testimonials"`
	ServiceItems                 []AdminServiceItem     `json:"service_items" firestore:"service_items"`
	CaseItems                    []AdminCaseItem        `json:"case_items" firestore:"case_items"`

	FaviconSVGURL        string
	Favicon32URL         string
	Favicon16URL         string
	FaviconICOURL        string
	AppleTouchIconURL    string
	HeroPersonDesktopURL string
	HeroPersonAltFI      string
	HeroPersonAltEN      string
	HeroBackgroundURL    string
	AreasMapImageURL     string
	ServicesImage1URL    string
	ServicesImage2URL    string
	ServicesImage3URL    string
	ServicesImage4URL    string
	ServicesImage5URL    string
	ServicesImage6URL    string

	HeroScenePreset     string
	HeroSceneBrightness string
	HeroSceneContrast   string
	HeroSceneSaturation string
	HeroSceneBlur       string
	HeroSceneOpacity    string
	HeroSceneScale      string
	HeroOverlayOpacity  string

	HeroPersonPreset     string
	HeroPersonBrightness string
	HeroPersonContrast   string
	HeroPersonSaturation string
	HeroPersonOpacity    string
	HeroPersonScale      string
	HeroPersonPositionX  string
	HeroPersonPositionY  string
	HeroPersonShadow     string

	AreasMapPreset      string
	AreasMapBrightness  string
	AreasMapContrast    string
	AreasMapSaturation  string
	AreasMapBlur        string
	AreasMapOpacity     string
	AreasMapScale       string
	AreasOverlayOpacity string

	ServiceImagePreset     string
	ServiceImageBrightness string
	ServiceImageContrast   string
	ServiceImageSaturation string
	ServiceImageBlur       string
	ServiceImageOpacity    string
	ServiceImageScale      string

	HeaderCTAFI string
	HeaderCTAEN string

	CoverageCities string

	HeroBadgeFI  string
	HeroBadgeEN  string
	HeroPointsFI string
	HeroPointsEN string

	HeroTrustCard1TitleFI string
	HeroTrustCard1TextFI  string
	HeroTrustCard2TitleFI string
	HeroTrustCard2TextFI  string
	HeroTrustCard3TitleFI string
	HeroTrustCard3TextFI  string
	HeroTrustCard1TitleEN string
	HeroTrustCard1TextEN  string
	HeroTrustCard2TitleEN string
	HeroTrustCard2TextEN  string
	HeroTrustCard3TitleEN string
	HeroTrustCard3TextEN  string

	TrustKickerFI string
	TrustKickerEN string
	TrustTitleFI  string
	TrustTitleEN  string
	TrustTextFI   string
	TrustTextEN   string
	TrustItemsFI  string
	TrustItemsEN  string

	CapabilityKickerFI             string
	CapabilityKickerEN             string
	CapabilityTitleFI              string
	CapabilityTitleEN              string
	CapabilityTextFI               string
	CapabilityTextEN               string
	CapabilityCanDoTitleFI         string
	CapabilityCanDoTitleEN         string
	CapabilityCanDoItemsFI         string
	CapabilityCanDoItemsEN         string
	CapabilityNeedsWorkshopTitleFI string
	CapabilityNeedsWorkshopTitleEN string
	CapabilityNeedsWorkshopItemsFI string
	CapabilityNeedsWorkshopItemsEN string

	HeroTitleFI    string
	HeroTitleEN    string
	HeroSubtitleFI string
	HeroSubtitleEN string

	PrimaryCTAFI   string
	PrimaryCTAEN   string
	SecondaryCTAFI string
	SecondaryCTAEN string

	ServicesKickerFI string
	ServicesKickerEN string
	ServicesTitleFI  string
	ServicesTitleEN  string
	ServicesTextFI   string
	ServicesTextEN   string

	ProcessKickerFI string
	ProcessKickerEN string
	ProcessTitleFI  string
	ProcessTitleEN  string
	ProcessTextFI   string
	ProcessTextEN   string

	FinalCTAKickerFI string
	FinalCTAKickerEN string
	FinalCTATitleFI  string
	FinalCTATitleEN  string
	FinalCTATextFI   string
	FinalCTATextEN   string
	FinalCTAButtonFI string
	FinalCTAButtonEN string

	FormTitleFI                string
	FormIntroFI                string
	FormLabelNameFI            string
	FormLabelPhoneFI           string
	FormLabelEmailFI           string
	FormLabelAddressFI         string
	FormLabelCarFI             string
	FormLabelPhotosFI          string
	FormLabelMessageFI         string
	FormPlaceholderNameFI      string
	FormPlaceholderPhoneFI     string
	FormPlaceholderEmailFI     string
	FormPlaceholderAddressFI   string
	FormPlaceholderCarFI       string
	FormPlaceholderMessageFI   string
	FormUseLocationButtonFI    string
	FormPickMapButtonFI        string
	FormUploadButtonTextFI     string
	FormPhotosHintFI           string
	FormPhoneHelpFI            string
	FormAddressHelpFI          string
	FormCarHelpFI              string
	FormSubmitButtonFI         string
	FormSubmitLoadingTextFI    string
	FormSuccessMessageFI       string
	FormErrorMessageFI         string
	FormNameRequiredFI         string
	FormContactRequiredFI      string
	FormPhoneInvalidFI         string
	FormEmailInvalidFI         string
	FormRegistrationInvalidFI  string
	FormPhotosTooManyFI        string
	FormPhotosTypeFI           string
	FormPhotosSizeFI           string
	FormGeoUnsupportedFI       string
	FormGeoLoadingFI           string
	FormGeoAddedFI             string
	FormGeoFailedFI            string
	FormMapPickFirstFI         string
	FormMapPointAddedFI        string
	FormReverseGeocodeFailedFI string
	FormNetworkErrorFI         string

	FormTitleEN                string
	FormIntroEN                string
	FormLabelNameEN            string
	FormLabelPhoneEN           string
	FormLabelEmailEN           string
	FormLabelAddressEN         string
	FormLabelCarEN             string
	FormLabelPhotosEN          string
	FormLabelMessageEN         string
	FormPlaceholderNameEN      string
	FormPlaceholderPhoneEN     string
	FormPlaceholderEmailEN     string
	FormPlaceholderAddressEN   string
	FormPlaceholderCarEN       string
	FormPlaceholderMessageEN   string
	FormUseLocationButtonEN    string
	FormPickMapButtonEN        string
	FormUploadButtonTextEN     string
	FormPhotosHintEN           string
	FormPhoneHelpEN            string
	FormAddressHelpEN          string
	FormCarHelpEN              string
	FormSubmitButtonEN         string
	FormSubmitLoadingTextEN    string
	FormSuccessMessageEN       string
	FormErrorMessageEN         string
	FormNameRequiredEN         string
	FormContactRequiredEN      string
	FormPhoneInvalidEN         string
	FormEmailInvalidEN         string
	FormRegistrationInvalidEN  string
	FormPhotosTooManyEN        string
	FormPhotosTypeEN           string
	FormPhotosSizeEN           string
	FormGeoUnsupportedEN       string
	FormGeoLoadingEN           string
	FormGeoAddedEN             string
	FormGeoFailedEN            string
	FormMapPickFirstEN         string
	FormMapPointAddedEN        string
	FormReverseGeocodeFailedEN string
	FormNetworkErrorEN         string

	SectionVisibilityConfigured bool
	StatsEnabled                bool
	ServicesEnabled             bool
	ProcessEnabled              bool
	CasesEnabled                bool
	CapabilityEnabled           bool
	AreasEnabled                bool
	TestimonialsEnabled         bool
	FAQEnabled                  bool
	FinalCTAEnabled             bool

	PromoEnabled  bool
	PromoBadgeFI  string
	PromoBadgeEN  string
	PromoTitleFI  string
	PromoTitleEN  string
	PromoTextFI   string
	PromoTextEN   string
	PromoButtonFI string
	PromoButtonEN string
	PromoURL      string

	LandingSEOTitleFI       string
	LandingSEOTitleEN       string
	LandingSEODescriptionFI string
	LandingSEODescriptionEN string

	ThankYouSEOTitleFI       string
	ThankYouSEOTitleEN       string
	ThankYouSEODescriptionFI string
	ThankYouSEODescriptionEN string
	ThankYouTitleFI          string
	ThankYouTitleEN          string
	ThankYouTextFI           string
	ThankYouTextEN           string
	ThankYouButtonFI         string
	ThankYouButtonEN         string
	ThankYouButtonURLFI      string
	ThankYouButtonURLEN      string

	PrivacySEOTitleFI       string
	PrivacySEOTitleEN       string
	PrivacySEODescriptionFI string
	PrivacySEODescriptionEN string
	PrivacyTitleFI          string
	PrivacyTitleEN          string
	PrivacyIntroFI          string
	PrivacyIntroEN          string
	PrivacySection1TitleFI  string
	PrivacySection1TitleEN  string
	PrivacySection1TextFI   string
	PrivacySection1TextEN   string
	PrivacySection2TitleFI  string
	PrivacySection2TitleEN  string
	PrivacySection2TextFI   string
	PrivacySection2TextEN   string
	PrivacySection3TitleFI  string
	PrivacySection3TitleEN  string
	PrivacySection3TextFI   string
	PrivacySection3TextEN   string
	PrivacySection4TitleFI  string
	PrivacySection4TitleEN  string
	PrivacySection4TextFI   string
	PrivacySection4TextEN   string
	PrivacySection5TitleFI  string
	PrivacySection5TitleEN  string
	PrivacySection5TextFI   string
	PrivacySection5TextEN   string

	LandingRobotsMeta  string
	ThankYouRobotsMeta string
	PrivacyRobotsMeta  string
	RobotsTXT          string
}

type AdminLeadListItem struct {
	ID               string
	CreatedAt        string
	Name             string
	Phone            string
	Email            string
	Car              string
	MessagePreview   string
	Language         string
	ProcessingStatus string
	TelegramStatus   string
	SheetsStatus     string
	ClientStatus     string
	ReadyEmailStatus string
	LastError        string
}

type AdminLeadDetails struct {
	ID                      string
	CreatedAt               string
	UpdatedAt               string
	TaskEnqueuedAt          string
	ProcessedAt             string
	Name                    string
	Phone                   string
	Email                   string
	Address                 string
	AddressHint             string
	LocationSource          string
	Latitude                string
	Longitude               string
	MapsURL                 string
	Car                     string
	Message                 string
	VisitPriceEUR           int
	VisitPriceAccepted      bool
	Language                string
	Source                  string
	PageURL                 string
	Referer                 string
	UTMSource               string
	UTMMedium               string
	UTMCampaign             string
	UTMTerm                 string
	UTMContent              string
	IP                      string
	UserAgent               string
	ProcessingStatus        string
	TelegramStatus          string
	SheetsStatus            string
	ClientStatus            string
	ReadyEmailStatus        string
	ReadyEmailError         string
	ReadyEmailSentAt        string
	ConfirmationEmailStatus string
	ConfirmationEmailError  string
	ConfirmationEmailSentAt string
	ReviewInviteEmailStatus string
	ReviewInviteEmailError  string
	ReviewInviteEmailSentAt string
	LastError               string
	PhotoURLs               []string
}

type AdminReviewListItem struct {
	ID        string
	LeadID    string
	CreatedAt string
	Name      string
	Rating    int
	Stars     string
	Language  string
	Message   string
	Status    string
}

type AdminPageData struct {
	AnalyticsEvents    []analyticsEventSetting
	AssetsConfigured   bool
	AssetUploadMessage string
	TelegramConfigured bool
	CSRF               string
	ReleaseVersion     string
	IntakePaused       bool
	PageTitle          string
	LoginRequired      bool
	Error              string
	Saved              bool
	MailEnabled        bool
	Settings           SiteSettings
	ActiveTab          string

	OverviewTotalLeads     int
	OverviewFailedLeads    int
	OverviewPendingEmail   int
	OverviewMissingAssets  int
	OverviewPendingReviews int

	StatusFilter          string
	Leads                 []AdminLeadListItem
	SelectedLead          *AdminLeadDetails
	RetryQueued           bool
	ReadyEmailSent        bool
	ConfirmationEmailSent bool
	ReviewEmailSent       bool
	StatusUpdated         bool
	ClientStatusFilter    string
	ReviewStatusFilter    string
	Reviews               []AdminReviewListItem
}
