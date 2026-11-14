package crm

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCRMOriginProtectionWithTaggedCloudRunPreview(t *testing.T) {
    canonical := "https://liikkuvakorjaamo.fi"
    testCases := []struct{
        name, origin, host string
        want bool
    }{
        {"production domain", canonical, "liikkuvakorjaamo.fi", true},
        {"Cloud Run SMTP preview", "https://brevo-smtp-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app", "brevo-smtp-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app", true},
        {"Cloud Run UX preview", "https://ux-crm-brevo-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app", "ux-crm-brevo-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app", true},
        {"cross-site attack", "https://evil.example", "liikkuvakorjaamo.fi", false},
        {"canonical on preview must not be trusted", canonical, "brevo-smtp-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app", false},
        {"preview on production must not be trusted", "https://brevo-smtp-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app", "liikkuvakorjaamo.fi", false},
        {"insecure origin", "http://liikkuvakorjaamo.fi", "liikkuvakorjaamo.fi", false},
        {"suffix host confusion", canonical+".evil.example", "liikkuvakorjaamo.fi", false},
        {"origin with port", canonical+":444", "liikkuvakorjaamo.fi", false},
        {"origin with path", canonical+"/crm/", "liikkuvakorjaamo.fi", false},
        {"no Origin header backwards-compatible", "", "liikkuvakorjaamo.fi", true},
        {"http localhost", "http://crm.example", "crm.example", false},
    }
    for _,tc := range testCases {
        t.Run(tc.name,func(t *testing.T){
            got:=crmAllowedOrigin(tc.origin,tc.host,canonical)
            if got!=tc.want {t.Errorf("origin %q, host %q: got %v, want %v",tc.origin,tc.host,got,tc.want)}
        })
    }
    if !crmAllowedOrigin("http://localhost:8080","localhost:8080","http://localhost:8080") {
        t.Fatal("HTTP localhost developer mode should be allowed")
    }
}

func TestCRMOriginThroughFirebaseHostingRewrite(t *testing.T) {
    canonical := "https://liikkuvakorjaamo.fi"
    cases := []struct {
        name, host, origin, firebaseHost, forwardedHost string
        want bool
    }{
        {"exact Cloud Run backend with production Origin",
            "liikkuva-preview-451713593869.europe-north1.run.app", canonical, "", "", true},
        {"current Cloud Run backend with production Origin",
            "liikkuva-preview-knrjxcx2bq-lz.a.run.app", canonical, "", "", true},
        {"tagged rewrite with Firebase requested Host",
            "site-crm-release---liikkuva-preview-knrjxcx2bq-lz.a.run.app", canonical, "liikkuvakorjaamo.fi", "", true},
        {"tagged rewrite with forwarded Host",
            "crm-password-reset---liikkuva-preview-knrjxcx2bq-lz.a.run.app", canonical, "", "https://liikkuvakorjaamo.fi", true},
        {"tagged preview cannot claim public Origin without rewrite header",
            "ux-crm-seo-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app", canonical, "", "", false},
        {"tagged preview with its own same Origin",
            "ux-crm-seo-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app",
            "https://ux-crm-seo-preview---liikkuva-preview-knrjxcx2bq-lz.a.run.app", "", "", true},
        {"forged foreign Origin with trusted forwarded Host",
            "site-crm-release---liikkuva-preview-knrjxcx2bq-lz.a.run.app", "https://evil.example", "liikkuvakorjaamo.fi", "", false},
        {"untrusted backend with claimed forwarded Host",
            "evil-service.a.run.app", canonical, "liikkuvakorjaamo.fi", "", false},
        {"host suffix impersonation",
            "liikkuva-preview-knrjxcx2bq-lz.a.run.app.evil.example", canonical, "liikkuvakorjaamo.fi", "", false},
        {"forwarded host suffix impersonation",
            "crm-password-reset---liikkuva-preview-knrjxcx2bq-lz.a.run.app", canonical, "liikkuvakorjaamo.fi.evil.example", "", false},
        {"insecure canonical Origin",
            "liikkuva-preview-knrjxcx2bq-lz.a.run.app", "http://liikkuvakorjaamo.fi", "liikkuvakorjaamo.fi", "", false},
        {"foreign canonical setting",
            "liikkuva-preview-knrjxcx2bq-lz.a.run.app", canonical, "liikkuvakorjaamo.fi", "", false},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            r := httptest.NewRequest(http.MethodPost, "https://"+tc.host+"/crm/api/password/forgot", nil)
            r.Header.Set("Origin", tc.origin)
            r.Header.Set("X-Fh-Requested-Host", tc.firebaseHost)
            r.Header.Set("X-Forwarded-Host", tc.forwardedHost)
            target := canonical
            if tc.name == "foreign canonical setting" {
                target = "https://other.example"
            }
            if got := crmAllowedOriginForRequest(r, target); got != tc.want {
                t.Errorf("origin=%q backend=%q firebase=%q got %v want %v",
                    tc.origin, tc.host, tc.firebaseHost, got, tc.want)
            }
        })
    }
}

func TestCRMFirebaseSessionCookieNameAndLegacyFallback(t *testing.T) {
    if crmSessionCookieName != "__session" {
        t.Fatalf("Firebase Hosting requires __session, got %q", crmSessionCookieName)
    }
    for _, tc := range []struct{
        old, modern, want string
    }{
        {"legacy-token", "", "legacy-token"},
        {"", "new-token", "new-token"},
        {"legacy-token", "new-token", "new-token"},
    } {
        r := httptest.NewRequest(http.MethodGet, "https://liikkuvakorjaamo.fi/crm/", nil)
        if tc.old != "" {
            r.AddCookie(&http.Cookie{Name:"crm_session", Value:tc.old})
        }
        if tc.modern != "" {
            r.AddCookie(&http.Cookie{Name:"__session", Value:tc.modern})
        }
        cookie, err := crmSessionCookie(r)
        if err != nil || cookie.Value != tc.want {
            t.Errorf("cookie resolution: %v %#v", err, cookie)
        }
    }
}
