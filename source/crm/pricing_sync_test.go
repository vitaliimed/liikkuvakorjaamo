package crm

import "testing"

func TestSiteAdminOverridesLiikkuvaPriceBookOnly(t *testing.T) {
	book := PriceBook{Revision: 9, Items: []Tariff{
		{Brand: "liikkuva", Service: "visit", Title: "Выезд", Cents: 7000, Currency: "EUR", Active: false},
		{Brand: "liikkuva", Service: "diagnostics", Title: "Диагностика", Cents: 5000, Currency: "EUR", Active: false},
		{Brand: "liikkuva", Service: "battery", Title: "Battery", Cents: 9900, Currency: "EUR", Active: true},
		{Brand: "autohelp", Service: "visit", Title: "Выезд", Cents: 5000, Currency: "EUR", Active: true},
	}}
	got := applySiteTariffs(book, 9200, 8300)
	if got.Revision != 9 || len(got.Items) != 4 {
		t.Fatalf("unexpected pricebook revision or count: %+v", got)
	}
	for _, p := range got.Items {
		switch p.Brand+":"+p.Service {
		case "liikkuva:visit":
			if p.Cents != 9200 || !p.Active || p.Minimum { t.Fatalf("visit: %+v", p) }
		case "liikkuva:diagnostics":
			if p.Cents != 8300 || !p.Active || !p.Minimum { t.Fatalf("diagnostics: %+v", p) }
		case "liikkuva:battery":
			if p.Cents != 9900 { t.Fatalf("unrelated service changed: %+v", p) }
		case "autohelp:visit":
			if p.Cents != 5000 { t.Fatalf("autohelp changed: %+v", p) }
		}
	}
	var newJob Job
	newJob.Brand = "liikkuva"
	applyPrices(&newJob, got)
	if newJob.Proposal.VisitCents != 9200 || newJob.Proposal.WorkCents != 8300 {
		t.Fatalf("new CRM job still has old prices: %+v", newJob.Proposal)
	}
}

func TestCRMEmailTokensUseCurrentLandingPrices(t *testing.T) {
	got := crmPricePlaceholders("Käynti {visit} €, diagnostiikka {diagnostic} €, yhteensä {total} €. Osat 75 €.", 5000, 6500)
	want := "Käynti 50 €, diagnostiikka 65 €, yhteensä 115 €. Osat 75 €."
	if got != want { t.Fatalf("unexpected email price template: %q", got) }
}

func TestSiteTariffsRecreateMissingStandardServices(t *testing.T) {
	book := applySiteTariffs(PriceBook{}, 5000, 5000)
	if len(book.Items) != 2 { t.Fatalf("need both site-owned tariffs: %+v", book.Items) }
	book = applySiteTariffs(book, 7500, 6000)
	if len(book.Items) != 2 { t.Fatalf("duplicate site tariffs: %+v", book.Items) }
}
