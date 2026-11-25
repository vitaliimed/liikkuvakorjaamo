package main

import (
    "net/http/httptest"
    "strings"
    "testing"
    "time"
)

func TestServiceLandingUsesSameLeadFlow(t *testing.T) {
    cfg:=Config{SiteURL:"https://liikkuvakorjaamo.fi", SettingsStore:seoSettingsStore{SiteSettings{}}}
    handler:=liikkuvaPro(cfg,liikkuvaSecurity(cfg,nil))
    for path:=range servicePages {
        w:=httptest.NewRecorder()
        handler.ServeHTTP(w,httptest.NewRequest("GET",cfg.SiteURL+path,nil))
        html:=w.Body.String()
        for _,part:=range []string{`id="leadForm"`,`id="footerLeadFormMount"`,`rel="canonical" href="`+cfg.SiteURL+path+`"`,`class="container service-detail"`} {
            if w.Code!=200 || !strings.Contains(html,part){t.Fatalf("%s missing %s (HTTP %d)",path,part,w.Code)}
        }
        if strings.Contains(html,`hreflang="en"`){t.Fatalf("%s advertises an unrelated translation",path)}
    }
}
func TestReviewExamplesStayPrivate(t *testing.T){
    settings:=normalizeAdminSettingsForTemplate(SiteSettings{})
    if len(settings.Testimonials)!=6 || len(buildTestimonialsFromSettings(settings,"fi"))!=0 {t.Fatal("six private examples expected")}
}
func TestOfferEndsAtHelsinkiMidnight(t *testing.T){
    if got:=time.Unix(campaignDeadline("2026-12-31"),0).UTC().Format(time.RFC3339);got!="2026-12-31T22:00:00Z"{t.Fatal(got)}
    if campaignDeadline("2026-02-31")!=0{t.Fatal("invalid deadline accepted")}
}
