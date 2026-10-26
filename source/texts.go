package main

import (
	"fmt"
	"strings"
)

func getLandingTexts(lang string) map[string]string {
	content := getLandingContent(lang)

	pageTitle := "Mobile car repair in Uusimaa"
	metaDescription := "Fast mobile car repair in Uusimaa. Send a request and get contacted quickly."
	if lang == "fi" {
		pageTitle = "Auton korjaus paikan päällä Uudellamaalla"
		metaDescription = "Nopea auton korjaus paikan päällä Uudellamaalla. Lähetä pyyntö ja saat nopean yhteydenoton."
	}

	return map[string]string{
		"LANG":                     lang,
		"PAGE_TITLE":               pageTitle,
		"META_DESCRIPTION":         metaDescription,
		"HERO_TITLE":               content.Hero.Title,
		"HERO_SUBTITLE":            content.Hero.Subtitle,
		"FORM_TITLE":               content.Form.Title,
		"LABEL_NAME":               content.Form.LabelName,
		"LABEL_PHONE":              content.Form.LabelPhone,
		"LABEL_EMAIL":              content.Form.LabelEmail,
		"LABEL_ADDRESS":            content.Form.LabelAddress,
		"LABEL_ADDRESS_HINT":       content.Form.LabelAddressHint,
		"LABEL_CAR":                content.Form.LabelCar,
		"LABEL_MESSAGE":            content.Form.LabelMessage,
		"PLACEHOLDER_NAME":         content.Form.PlaceholderName,
		"PLACEHOLDER_PHONE":        content.Form.PlaceholderPhone,
		"PLACEHOLDER_EMAIL":        content.Form.PlaceholderEmail,
		"PLACEHOLDER_ADDRESS":      content.Form.PlaceholderAddress,
		"PLACEHOLDER_ADDRESS_HINT": content.Form.PlaceholderAddressHint,
		"PLACEHOLDER_CAR":          content.Form.PlaceholderCar,
		"PLACEHOLDER_MESSAGE":      content.Form.PlaceholderMessage,
		"BUTTON_SUBMIT":            content.Form.SubmitButton,
		"SUCCESS_MESSAGE":          content.Form.SuccessMessage,
		"ERROR_MESSAGE":            content.Form.ErrorMessage,
	}
}

func getLandingContent(lang string) LandingContent {
	if lang == "fi" {
		return liikkuvaContent(landingContentFI(), "fi")
	}
	return liikkuvaContent(landingContentEN(), "en")
}

func landingContentFI() LandingContent {
	content := LandingContent{
		SiteName: "Liikkuva Auto Korjaamo",
		Header: HeaderContent{
			LogoText: "Liikkuva Auto Korjaamo",
			LogoURL:  "/static/img/logo_liikkuva.svg",
			Nav: []NavItem{
				{Label: "Palvelut", Href: "#services"},
				{Label: "Miten toimii", Href: "#how-it-works"},
				{Label: "Alueet", Href: "#areas"},
				{Label: "UKK", Href: "#faq"},
			},
			LangFI: "FI",
			LangEN: "EN",
			CTA:    "Pyydä huoltoa",
		},
		Footer: FooterContent{
			PrivacyLabel: "Tietosuoja",
			Email:        "service@autohelp.fi",
		},
		MapModal: MapModalContent{
			Title:         "Valitse auton sijainti",
			ConfirmButton: "Käytä tätä pistettä",
		},
		Hero: HeroContent{
			Badge:        fmt.Sprintf("Liikkuva autokorjaamo • Käynti %d €", visitPriceEUR),
			Title:        "Liikkuva autokorjaamo Uudellamaalla",
			Subtitle:     "Diagnostiikka, pienet korjaukset ja selkeä ensiarvio kotiin, työpaikalle tai tien varteen.",
			PrimaryCTA:   "Pyydä apua nyt",
			SecondaryCTA: "Katso palvelut",
			Points: []string{
				"Nopea yhteydenotto",
				"Diagnostiikka ja korjaukset paikan päällä",
				"Selkeä arvio ennen työn alkua",
			},
			TrustCards: []SimpleCard{
				{Title: "Paikan päällä", Text: "Koti, työpaikka tai tienvarsi"},
				{Title: "Selkeä", Text: "Selkeä viestintä ja arvio"},
				{Title: "Paikallinen", Text: "Painotus Uudenmaan alueella"},
			},
		},
		Stats: []SimpleCard{
			{Title: fmt.Sprintf("Käynti %d €", visitPriceEUR), Text: "Sisältää saapumisen ja alustavan vianmäärityksen"},
			{Title: "Paikan päällä korjaukset", Text: "Moni yleinen vika ilman hinausta"},
			{Title: "Selkeä prosessi", Text: "Diagnoosi, arvio ja vasta sen jälkeen työ"},
			{Title: "8 pääaluetta", Text: "Helsinki, Espoo, Vantaa, Porvoo, Hyvinkää, Järvenpää, Lohja ja Kerava"},
		},
		Areas: AreasContent{
			Heading: SectionHeading{
				Kicker: "Toiminta-alue",
				Title:  "Palvelualue Uudellamaalla",
				Text:   "Näytämme sivulla 8 suurinta ja käytännössä tärkeintä kaupunkia Uudeltamaalta, jotta lista pysyy selkeänä myös mobiilissa.",
			},
			Cities:     []string{"Helsinki", "Espoo", "Vantaa", "Porvoo", "Hyvinkää", "Järvenpää", "Lohja", "Kerava"},
			NoteTitle:  "Tarvitsetko apua ydinaluetta kauempana?",
			NoteText:   "Lähetä sijainti lomakkeella. Tarkistamme tapauksen ja vahvistamme saatavuuden ennen käyntiä.",
			NoteButton: "Lähetä pyyntö",
		},
		TrustPanel: TrustPanelContent{
			Kicker: "Luottamus",
			Title:  "Tiedät etukäteen kuka tulee ja miten työ etenee",
			Text:   "Tämä on se vahva luottamusblokki formiin viereen: selkeä vastausaika, kielituki, maksutapa ja kuitti ennen kuin asiakas lähettää pyynnön.",
			Items:  []string{fmt.Sprintf("Asentajan käynti ja alustava vianmääritys %d €", visitPriceEUR), "Mahdollisuus nopeaan korjaukseen paikan päällä", "Lisätyöt ja varaosat sovitaan ennen työn alkua", "Palvelu suomeksi, englanniksi, venäjäksi ja ukrainaksi"},
		},
		Capability: CapabilityContent{
			Heading:       SectionHeading{Kicker: "Paikan päällä / korjaamo", Title: "Mikä onnistuu paikan päällä — ja mikä vaatii korjaamon", Text: "Tämä blokki vähentää turhia odotuksia jo ennen yhteydenottoa."},
			CanDo:         CapabilityColumn{Title: "Voimme usein tehdä paikan päällä", Items: []string{"Akun testaus ja akun vaihto", "Vikakoodien luku ja ensidiagnoosi", "Käynnistys- ja latausongelmien perusselvitys", "Valitut anturien ja pienten osien vaihdot", "Jarrujen tarkastus ja mallikohtaiset kevyet työt"}},
			NeedsWorkshop: CapabilityColumn{Title: "Usein vaatii korjaamon", Items: []string{"Laajat moottori- ja vaihteistotyöt", "Työt jotka vaativat nosturia tai erikoislaitteita", "Suuret alustaremontit ja ruostekorjaukset", "Työt joissa turvallinen pääsy osiin ei onnistu paikan päällä", "Tapaukset joissa vika vaatii pidemmän purun tai syvätestauksen"}},
		},

		FAQ: FAQContent{
			Heading: SectionHeading{Kicker: "UKK", Title: "Usein kysytyt kysymykset"},
			Items: []FAQItem{
				{Question: fmt.Sprintf("Mitä %d € käyntimaksu sisältää?", visitPriceEUR), Answer: "Käyntimaksu sisältää asentajan saapumisen kohteeseen, alustavan vianmäärityksen ja mahdollisuuden nopeaan korjaukseen, jos työ voidaan tehdä turvallisesti heti. Varaosat ja laajemmat työt sovitaan ja veloitetaan erikseen ennen työn aloittamista."},
				{Question: "Kuinka nopeasti voitte vastata?", Answer: "Se riippuu työtilanteesta, sijainnista ja viasta. Lähetä pyyntö ja vahvistamme saatavuuden nopeasti."},
				{Question: "Tuotteko työkalut ja osat?", Answer: "Työkalut kuuluvat palveluun. Sopivat osat riippuvat viasta ja vahvistetaan yhteydenoton aikana."},
				{Question: "Voiko kaikki korjaukset tehdä paikan päällä?", Answer: "Ei. Monet yleiset viat voidaan diagnosoida tai korjata paikan päällä, mutta osa töistä vaatii korjaamon."},
				{Question: "Palveletteko vain suomeksi?", Answer: "Palvelu onnistuu sekä suomeksi ja englanniksi ja venäjäksi ja ukrainaksi"},
			},
		},
		FinalCTA: FinalCTAContent{
			Kicker: "Tarvitsetko apua nyt?",
			Title:  "Lähetä pyyntö ja saat nopean vastauksen",
			Text:   "Kuvaile ongelma, sijainti ja auton tiedot. Tarkistamme pyyntösi mahdollisimman pian.",
			Button: "Pyydä huoltoa",
		},
		Form: FormContent{
			Title:                  "Pyydä ensiarvio",
			Intro:                  "Kerro ongelma, sijainti ja yhteystietosi. Rekisterinumero ja kuvat nopeuttavat arviota.",
			LabelName:              "Nimi",
			LabelPhone:             "Puhelin",
			LabelEmail:             "Sähköposti",
			LabelAddress:           "Osoite",
			LabelAddressHint:       "Tarkennus osoitteeseen",
			LabelCar:               "Rekisterinumero",
			LabelPhotos:            "Kuvat",
			LabelMessage:           "Vika / pyyntö",
			PlaceholderName:        "Nimesi",
			PlaceholderPhone:       "+358... tai 040...",
			PlaceholderEmail:       "email@example.com",
			PlaceholderAddress:     "Katu ja numero / piha / pysäköinti",
			PlaceholderAddressHint: "Jos osoite toistuu, lisää kaupunki, rappu tai maamerkki",
			PlaceholderCar:         "Esim. ABC-123",
			PlaceholderMessage:     "Kuvaile ongelma",
			UseLocationButton:      "Käytä sijaintiani",
			PickMapButton:          "Valitse piste kartalta",
			UploadButtonText:       "Lisää kuvat",
			PhotosHint:             "Voit lisätä enintään 3 kuvaa. JPG, PNG tai WEBP, max 10 MB / kuva.",
			//PhoneHelp:              "Syötä numero muodossa +358 40 123 45 67",
			AddressHelp:               "Voit kirjoittaa osoitteen itse tai täyttää sen sijainnista.",
			CarHelp:                   "Syötä rekisterinumero muodossa ABC-123",
			VisitPriceKicker:          "Selkeä hinta ennen lähetystä",
			VisitPriceTitle:           fmt.Sprintf("Asentajan käynti %d €", visitPriceEUR),
			VisitPriceText:            "Sisältää saapumisen, alustavan vianmäärityksen ja mahdollisuuden nopeaan korjaukseen paikan päällä.",
			VisitPriceItems:           []string{"Asentajan saapumisen ajoneuvon luokse", "Alustavan vianmäärityksen paikan päällä", "Mahdollisuuden nopeaan korjaukseen, jos se voidaan tehdä turvallisesti heti"},
			VisitPriceFootnote:        "Varaosat ja laajemmat työt eivät sisälly käyntimaksuun. Niistä sovitaan kanssasi ennen työn aloittamista.",
			VisitPriceConsentLabel:    fmt.Sprintf("Ymmärrän, että asentajan käynti maksaa %d €.", visitPriceEUR),
			VisitPriceRequiredMessage: fmt.Sprintf("Vahvista, että ymmärrät %d € käyntimaksun.", visitPriceEUR),
			SubmitButton:              "Lähetä pyyntö",
			SubmitLoadingText:         "Lähetetään...",
			SuccessMessage:            "Pyyntö lähetettiin onnistuneesti.",
			ErrorMessage:              "Pyynnön lähetys epäonnistui. Yritä uudelleen.",

			NameRequiredMessage:         "Kirjoita nimesi.",
			ContactRequiredMessage:      "Anna puhelin tai sähköposti.",
			PhoneInvalidMessage:         "Anna kelvollinen puhelinnumero.",
			EmailInvalidMessage:         "Anna kelvollinen sähköpostiosoite.",
			RegistrationInvalidMessage:  "Anna rekisterinumero muodossa ABC-123.",
			PhotosTooManyMessage:        "Voit lähettää enintään 3 kuvaa.",
			PhotosTypeMessage:           "Vain kuvatiedostot ovat sallittuja.",
			PhotosSizeMessage:           "Yhden kuvan maksimikoko on 10 MB.",
			GeoUnsupportedMessage:       "Selaimesi ei tue sijainnin hakemista.",
			GeoLoadingText:              "Haetaan sijaintia...",
			GeoAddedText:                "Sijainti lisätty",
			GeoFailedMessage:            "Sijainnin haku epäonnistui.",
			MapPickFirstMessage:         "Valitse ensin piste kartalta.",
			MapPointAddedText:           "Karttapiste lisätty",
			ReverseGeocodeFailedMessage: "Koordinaatit tallennettiin, mutta osoitetta ei voitu täyttää automaattisesti.",
			NetworkErrorMessage:         "Verkko- tai selainvirhe. Yritä uudelleen.",
		},
	}
	content.ShowStats = true
	content.ShowServices = true
	content.ShowProcess = true
	content.ShowCases = true
	content.ShowCapability = true
	content.ShowAreas = true
	content.ShowTestimonials = true
	content.ShowFAQ = true
	content.ShowFinalCTA = true

	content.Services.Heading = SectionHeading{
		Kicker: "Palvelut",
		Title:  "Mitä voimme tehdä paikan päällä",
		Text:   "Kirjoitimme palvelut tarkemmin ja lisäsimme hintakoukut muodossa alkaen / tarjous, jotta asiakkaan on helpompi ymmärtää mitä pyynnöllä saa.",
	}
	content.Services.Items = []SimpleCard{
		{Badge: "Suosittu", Price: "alkaen 79 €", Title: "Akun testaus, akun vaihto ja käynnistysongelmat", Text: "Akun kunnon tarkistus, napojen ja kaapeleiden tarkastus sekä ensiarvio siitä, onko kyse akusta, latauksesta vai käynnistyspiiristä."},
		{Badge: "Nopea arvio", Price: fmt.Sprintf("käynti %d €", visitPriceEUR), Title: "Vikakoodien luku ja ensidiagnoosi", Text: "Alustava vianmääritys sisältyy käyntimaksuun. OBD-luku ja laajemmat testit tehdään auton ja tilanteen mukaan."},
		{Badge: "Huolto", Price: "alkaen 89 €", Title: "Jarrujen tarkastus ja kevyet vaihtotyöt", Text: "Jarrujen oirekartoitus, näkyvien osien tarkastus ja mallista riippuvat kevyet vaihtotyöt, jos ne voidaan tehdä turvallisesti paikan päällä."},
		{Badge: "Sähkö / lataus", Price: "alkaen 79 €", Title: "Startti- ja laturiongelmien diagnostiikka", Text: "Jännitteen, latauksen ja käynnistyspiirin tarkastus sekä mallikohtainen arvio siitä, voidaanko osa vaihtaa paikan päällä."},
		{Badge: "Pienet sähköviat", Price: "alkaen 79 €", Title: "Anturien ja pienten sähköosien vaihto", Text: "Yleisten antureiden ja pienten sähköosien vaihto silloin, kun pääsy osaan on järkevä ilman täyttä korjaamokäyntiä."},
		{Badge: "Kiireellinen apu", Price: "tarjouksen mukaan", Title: "Pienet kiirekorjaukset ja tarkastukset", Text: "Kun auto pitää saada nopeasti tarkastettua kotona, työpaikalla tai parkkipaikalla ennen seuraavaa päätöstä."},
	}

	content.Process.Heading = SectionHeading{
		Kicker: "Prosessi",
		Title:  "Miten palvelu toimii",
		Text:   "Selkeä prosessi pyynnöstä korjaukseen.",
	}
	content.Process.Items = []StepCard{
		{Number: "01", Title: "Lähetä pyyntö", Text: "Kerro mitä tapahtui, missä auto on ja yhteystietosi."},
		{Number: "02", Title: "Nopea yhteydenotto", Text: "Käymme ongelman läpi, kysymme tarkennukset ja kerromme seuraavat vaiheet."},
		{Number: "03", Title: fmt.Sprintf("Käynti %d €", visitPriceEUR), Text: "Vahvistamme ajan. Lisätyöt ja varaosat sovitaan erikseen ennen työn aloittamista."},
		{Number: "04", Title: "Korjaus paikan päällä", Text: "Diagnoosi ja soveltuva korjaus tehdään suoraan kohteessa."},
	}

	content.Cases.Heading = SectionHeading{
		Kicker: "Esimerkit",
		Title:  "Tyypillisiä tilanteita, joissa autamme",
		Text:   "Voit nyt lisätä, poistaa ja muokata case-kortteja vapaasti administa ilman kiinteää 4 kortin rajaa.",
	}
	content.Cases.Items = []CaseCard{
		{Tag: "Tapaus 1", Title: "Auto ei käynnisty aamulla", Text: "Tarkistamme akun kunnon, liitännät, latauksen ja todennäköiset syyt.", Items: []string{"Paikan päällä aloitusdiagnoosi", "Akun / latauksen tarkistus", "Suositus seuraavaksi korjaukseksi"}},
		{Tag: "Tapaus 2", Title: "Jarruissa tärinää tai ääntä", Text: "Tarkistamme oireet ja arvioimme, onnistuuko työ paikan päällä.", Items: []string{"Oireiden tarkistus", "Näkyvien osien arvio", "Korjauksen laajuuden selvennys"}},
		{Tag: "Tapaus 3", Title: "Varoitusvalo syttyi", Text: "Vikakoodien luku ja ensitason diagnoosi auttavat löytämään seuraavan korjausaskeleen.", Items: []string{"Perus vikakoodien tarkistus", "Oireiden läpikäynti", "Korjauspolun arvio"}},
		{Tag: "Tapaus 4", Title: "Tarvitset kiireellisen tarkastuksen", Text: "Autamme ymmärtämään tilanteen ja sen, voiko auton korjata paikan päällä vai tarvitseeko korjaamoa.", Items: []string{"Nopea tarkistus", "Tekninen neuvo", "Suositus seuraavaksi vaiheeksi"}},
	}

	content.Testimonials.Heading = SectionHeading{
		Kicker: "Palaute",
		Title:  "Asiakkaiden kokemuksia",
		Text:   "Julkaisemme asiakkaiden lähettämät arvostelut vasta hyväksynnän jälkeen.",
	}
	content.Testimonials.Items = []TestimonialCard{
		{
			Quote:       "Nopea vastaus ja selkeä selitys siitä, mikä oli vialla ja mitä voitiin tehdä heti.",
			Author:      "Asiakas Helsingistä",
			Rating:      5,
			RatingStars: testimonialStars(5),
		},
		{
			Quote:       "Todella kätevä palvelu, kun auto oireili työpaikan parkkipaikalla. Hyvä ensidiagnoosi ja käytännölliset neuvot.",
			Author:      "Asiakas Espoosta",
			Rating:      5,
			RatingStars: testimonialStars(5),
		},
		{
			Quote:       "Ystävällinen palvelu englanniksi ja nopea yhteydenpito.",
			Author:      "Asiakas Vantaalta",
			Rating:      5,
			RatingStars: testimonialStars(5),
		},
	}

	return content
}

func landingContentEN() LandingContent {
	content := LandingContent{
		SiteName: "Liikkuva Auto Korjaamo",
		Header: HeaderContent{
			LogoText: "Liikkuva Auto Korjaamo",
			LogoURL:  "/static/img/logo_liikkuva.svg",
			Nav: []NavItem{
				{Label: "Services", Href: "#services"},
				{Label: "How it works", Href: "#how-it-works"},
				{Label: "Areas", Href: "#areas"},
				{Label: "FAQ", Href: "#faq"},
			},
			LangFI: "FI",
			LangEN: "EN",
			CTA:    "Request service",
		},
		Footer: FooterContent{
			PrivacyLabel: "Privacy",
			Email:        "service@autohelp.fi",
		},
		MapModal: MapModalContent{
			Title:         "Select car location",
			ConfirmButton: "Use this point",
		},
		Hero: HeroContent{
			Badge:        fmt.Sprintf("Mobile car repair • €%d visit", visitPriceEUR),
			Title:        "Premium mobile car repair in Uusimaa",
			Subtitle:     "Diagnostics, light repairs and a clear first estimate at home, work or roadside.",
			PrimaryCTA:   "Get help now",
			SecondaryCTA: "See services",
			Points: []string{
				"Quick reply",
				"On-site diagnostics and repairs",
				"Clear estimate before work starts",
			},
			TrustCards: []SimpleCard{
				{Title: "On-site", Text: "Home, workplace or roadside"},
				{Title: "Transparent", Text: "Clear communication and estimate"},
				{Title: "Local", Text: "Focused on Uusimaa area"},
			},
		},
		Stats: []SimpleCard{
			{Title: fmt.Sprintf("€%d visit", visitPriceEUR), Text: "Includes travel and initial on-site diagnosis"},
			{Title: "On-site repairs", Text: "Many common issues can be checked without towing"},
			{Title: "Clear process", Text: "Diagnosis, estimate, then work"},
			{Title: "8 main cities", Text: "Helsinki, Espoo, Vantaa, Porvoo, Hyvinkää, Järvenpää, Lohja and Kerava"},
		},
		Areas: AreasContent{
			Heading: SectionHeading{
				Kicker: "Coverage",
				Title:  "Service area in Uusimaa",
				Text:   "We show only 8 major cities on the landing page so the design stays clean on smaller screens.",
			},
			Cities:     []string{"Helsinki", "Espoo", "Vantaa", "Porvoo", "Hyvinkää", "Järvenpää", "Lohja", "Kerava"},
			NoteTitle:  "Need help outside the main zone?",
			NoteText:   "Send your exact location in the request form. We review the case first and then confirm availability.",
			NoteButton: "Send request",
		},
		TrustPanel: TrustPanelContent{
			Kicker: "Trust",
			Title:  "The client sees who comes and how the work is handled",
			Text:   "This is the strong trust block next to the form: response speed, language support, payment clarity and service area before the lead is sent.",
			Items:  []string{fmt.Sprintf("Mechanic visit and initial diagnosis €%d", visitPriceEUR), "Possibility of a quick on-site repair", "Extra work and parts are agreed before work begins", "Service in Finnish, English, Russian and Ukrainian"},
		},
		Capability: CapabilityContent{
			Heading:       SectionHeading{Kicker: "On-site / workshop", Title: "What can often be done on-site — and what usually needs a workshop", Text: "This block reduces wrong expectations before the request is sent."},
			CanDo:         CapabilityColumn{Title: "Often possible on-site", Items: []string{"Battery testing and battery replacement", "Fault code reading and first diagnosis", "Basic starting and charging system checks", "Selected sensor and small part replacements", "Brake inspection and model-dependent light work"}},
			NeedsWorkshop: CapabilityColumn{Title: "Usually needs a workshop", Items: []string{"Major engine or gearbox repairs", "Work that needs a lift or specialist equipment", "Large suspension or rust repairs", "Jobs where safe access to parts is not possible on-site", "Cases that need longer disassembly or deeper testing"}},
		},
		FAQ: FAQContent{
			Heading: SectionHeading{Kicker: "FAQ", Title: "Frequently asked questions"},
			Items: []FAQItem{
				{Question: fmt.Sprintf("What does the €%d visit fee include?", visitPriceEUR), Answer: "The visit fee includes travel to the vehicle, an initial on-site diagnosis and the possibility of a quick repair when it can be completed safely right away. Parts and more extensive work are agreed and charged separately before work begins."},
				{Question: "How fast can you respond?", Answer: "It depends on workload, location and issue type. Send a request and we will confirm availability quickly."},
				{Question: "Do you bring tools and parts?", Answer: "Tools are included. Suitable parts depend on the issue and are discussed during request review."},
				{Question: "Can every repair be done on-site?", Answer: "No. Many common issues can be diagnosed or fixed on-site, but some jobs still require a workshop."},
				{Question: "Do you work only in Finnish?", Answer: "Not only in Finnish, the service can be handled in Finnish and English, Russian and Ukrainian"},
			},
		},
		FinalCTA: FinalCTAContent{
			Kicker: "Need help now?",
			Title:  "Send your request and get a quick response",
			Text:   "Describe the problem, your location and the car model. We will review your request as soon as possible.",
			Button: "Request service",
		},
		Form: FormContent{
			Title:                  "Request first estimate",
			Intro:                  "Tell us the issue, location and your contact details. Registration number and photos help us reply faster.",
			LabelName:              "Name",
			LabelPhone:             "Phone",
			LabelEmail:             "Email",
			LabelAddress:           "Address",
			LabelAddressHint:       "Address note",
			LabelCar:               "Registration number",
			LabelPhotos:            "Photos",
			LabelMessage:           "Problem / request",
			PlaceholderName:        "Your name",
			PlaceholderPhone:       "+358... or 040...",
			PlaceholderEmail:       "email@example.com",
			PlaceholderAddress:     "Street and number / yard / parking area",
			PlaceholderAddressHint: "If the address repeats, add city, entrance or landmark",
			PlaceholderCar:         "For example ABC-123",
			PlaceholderMessage:     "Describe the issue",
			UseLocationButton:      "Use my location",
			PickMapButton:          "Pick point on map",
			UploadButtonText:       "Upload photos",
			PhotosHint:             "You can add up to 3 photos. JPG, PNG or WEBP, max 10 MB each.",
			//PhoneHelp:              "Use format +358 40 123 45 67",
			AddressHelp:               "You can type the address manually or fill it from location.",
			CarHelp:                   "Use registration number format ABC-123",
			VisitPriceKicker:          "Clear price before submitting",
			VisitPriceTitle:           fmt.Sprintf("Mechanic visit €%d", visitPriceEUR),
			VisitPriceText:            "Includes travel, an initial diagnosis and the possibility of a quick on-site repair.",
			VisitPriceItems:           []string{"Travel to the vehicle", "Initial on-site diagnosis", "Possibility of a quick repair when it can be completed safely right away"},
			VisitPriceFootnote:        "Parts and more extensive work are not included in the visit fee. We agree them with you before starting work.",
			VisitPriceConsentLabel:    fmt.Sprintf("I understand that the mechanic visit costs €%d.", visitPriceEUR),
			VisitPriceRequiredMessage: fmt.Sprintf("Please confirm that you understand the €%d visit fee.", visitPriceEUR),
			SubmitButton:              "Send request",
			SubmitLoadingText:         "Sending...",
			SuccessMessage:            "Your request has been sent successfully.",
			ErrorMessage:              "Failed to send request. Please try again.",

			NameRequiredMessage:         "Please enter your name.",
			ContactRequiredMessage:      "Please enter phone or email.",
			PhoneInvalidMessage:         "Please enter a valid phone number.",
			EmailInvalidMessage:         "Please enter a valid email address.",
			RegistrationInvalidMessage:  "Please enter a valid registration number in format ABC-123.",
			PhotosTooManyMessage:        "Please upload up to 3 photos only.",
			PhotosTypeMessage:           "Only image files are allowed.",
			PhotosSizeMessage:           "Each photo must be smaller than 10 MB.",
			GeoUnsupportedMessage:       "Geolocation is not supported in this browser.",
			GeoLoadingText:              "Detecting location...",
			GeoAddedText:                "Location added",
			GeoFailedMessage:            "Could not get your location.",
			MapPickFirstMessage:         "Please pick a point on the map first.",
			MapPointAddedText:           "Map point added",
			ReverseGeocodeFailedMessage: "Coordinates were saved, but the address could not be filled automatically.",
			NetworkErrorMessage:         "Network or browser error. Please try again.",
		},
	}
	content.ShowStats = true
	content.ShowServices = true
	content.ShowProcess = true
	content.ShowCases = true
	content.ShowCapability = true
	content.ShowAreas = true
	content.ShowTestimonials = true
	content.ShowFAQ = true
	content.ShowFinalCTA = true

	content.Services.Heading = SectionHeading{
		Kicker: "Services",
		Title:  "What we can do on-site",
		Text:   "We rewrote the service labels to be clearer and added optional price / offer output so the cards can say from € or campaign bonus directly from admin.",
	}
	content.Services.Items = []SimpleCard{
		{Badge: "Popular", Price: "from €79", Title: "Battery testing, battery replacement and no-start problems", Text: "Battery condition check, terminal inspection and first diagnosis of whether the issue comes from the battery, charging system or starting circuit."},
		{Badge: "Quick estimate", Price: fmt.Sprintf("€%d visit", visitPriceEUR), Title: "Fault code reading and first diagnosis", Text: "Initial diagnosis is included in the visit fee. OBD reading and wider tests are completed according to the vehicle and situation."},
		{Badge: "Service", Price: "from €89", Title: "Brake inspection and light brake work", Text: "Brake symptom inspection, visible component check and model-dependent light replacement work when it is safe to do on-site."},
		{Badge: "Electrical / charging", Price: "from €79", Title: "Starter and alternator diagnostics", Text: "Voltage, charging and starting circuit checks with a model-based view on whether replacement work is realistic on-site."},
		{Badge: "Minor electrical", Price: "from €79", Title: "Sensor and small electrical part replacement", Text: "Replacement of common sensors and smaller electrical parts when access is practical without a full workshop visit."},
		{Badge: "Urgent help", Price: "quoted by case", Title: "Urgent small repairs and inspections", Text: "Useful when the car needs to be checked quickly at home, at work or in a parking area before the next decision."},
	}

	content.Process.Heading = SectionHeading{
		Kicker: "Process",
		Title:  "How it works",
		Text:   "Simple and clear process from request to repair.",
	}
	content.Process.Items = []StepCard{
		{Number: "01", Title: "Send request", Text: "Tell us what happened, where the car is, and your contact details."},
		{Number: "02", Title: "Quick contact", Text: "We review the issue, ask clarifying questions, and discuss possible next steps."},
		{Number: "03", Title: fmt.Sprintf("€%d visit", visitPriceEUR), Text: "We confirm the time. Extra work and parts are agreed separately before work begins."},
		{Number: "04", Title: "Repair on-site", Text: "We diagnose and complete suitable repair work directly on location."},
	}

	content.Cases.Heading = SectionHeading{
		Kicker: "Examples",
		Title:  "Typical situations we handle",
		Text:   "Cases can now be added and removed from admin without a fixed four-card limit.",
	}
	content.Cases.Items = []CaseCard{
		{Tag: "Case 1", Title: "Car does not start in the morning", Text: "We check battery condition, connections, charging behavior and likely no-start causes.", Items: []string{"On-site initial diagnosis", "Battery / charging issue review", "Next repair recommendation"}},
		{Tag: "Case 2", Title: "Brake vibration or noise", Text: "We inspect brake-related symptoms and determine whether on-site work is possible.", Items: []string{"Symptom inspection", "Visible component check", "Repair scope clarification"}},
		{Tag: "Case 3", Title: "Warning light turned on", Text: "Fault reading and first-level diagnosis help identify what should be fixed next.", Items: []string{"Basic code check", "Symptom discussion", "Repair path estimate"}},
		{Tag: "Case 4", Title: "Need urgent inspection before service", Text: "We help understand the issue and whether the car can be repaired on-site or needs workshop work.", Items: []string{"Quick review", "Technical advice", "Next-step recommendation"}},
	}

	content.Testimonials.Heading = SectionHeading{
		Kicker: "Feedback",
		Title:  "Customer experiences",
		Text:   "We publish customer-submitted reviews only after approval.",
	}
	content.Testimonials.Items = []TestimonialCard{
		{
			Quote:       "Fast response and clear explanation of what was wrong and what could be done immediately.",
			Author:      "Client from Helsinki",
			Rating:      5,
			RatingStars: testimonialStars(5),
		},
		{
			Quote:       "Very convenient when the car had issues at work parking. Good first diagnosis and practical advice.",
			Author:      "Client from Espoo",
			Rating:      5,
			RatingStars: testimonialStars(5),
		},
		{
			Quote:       "Helpful service in English and quick communication.",
			Author:      "Client from Vantaa",
			Rating:      5,
			RatingStars: testimonialStars(5),
		},
	}

	return content
}

func applySiteSettingsToContent(content LandingContent, settings SiteSettings, lang string) LandingContent {
	content.Footer.Phone = strings.TrimSpace(settings.Phone)
	content.Footer.PhoneHref = buildPhoneHref(settings.Phone)
	content.Footer.WhatsAppHref = buildWhatsAppHref(firstNonEmpty(strings.TrimSpace(settings.WhatsAppNumber), strings.TrimSpace(settings.Phone)))
	content.Footer.Email = firstNonEmpty(strings.TrimSpace(settings.ServiceEmail), content.Footer.Email)
	content.Footer.BusinessName = strings.TrimSpace(settings.BusinessName)
	content.Footer.YTunnus = strings.TrimSpace(settings.YTunnus)
	content.Footer.BaseCity = strings.TrimSpace(settings.BaseCity)
	content.Footer.InstagramURL = strings.TrimSpace(settings.InstagramURL)
	content.Footer.TikTokURL = strings.TrimSpace(settings.TikTokURL)
	content.Footer.FacebookGroupURL = strings.TrimSpace(settings.FacebookGroupURL)
	applySectionVisibility(&content, settings)
	applyStringOverride(&content.Header.LogoURL, settings.LogoURL)
	applyStringOverride(&content.Hero.PersonDesktopURL, settings.HeroPersonDesktopURL)
	applyStringOverride(&content.Hero.BackgroundURL, settings.HeroBackgroundURL)
	applyStringOverride(&content.Areas.MapImageURL, settings.AreasMapImageURL)
	if len(content.Services.Items) > 0 {
		content.Services.Items[0].ImageURL = strings.TrimSpace(settings.ServicesImage1URL)
	}
	if len(content.Services.Items) > 1 {
		content.Services.Items[1].ImageURL = strings.TrimSpace(settings.ServicesImage2URL)
	}
	if len(content.Services.Items) > 2 {
		content.Services.Items[2].ImageURL = strings.TrimSpace(settings.ServicesImage3URL)
	}
	if len(content.Services.Items) > 3 {
		content.Services.Items[3].ImageURL = strings.TrimSpace(settings.ServicesImage4URL)
	}
	if len(content.Services.Items) > 4 {
		content.Services.Items[4].ImageURL = strings.TrimSpace(settings.ServicesImage5URL)
	}
	if len(content.Services.Items) > 5 {
		content.Services.Items[5].ImageURL = strings.TrimSpace(settings.ServicesImage6URL)
	}
	if items := buildServicesFromSettings(settings, lang); len(items) > 0 {
		content.Services.Items = items
	}

	if items := buildCasesFromSettings(settings, lang); len(items) > 0 {
		content.Cases.Items = items
	}

	if items := buildTestimonialsFromSettings(settings, lang); len(items) > 0 {
		content.Testimonials.Items = items
	}

	content.Promo.Enabled = settings.PromoEnabled
	content.Promo.URL = strings.TrimSpace(settings.PromoURL)
	if content.Promo.URL == "" {
		content.Promo.URL = "#lead-form"
	}

	if lang == "fi" {
		applyStringOverride(&content.Hero.PersonAlt, settings.HeroPersonAltFI)
		content.Footer.WorkingHours = firstNonEmpty(strings.TrimSpace(settings.WorkingHoursFI), content.Footer.WorkingHours)
		applyStringOverride(&content.TrustPanel.Kicker, settings.TrustKickerFI)
		applyStringOverride(&content.TrustPanel.Title, settings.TrustTitleFI)
		applyStringOverride(&content.TrustPanel.Text, settings.TrustTextFI)
		if items := splitMultilineSetting(settings.TrustItemsFI); len(items) > 0 {
			content.TrustPanel.Items = items
		}
		applyStringOverride(&content.Capability.Heading.Kicker, settings.CapabilityKickerFI)
		applyStringOverride(&content.Capability.Heading.Title, settings.CapabilityTitleFI)
		applyStringOverride(&content.Capability.Heading.Text, settings.CapabilityTextFI)
		applyStringOverride(&content.Capability.CanDo.Title, settings.CapabilityCanDoTitleFI)
		if items := splitMultilineSetting(settings.CapabilityCanDoItemsFI); len(items) > 0 {
			content.Capability.CanDo.Items = items
		}
		applyStringOverride(&content.Capability.NeedsWorkshop.Title, settings.CapabilityNeedsWorkshopTitleFI)
		if items := splitMultilineSetting(settings.CapabilityNeedsWorkshopItemsFI); len(items) > 0 {
			content.Capability.NeedsWorkshop.Items = items
		}
		applyHeroBadgePointsAndCards(&content, settings, lang)
		applyStringOverride(&content.Header.CTA, settings.HeaderCTAFI)
		applyStringOverride(&content.Hero.Title, settings.HeroTitleFI)
		applyStringOverride(&content.Hero.Subtitle, settings.HeroSubtitleFI)
		applyStringOverride(&content.Hero.PrimaryCTA, settings.PrimaryCTAFI)
		applyStringOverride(&content.Hero.SecondaryCTA, settings.SecondaryCTAFI)
		applyStringOverride(&content.Services.Heading.Kicker, settings.ServicesKickerFI)
		applyStringOverride(&content.Services.Heading.Title, settings.ServicesTitleFI)
		applyStringOverride(&content.Services.Heading.Text, settings.ServicesTextFI)

		applyStringOverride(&content.Process.Heading.Kicker, settings.ProcessKickerFI)
		applyStringOverride(&content.Process.Heading.Title, settings.ProcessTitleFI)
		applyStringOverride(&content.Process.Heading.Text, settings.ProcessTextFI)

		applyStringOverride(&content.FinalCTA.Kicker, settings.FinalCTAKickerFI)
		applyStringOverride(&content.FinalCTA.Title, settings.FinalCTATitleFI)
		applyStringOverride(&content.FinalCTA.Text, settings.FinalCTATextFI)
		applyStringOverride(&content.FinalCTA.Button, settings.FinalCTAButtonFI)

		applyStringOverride(&content.Form.Title, settings.FormTitleFI)
		applyStringOverride(&content.Form.Intro, settings.FormIntroFI)
		applyStringOverride(&content.Form.LabelName, settings.FormLabelNameFI)
		applyStringOverride(&content.Form.LabelPhone, settings.FormLabelPhoneFI)
		applyStringOverride(&content.Form.LabelEmail, settings.FormLabelEmailFI)
		applyStringOverride(&content.Form.LabelAddress, settings.FormLabelAddressFI)
		applyStringOverride(&content.Form.LabelCar, settings.FormLabelCarFI)
		applyStringOverride(&content.Form.LabelPhotos, settings.FormLabelPhotosFI)
		applyStringOverride(&content.Form.LabelMessage, settings.FormLabelMessageFI)
		applyStringOverride(&content.Form.PlaceholderName, settings.FormPlaceholderNameFI)
		applyStringOverride(&content.Form.PlaceholderPhone, settings.FormPlaceholderPhoneFI)
		applyStringOverride(&content.Form.PlaceholderEmail, settings.FormPlaceholderEmailFI)
		applyStringOverride(&content.Form.PlaceholderAddress, settings.FormPlaceholderAddressFI)
		applyStringOverride(&content.Form.PlaceholderCar, settings.FormPlaceholderCarFI)
		applyStringOverride(&content.Form.PlaceholderMessage, settings.FormPlaceholderMessageFI)
		applyStringOverride(&content.Form.UseLocationButton, settings.FormUseLocationButtonFI)
		applyStringOverride(&content.Form.PickMapButton, settings.FormPickMapButtonFI)
		applyStringOverride(&content.Form.UploadButtonText, settings.FormUploadButtonTextFI)
		applyStringOverride(&content.Form.PhotosHint, settings.FormPhotosHintFI)
		applyStringOverride(&content.Form.PhoneHelp, settings.FormPhoneHelpFI)
		applyStringOverride(&content.Form.AddressHelp, settings.FormAddressHelpFI)
		applyStringOverride(&content.Form.CarHelp, settings.FormCarHelpFI)
		applyStringOverride(&content.Form.SubmitButton, settings.FormSubmitButtonFI)
		applyStringOverride(&content.Form.SubmitLoadingText, settings.FormSubmitLoadingTextFI)
		applyStringOverride(&content.Form.SuccessMessage, settings.FormSuccessMessageFI)
		applyStringOverride(&content.Form.ErrorMessage, settings.FormErrorMessageFI)
		applyStringOverride(&content.Form.NameRequiredMessage, settings.FormNameRequiredFI)
		applyStringOverride(&content.Form.ContactRequiredMessage, settings.FormContactRequiredFI)
		applyStringOverride(&content.Form.PhoneInvalidMessage, settings.FormPhoneInvalidFI)
		applyStringOverride(&content.Form.EmailInvalidMessage, settings.FormEmailInvalidFI)
		applyStringOverride(&content.Form.RegistrationInvalidMessage, settings.FormRegistrationInvalidFI)
		applyStringOverride(&content.Form.PhotosTooManyMessage, settings.FormPhotosTooManyFI)
		applyStringOverride(&content.Form.PhotosTypeMessage, settings.FormPhotosTypeFI)
		applyStringOverride(&content.Form.PhotosSizeMessage, settings.FormPhotosSizeFI)
		applyStringOverride(&content.Form.GeoUnsupportedMessage, settings.FormGeoUnsupportedFI)
		applyStringOverride(&content.Form.GeoLoadingText, settings.FormGeoLoadingFI)
		applyStringOverride(&content.Form.GeoAddedText, settings.FormGeoAddedFI)
		applyStringOverride(&content.Form.GeoFailedMessage, settings.FormGeoFailedFI)
		applyStringOverride(&content.Form.MapPickFirstMessage, settings.FormMapPickFirstFI)
		applyStringOverride(&content.Form.MapPointAddedText, settings.FormMapPointAddedFI)
		applyStringOverride(&content.Form.ReverseGeocodeFailedMessage, settings.FormReverseGeocodeFailedFI)
		applyStringOverride(&content.Form.NetworkErrorMessage, settings.FormNetworkErrorFI)
		applyStringOverride(&content.Promo.Badge, settings.PromoBadgeFI)
		applyStringOverride(&content.Promo.Title, settings.PromoTitleFI)
		applyStringOverride(&content.Promo.Text, settings.PromoTextFI)
		applyStringOverride(&content.Promo.Button, settings.PromoButtonFI)
		return content
	}

	applyStringOverride(&content.Hero.PersonAlt, settings.HeroPersonAltEN)
	content.Footer.WorkingHours = firstNonEmpty(strings.TrimSpace(settings.WorkingHoursEN), content.Footer.WorkingHours)
	applyStringOverride(&content.TrustPanel.Kicker, settings.TrustKickerEN)
	applyStringOverride(&content.TrustPanel.Title, settings.TrustTitleEN)
	applyStringOverride(&content.TrustPanel.Text, settings.TrustTextEN)
	if items := splitMultilineSetting(settings.TrustItemsEN); len(items) > 0 {
		content.TrustPanel.Items = items
	}
	applyStringOverride(&content.Capability.Heading.Kicker, settings.CapabilityKickerEN)
	applyStringOverride(&content.Capability.Heading.Title, settings.CapabilityTitleEN)
	applyStringOverride(&content.Capability.Heading.Text, settings.CapabilityTextEN)
	applyStringOverride(&content.Capability.CanDo.Title, settings.CapabilityCanDoTitleEN)
	if items := splitMultilineSetting(settings.CapabilityCanDoItemsEN); len(items) > 0 {
		content.Capability.CanDo.Items = items
	}
	applyStringOverride(&content.Capability.NeedsWorkshop.Title, settings.CapabilityNeedsWorkshopTitleEN)
	if items := splitMultilineSetting(settings.CapabilityNeedsWorkshopItemsEN); len(items) > 0 {
		content.Capability.NeedsWorkshop.Items = items
	}
	applyHeroBadgePointsAndCards(&content, settings, lang)
	applyStringOverride(&content.Header.CTA, settings.HeaderCTAEN)
	applyStringOverride(&content.Hero.Title, settings.HeroTitleEN)
	applyStringOverride(&content.Hero.Subtitle, settings.HeroSubtitleEN)
	applyStringOverride(&content.Hero.PrimaryCTA, settings.PrimaryCTAEN)
	applyStringOverride(&content.Hero.SecondaryCTA, settings.SecondaryCTAEN)
	applyStringOverride(&content.Services.Heading.Kicker, settings.ServicesKickerEN)
	applyStringOverride(&content.Services.Heading.Title, settings.ServicesTitleEN)
	applyStringOverride(&content.Services.Heading.Text, settings.ServicesTextEN)

	applyStringOverride(&content.Process.Heading.Kicker, settings.ProcessKickerEN)
	applyStringOverride(&content.Process.Heading.Title, settings.ProcessTitleEN)
	applyStringOverride(&content.Process.Heading.Text, settings.ProcessTextEN)

	applyStringOverride(&content.FinalCTA.Kicker, settings.FinalCTAKickerEN)
	applyStringOverride(&content.FinalCTA.Title, settings.FinalCTATitleEN)
	applyStringOverride(&content.FinalCTA.Text, settings.FinalCTATextEN)
	applyStringOverride(&content.FinalCTA.Button, settings.FinalCTAButtonEN)

	applyStringOverride(&content.Form.Title, settings.FormTitleEN)
	applyStringOverride(&content.Form.Intro, settings.FormIntroEN)
	applyStringOverride(&content.Form.LabelName, settings.FormLabelNameEN)
	applyStringOverride(&content.Form.LabelPhone, settings.FormLabelPhoneEN)
	applyStringOverride(&content.Form.LabelEmail, settings.FormLabelEmailEN)
	applyStringOverride(&content.Form.LabelAddress, settings.FormLabelAddressEN)
	applyStringOverride(&content.Form.LabelCar, settings.FormLabelCarEN)
	applyStringOverride(&content.Form.LabelPhotos, settings.FormLabelPhotosEN)
	applyStringOverride(&content.Form.LabelMessage, settings.FormLabelMessageEN)
	applyStringOverride(&content.Form.PlaceholderName, settings.FormPlaceholderNameEN)
	applyStringOverride(&content.Form.PlaceholderPhone, settings.FormPlaceholderPhoneEN)
	applyStringOverride(&content.Form.PlaceholderEmail, settings.FormPlaceholderEmailEN)
	applyStringOverride(&content.Form.PlaceholderAddress, settings.FormPlaceholderAddressEN)
	applyStringOverride(&content.Form.PlaceholderCar, settings.FormPlaceholderCarEN)
	applyStringOverride(&content.Form.PlaceholderMessage, settings.FormPlaceholderMessageEN)
	applyStringOverride(&content.Form.UseLocationButton, settings.FormUseLocationButtonEN)
	applyStringOverride(&content.Form.PickMapButton, settings.FormPickMapButtonEN)
	applyStringOverride(&content.Form.UploadButtonText, settings.FormUploadButtonTextEN)
	applyStringOverride(&content.Form.PhotosHint, settings.FormPhotosHintEN)
	applyStringOverride(&content.Form.PhoneHelp, settings.FormPhoneHelpEN)
	applyStringOverride(&content.Form.AddressHelp, settings.FormAddressHelpEN)
	applyStringOverride(&content.Form.CarHelp, settings.FormCarHelpEN)
	applyStringOverride(&content.Form.SubmitButton, settings.FormSubmitButtonEN)
	applyStringOverride(&content.Form.SubmitLoadingText, settings.FormSubmitLoadingTextEN)
	applyStringOverride(&content.Form.SuccessMessage, settings.FormSuccessMessageEN)
	applyStringOverride(&content.Form.ErrorMessage, settings.FormErrorMessageEN)
	applyStringOverride(&content.Form.NameRequiredMessage, settings.FormNameRequiredEN)
	applyStringOverride(&content.Form.ContactRequiredMessage, settings.FormContactRequiredEN)
	applyStringOverride(&content.Form.PhoneInvalidMessage, settings.FormPhoneInvalidEN)
	applyStringOverride(&content.Form.EmailInvalidMessage, settings.FormEmailInvalidEN)
	applyStringOverride(&content.Form.RegistrationInvalidMessage, settings.FormRegistrationInvalidEN)
	applyStringOverride(&content.Form.PhotosTooManyMessage, settings.FormPhotosTooManyEN)
	applyStringOverride(&content.Form.PhotosTypeMessage, settings.FormPhotosTypeEN)
	applyStringOverride(&content.Form.PhotosSizeMessage, settings.FormPhotosSizeEN)
	applyStringOverride(&content.Form.GeoUnsupportedMessage, settings.FormGeoUnsupportedEN)
	applyStringOverride(&content.Form.GeoLoadingText, settings.FormGeoLoadingEN)
	applyStringOverride(&content.Form.GeoAddedText, settings.FormGeoAddedEN)
	applyStringOverride(&content.Form.GeoFailedMessage, settings.FormGeoFailedEN)
	applyStringOverride(&content.Form.MapPickFirstMessage, settings.FormMapPickFirstEN)
	applyStringOverride(&content.Form.MapPointAddedText, settings.FormMapPointAddedEN)
	applyStringOverride(&content.Form.ReverseGeocodeFailedMessage, settings.FormReverseGeocodeFailedEN)
	applyStringOverride(&content.Form.NetworkErrorMessage, settings.FormNetworkErrorEN)
	applyStringOverride(&content.Promo.Badge, settings.PromoBadgeEN)
	applyStringOverride(&content.Promo.Title, settings.PromoTitleEN)
	applyStringOverride(&content.Promo.Text, settings.PromoTextEN)
	applyStringOverride(&content.Promo.Button, settings.PromoButtonEN)

	return content
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func splitMultilineSetting(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	rows := strings.Split(value, "\n")
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row != "" {
			out = append(out, row)
		}
	}
	return out
}

func applyStringOverride(target *string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	*target = value
}

func applySectionVisibility(content *LandingContent, settings SiteSettings) {
	content.ShowStats = false
	content.ShowServices = true
	content.ShowProcess = true
	content.ShowCases = true
	content.ShowCapability = true
	content.ShowAreas = true
	content.ShowTestimonials = true
	content.ShowFAQ = true
	content.ShowFinalCTA = true

	if !settings.SectionVisibilityConfigured {
		return
	}

	// Safety fallback:
	// if all section flags are false, treat it as a broken old admin save
	// and keep all sections visible.
	if !settings.StatsEnabled &&
		!settings.ServicesEnabled &&
		!settings.ProcessEnabled &&
		!settings.CasesEnabled &&
		!settings.CapabilityEnabled &&
		!settings.AreasEnabled &&
		!settings.TestimonialsEnabled &&
		!settings.FAQEnabled &&
		!settings.FinalCTAEnabled {
		return
	}

	content.ShowStats = settings.StatsEnabled
	content.ShowServices = settings.ServicesEnabled
	content.ShowProcess = settings.ProcessEnabled
	content.ShowCases = settings.CasesEnabled
	content.ShowCapability = settings.CapabilityEnabled
	content.ShowAreas = settings.AreasEnabled
	content.ShowTestimonials = settings.TestimonialsEnabled
	content.ShowFAQ = settings.FAQEnabled
	content.ShowFinalCTA = settings.FinalCTAEnabled
}

func applyHeroBadgePointsAndCards(content *LandingContent, settings SiteSettings, lang string) {
	if cities := splitMultilineSetting(settings.CoverageCities); len(cities) > 0 {
		content.Areas.Cities = cities
	}

	var badge string
	var points string
	var cardTitles []string
	var cardTexts []string

	if lang == "fi" {
		badge = settings.HeroBadgeFI
		points = settings.HeroPointsFI
		cardTitles = []string{
			settings.HeroTrustCard1TitleFI,
			settings.HeroTrustCard2TitleFI,
			settings.HeroTrustCard3TitleFI,
		}
		cardTexts = []string{
			settings.HeroTrustCard1TextFI,
			settings.HeroTrustCard2TextFI,
			settings.HeroTrustCard3TextFI,
		}
	} else {
		badge = settings.HeroBadgeEN
		points = settings.HeroPointsEN
		cardTitles = []string{
			settings.HeroTrustCard1TitleEN,
			settings.HeroTrustCard2TitleEN,
			settings.HeroTrustCard3TitleEN,
		}
		cardTexts = []string{
			settings.HeroTrustCard1TextEN,
			settings.HeroTrustCard2TextEN,
			settings.HeroTrustCard3TextEN,
		}
	}

	applyStringOverride(&content.Hero.Badge, badge)
	if items := splitMultilineSetting(points); len(items) > 0 {
		content.Hero.Points = items
	}

	for len(content.Hero.TrustCards) < 3 {
		content.Hero.TrustCards = append(content.Hero.TrustCards, SimpleCard{})
	}

	for i := 0; i < 3; i++ {
		applyStringOverride(&content.Hero.TrustCards[i].Title, cardTitles[i])
		applyStringOverride(&content.Hero.TrustCards[i].Text, cardTexts[i])
	}
}

func filterNonEmptyStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func normalizeTestimonialRatingValue(value int) int {
	if value < 1 || value > 5 {
		return 5
	}
	return value
}

func testimonialStars(value int) string {
	value = normalizeTestimonialRatingValue(value)
	return strings.Repeat("★", value) + strings.Repeat("☆", 5-value)
}

func buildTestimonialsFromSettings(settings SiteSettings, lang string) []TestimonialCard {
	if len(settings.Testimonials) == 0 {
		return nil
	}

	out := make([]TestimonialCard, 0, len(settings.Testimonials))
	for _, item := range settings.Testimonials {
		author := strings.TrimSpace(item.Name)
		if author == "" {
			continue
		}

		quote := strings.TrimSpace(item.QuoteFI)
		if lang == "en" {
			quote = strings.TrimSpace(item.QuoteEN)
		}

		if quote == "" {
			continue
		}

		rating := normalizeTestimonialRatingValue(item.Rating)

		out = append(out, TestimonialCard{
			Quote:       quote,
			Author:      author,
			ImageURL:    strings.TrimSpace(item.PhotoURL),
			Rating:      rating,
			RatingStars: testimonialStars(rating),
		})
	}

	return out
}

func buildServicesFromSettings(settings SiteSettings, lang string) []SimpleCard {
	if len(settings.ServiceItems) == 0 {
		return nil
	}

	out := make([]SimpleCard, 0, len(settings.ServiceItems))
	for _, item := range settings.ServiceItems {
		title := strings.TrimSpace(item.TitleFI)
		text := strings.TrimSpace(item.TextFI)

		if lang == "en" {
			title = strings.TrimSpace(item.TitleEN)
			text = strings.TrimSpace(item.TextEN)
		}

		if title == "" || text == "" {
			continue
		}

		badge := strings.TrimSpace(item.BadgeFI)
		price := strings.TrimSpace(item.PriceFI)

		if lang == "en" {
			badge = strings.TrimSpace(item.BadgeEN)
			price = strings.TrimSpace(item.PriceEN)
		}

		out = append(out, SimpleCard{
			Title:    title,
			Text:     text,
			ImageURL: strings.TrimSpace(item.ImageURL),
			Badge:    badge,
			Price:    price,
		})
	}

	return out
}

func buildCasesFromSettings(settings SiteSettings, lang string) []CaseCard {
	if len(settings.CaseItems) == 0 {
		return nil
	}

	out := make([]CaseCard, 0, len(settings.CaseItems))
	for _, item := range settings.CaseItems {
		tag := strings.TrimSpace(item.TagFI)
		title := strings.TrimSpace(item.TitleFI)
		text := strings.TrimSpace(item.TextFI)
		points := filterNonEmptyStrings(item.PointsFI)

		if lang == "en" {
			tag = strings.TrimSpace(item.TagEN)
			title = strings.TrimSpace(item.TitleEN)
			text = strings.TrimSpace(item.TextEN)
			points = filterNonEmptyStrings(item.PointsEN)
		}

		if title == "" || text == "" {
			continue
		}

		out = append(out, CaseCard{
			Tag:   tag,
			Title: title,
			Text:  text,
			Items: points,
		})
	}

	return out
}

func buildPhoneHref(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		" ", "",
		"-", "",
		"(", "",
		")", "",
	)

	return "tel:" + replacer.Replace(phone)
}

func buildWhatsAppHref(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	cleaned := strings.NewReplacer(
		" ", "",
		"-", "",
		"(", "",
		")", "",
		"+", "",
	).Replace(value)

	if strings.HasPrefix(cleaned, "00") {
		cleaned = strings.TrimPrefix(cleaned, "00")
	}
	if strings.HasPrefix(cleaned, "0") {
		cleaned = "358" + strings.TrimPrefix(cleaned, "0")
	}

	var digitsOnly strings.Builder
	for _, ch := range cleaned {
		if ch >= '0' && ch <= '9' {
			digitsOnly.WriteRune(ch)
		}
	}

	digits := digitsOnly.String()
	if digits == "" {
		return ""
	}

	return "https://wa.me/" + digits
}

func applyLandingPageSEO(pageTitle, metaDescription, robotsMeta string, settings SiteSettings, lang string) (string, string, string) {
	if lang == "fi" {
		applyStringOverride(&pageTitle, settings.LandingSEOTitleFI)
		applyStringOverride(&metaDescription, settings.LandingSEODescriptionFI)
	} else {
		applyStringOverride(&pageTitle, settings.LandingSEOTitleEN)
		applyStringOverride(&metaDescription, settings.LandingSEODescriptionEN)
	}

	applyStringOverride(&robotsMeta, settings.LandingRobotsMeta)
	return pageTitle, metaDescription, robotsMeta
}

func applyThankYouSettingsToTexts(texts map[string]string, settings SiteSettings, lang string) map[string]string {
	out := cloneStringMap(texts)
	if lang == "fi" {
		applyMapOverride(out, "PAGE_TITLE", settings.ThankYouSEOTitleFI)
		applyMapOverride(out, "META_DESCRIPTION", settings.ThankYouSEODescriptionFI)
		applyMapOverride(out, "THANKYOU_TITLE", settings.ThankYouTitleFI)
		applyMapOverride(out, "THANKYOU_TEXT", settings.ThankYouTextFI)
		applyMapOverride(out, "THANKYOU_BUTTON", settings.ThankYouButtonFI)
		applyMapOverride(out, "THANKYOU_BUTTON_URL", settings.ThankYouButtonURLFI)
	} else {
		applyMapOverride(out, "PAGE_TITLE", settings.ThankYouSEOTitleEN)
		applyMapOverride(out, "META_DESCRIPTION", settings.ThankYouSEODescriptionEN)
		applyMapOverride(out, "THANKYOU_TITLE", settings.ThankYouTitleEN)
		applyMapOverride(out, "THANKYOU_TEXT", settings.ThankYouTextEN)
		applyMapOverride(out, "THANKYOU_BUTTON", settings.ThankYouButtonEN)
		applyMapOverride(out, "THANKYOU_BUTTON_URL", settings.ThankYouButtonURLEN)
	}

	applyMapOverride(out, "ROBOTS_META", settings.ThankYouRobotsMeta)
	return out
}

func applyPrivacySettingsToTexts(texts map[string]string, settings SiteSettings, lang string) map[string]string {
	out := cloneStringMap(texts)
	if lang == "fi" {
		applyMapOverride(out, "PAGE_TITLE", settings.PrivacySEOTitleFI)
		applyMapOverride(out, "META_DESCRIPTION", settings.PrivacySEODescriptionFI)
		applyMapOverride(out, "PRIVACY_TITLE", settings.PrivacyTitleFI)
		applyMapOverride(out, "PRIVACY_INTRO", settings.PrivacyIntroFI)
		applyMapOverride(out, "SECTION_1_TITLE", settings.PrivacySection1TitleFI)
		applyMapOverride(out, "SECTION_1_TEXT", settings.PrivacySection1TextFI)
		applyMapOverride(out, "SECTION_2_TITLE", settings.PrivacySection2TitleFI)
		applyMapOverride(out, "SECTION_2_TEXT", settings.PrivacySection2TextFI)
		applyMapOverride(out, "SECTION_3_TITLE", settings.PrivacySection3TitleFI)
		applyMapOverride(out, "SECTION_3_TEXT", settings.PrivacySection3TextFI)
		applyMapOverride(out, "SECTION_4_TITLE", settings.PrivacySection4TitleFI)
		applyMapOverride(out, "SECTION_4_TEXT", settings.PrivacySection4TextFI)
		applyMapOverride(out, "SECTION_5_TITLE", settings.PrivacySection5TitleFI)
		applyMapOverride(out, "SECTION_5_TEXT", settings.PrivacySection5TextFI)
	} else {
		applyMapOverride(out, "PAGE_TITLE", settings.PrivacySEOTitleEN)
		applyMapOverride(out, "META_DESCRIPTION", settings.PrivacySEODescriptionEN)
		applyMapOverride(out, "PRIVACY_TITLE", settings.PrivacyTitleEN)
		applyMapOverride(out, "PRIVACY_INTRO", settings.PrivacyIntroEN)
		applyMapOverride(out, "SECTION_1_TITLE", settings.PrivacySection1TitleEN)
		applyMapOverride(out, "SECTION_1_TEXT", settings.PrivacySection1TextEN)
		applyMapOverride(out, "SECTION_2_TITLE", settings.PrivacySection2TitleEN)
		applyMapOverride(out, "SECTION_2_TEXT", settings.PrivacySection2TextEN)
		applyMapOverride(out, "SECTION_3_TITLE", settings.PrivacySection3TitleEN)
		applyMapOverride(out, "SECTION_3_TEXT", settings.PrivacySection3TextEN)
		applyMapOverride(out, "SECTION_4_TITLE", settings.PrivacySection4TitleEN)
		applyMapOverride(out, "SECTION_4_TEXT", settings.PrivacySection4TextEN)
		applyMapOverride(out, "SECTION_5_TITLE", settings.PrivacySection5TitleEN)
		applyMapOverride(out, "SECTION_5_TEXT", settings.PrivacySection5TextEN)
	}

	applyMapOverride(out, "ROBOTS_META", settings.PrivacyRobotsMeta)
	return out
}

func applyMapOverride(target map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	target[key] = value
}

func cloneStringMap(in map[string]string) map[string]string {

	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func getThankYouTexts(lang string) map[string]string {
	if lang == "fi" {
		return map[string]string{
			"LANG":                "fi",
			"PAGE_TITLE":          "Kiitos pyynnöstäsi | Liikkuva Auto Korjaamo",
			"META_DESCRIPTION":    "Kiitos yhteydenotostasi. Olemme vastaanottaneet pyyntösi ja palaamme sinulle pian.",
			"ROBOTS_META":         "noindex, follow",
			"THANKYOU_TITLE":      "Kiitos pyynnöstäsi",
			"THANKYOU_TEXT":       fmt.Sprintf("Olemme vastaanottaneet pyyntösi. Asentajan käynti maksaa %d €. Odota viestiämme — ilmoitamme, kun asentaja on valmis lähtemään luoksesi.", visitPriceEUR),
			"THANKYOU_BUTTON":     "Takaisin etusivulle",
			"THANKYOU_BUTTON_URL": "/",
		}
	}

	return map[string]string{
		"LANG":                "en",
		"PAGE_TITLE":          "Thank you for your request | Liikkuva Auto Korjaamo",
		"META_DESCRIPTION":    "Thank you for contacting us. We have received your request and will get back to you soon.",
		"ROBOTS_META":         "noindex, follow",
		"THANKYOU_TITLE":      "Thank you for your request",
		"THANKYOU_TEXT":       fmt.Sprintf("We have received your request. The mechanic visit costs €%d. Please wait for our message confirming when the mechanic is ready to leave for your location.", visitPriceEUR),
		"THANKYOU_BUTTON":     "Back to homepage",
		"THANKYOU_BUTTON_URL": "/en",
	}
}

func getPrivacyTexts(lang string) map[string]string {
	if lang == "fi" {
		return map[string]string{
			"LANG":             "fi",
			"PAGE_TITLE":       "Tietosuojaseloste | Liikkuva Auto Korjaamo",
			"META_DESCRIPTION": "Tietoa siitä, miten Liikkuva Auto Korjaamo käsittelee yhteydenottolomakkeen kautta lähetettyjä tietoja.",
			"ROBOTS_META":      "index, follow",
			"PRIVACY_TITLE":    "Tietosuojaseloste",
			"PRIVACY_INTRO":    "Tällä sivulla kerromme, miten käsittelemme yhteydenotto- ja arvostelulomakkeiden kautta lähetettyjä henkilötietoja.",
			"SECTION_1_TITLE":  "1. Mitä tietoja keräämme",
			"SECTION_1_TEXT":   "Voimme kerätä nimen, puhelinnumeron, sähköpostiosoitteen, osoitteen, osoitteen tarkennuksen, auton tiedot, lomakkeessa annetun viestin, käyntimaksun hyväksynnän, sijaintikoordinaatit ja mahdolliset kuvat. Arvostelusta tallennetaan nimi, tähtiarvio, teksti, kieli ja julkaisulupa.",
			"SECTION_2_TITLE":  "2. Mihin tietoja käytetään",
			"SECTION_2_TEXT":   "Tietoja käytetään yhteydenottoon, palvelupyynnön käsittelyyn, vahvistus- ja arvostelupyyntöviestien lähettämiseen, ajan sopimiseen, asiakaspalveluun sekä hyväksyttyjen arvostelujen julkaisemiseen.",
			"SECTION_3_TITLE":  "3. Tekninen seuranta",
			"SECTION_3_TEXT":   "Voimme tallentaa teknisiä tietoja kuten sivun URL-osoitteen, viittaavan sivun, UTM-parametrit, IP-osoitteen ja selaintiedot palvelupyynnön lähteen analysointiin.",
			"SECTION_4_TITLE":  "4. Tietojen säilytys",
			"SECTION_4_TEXT":   "Tietoja voidaan säilyttää Firestoressa palvelupyyntöjen ja arvostelujen hallintaa varten, Telegram-ilmoituksissa, pilvitallennuksessa kuvia varten ja Google Sheets -taulukossa.",
			"SECTION_5_TITLE":  "5. Yhteystiedot",
			"SECTION_5_TEXT":   "Jos haluat kysyä tietojesi käsittelystä, voit ottaa meihin yhteyttä saman lomakkeen kautta.",
		}
	}

	return map[string]string{
		"LANG":             "en",
		"PAGE_TITLE":       "Privacy Policy | Liikkuva Auto Korjaamo",
		"META_DESCRIPTION": "Information about how Liikkuva Auto Korjaamo processes data submitted through the contact form.",
		"PRIVACY_TITLE":    "Privacy Policy",
		"PRIVACY_INTRO":    "This page explains how we process personal data submitted through the request and review forms.",
		"SECTION_1_TITLE":  "1. What data we collect",
		"SECTION_1_TEXT":   "We may collect your name, phone number, email address, address, address note, car details, submitted message, visit-fee acceptance, location coordinates and uploaded photos. A review stores the name, star rating, message, language and publication consent.",
		"SECTION_2_TITLE":  "2. How we use the data",
		"SECTION_2_TEXT":   "The data is used to contact you, process the service request, send confirmation and review-request messages, arrange a visit, provide support and publish approved reviews.",
		"SECTION_3_TITLE":  "3. Technical tracking",
		"SECTION_3_TEXT":   "We may store technical data such as page URL, referrer, UTM parameters, IP address and browser details for service request source analysis.",
		"SECTION_4_TITLE":  "4. Data storage",
		"SECTION_4_TEXT":   "The data may be stored in Firestore for service request and review management, in Telegram notifications, in cloud storage for uploaded photos, and in Google Sheets.",
		"SECTION_5_TITLE":  "5. Contact",
		"SECTION_5_TEXT":   "If you have questions about your data, you can contact us through the same request form.",
	}
}
