package main

import (
 "os"
 "strings"
 "testing"
)

// Visual regressions: green is the site identity; orange is only an accent.
func TestSiteKeepsGreenSectionsWithOrangeAccents(t *testing.T) {
 files := []string{"static/css/style.css", "static/css/third.css", "static/css/redesign.css"}
 for _, file := range files {
  content, err := os.ReadFile(file)
  if err != nil { t.Fatal(err) }
  css := string(content)
  for _, selector := range []string{".topbar{background:#f97315", ".areas-section{background:#f97315", ".final-cta{background:#f97315"} {
   if strings.Contains(css, selector) {
    t.Errorf("%s contains an orange full-width section: %s", file, selector)
   }
  }
 }
 third, err := os.ReadFile("static/css/third.css")
 if err != nil { t.Fatal(err) }
 for _, s := range []string{
  "--green:#23372b", ".topbar{background:#23372b",
  ".areas-section{background:#23372b",
  ".final-cta{background:#23372b",
  ".hero-points .icon{color:#f97315",
 } {
  if !strings.Contains(string(third), s) { t.Errorf("third.css is missing %q", s) }
 }
 brand, err := os.ReadFile("static/css/brand.css")
 if err != nil { t.Fatal(err) }
 for _, s := range []string{"--orange:#f97315", "--green:#23372b", "--pine:#23372b", ".footer,.subpage-footer"} {
  if !strings.Contains(string(brand), s) { t.Errorf("brand.css is missing %q", s) }
 }
}
