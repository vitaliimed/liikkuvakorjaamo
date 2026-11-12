package main

import "testing"

func TestLiikkuvaAllowedWorkerURL(t *testing.T) {
    cases := []struct{
        url string
        ok bool
    }{
        {"https://liikkuva-preview-451713593869.europe-north1.run.app/internal/process-lead", true},
        {"https://liikkuva-preview-knrjxcx2bq-lz.a.run.app/internal/process-lead", true},
        {"https://liikkuva-landing-451713593869.europe-north1.run.app/internal/process-lead", true},
        {"https://liikkuva-preview-451713593869.europe-north1.run.app/internal/other", false},
        {"https://crm-wa-v2---liikkuva-preview-knrjxcx2bq-lz.a.run.app/internal/process-lead", false},
        {"https://liikkuva-preview-451713593869.europe-north1.run.app/internal/process-lead?x=1", false},
        {"http://liikkuva-preview-451713593869.europe-north1.run.app/internal/process-lead", false},
        {"https://liikkuva-preview-451713593869.europe-north1.run.app.evil.example/internal/process-lead", false},
        {"https://liikkuva-preview-451713593869.europe-north1.run.app:444/internal/process-lead", false},
        {"", false},
    }
    for _,c := range cases {
        if got:=liikkuvaAllowedWorkerURL(c.url);got!=c.ok {
            t.Errorf("%q: got %v, want %v",c.url,got,c.ok)
        }
    }
}
