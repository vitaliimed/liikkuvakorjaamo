package main

import (
	"strings"
	"testing"
)

func TestOrangeBrandEmailLanguageAndEscaping(t *testing.T) {
	cfg := Config{SiteURL: "https://liikkuvakorjaamo.fi", MailFrom: "service@liikkuvakorjaamo.fi"}
	for _, lang := range []string{"fi", "en"} {
		lead := Lead{Language: lang, Name: "<Client>", Car: "ABC-123"}
		body := buildCustomEmailHTMLBody(cfg, lead, "<Subject>", "<script>alert(1)</script>\nSecond line")
		greeting := "Hello"
		if lang == "fi" {
			greeting = "Hei"
		}
		for _, wanted := range []string{`<html lang="` + lang + `">`, greeting + " &lt;Client&gt;", "logo-green.png", "#133e32", "#ff7410", "&lt;script&gt;", "<br>Second line"} {
			if !strings.Contains(body, wanted) {
				t.Errorf("%s email missing %q", lang, wanted)
			}
		}
		if strings.Contains(body, "<script>") {
			t.Fatal("unescaped message")
		}
		text := buildCustomEmailTextBody(cfg, lead, "Message")
		if !strings.HasPrefix(text, greeting+" <Client>") {
			t.Errorf("wrong text language: %s", text)
		}
		ready := buildReadyEmailHTMLBody(cfg, lead)
		if !strings.Contains(ready, `<html lang="`+lang+`">`) || !strings.Contains(ready, "logo-green.png") {
			t.Error("ready email lost shared branding/language")
		}
	}
}
func TestStandaloneBrandPreservesOperatorText(t *testing.T) {
	body := standaloneEmailHTMLLanguage("Hei!\nCustom message", "en")
	if !strings.Contains(body, `lang="en"`) || !strings.Contains(body, "Hei!\nCustom message") || strings.Contains(body, "Hello customer") {
		t.Fatal("standalone email changed operator text")
	}
}
func TestLegacyBlueMigratesToOrange(t *testing.T) {
	if got := liikkuvaProDefaults(SiteSettings{ProAccent: "#2458d6"}).ProAccent; got != "#ff7410" {
		t.Fatalf("legacy accent: %s", got)
	}
	if got := liikkuvaProDefaults(SiteSettings{ProAccent: "#123456"}).ProAccent; got != "#123456" {
		t.Fatalf("custom accent overwritten: %s", got)
	}
}
