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
  for _, selector := range []string{".topbar{background:#ff7410", ".areas-section{background:#ff7410", ".final-cta{background:#ff7410"} {
   if strings.Contains(css, selector) {
    t.Errorf("%s contains an orange full-width section: %s", file, selector)
   }
  }
 }
 third, err := os.ReadFile("static/css/third.css")
 if err != nil { t.Fatal(err) }
 for _, s := range []string{
  "--green:#223631", ".topbar{background:#223631",
  ".areas-section{background:#223631",
  ".final-cta{background:#223631",
  ".hero-points .icon{color:#ff7410",
 } {
  if !strings.Contains(string(third), s) { t.Errorf("third.css is missing %q", s) }
 }
 brand, err := os.ReadFile("static/css/brand.css")
 if err != nil { t.Fatal(err) }
 for _, s := range []string{"--orange:#ff7410", "--green:#223631", "--pine:#133e32", ".footer,.subpage-footer"} {
  if !strings.Contains(string(brand), s) { t.Errorf("brand.css is missing %q", s) }
 }
}
