package crm

import "testing"

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
