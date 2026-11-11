package main

import (
 "os"
 "strings"
 "testing"
)

// Approved primary hues are green #23372b and orange #f97315.
// Neutral text/surface colours and warning/error semantics are allowed.
func TestCanonicalTwoColourBrand(t *testing.T) {
 styles := []string{"static/css/style.css","static/css/third.css","static/css/redesign.css","static/css/pro.css","static/css/brand.css","static/css/theme.css","static/css/responsive.css","static/css/admin-access.css"}
 old := []string{"#223631","#133e32","#ff7410","#f97316","#a94000","#2458d6","#001a4b","#315ab6","#123f32"}
 for _, path := range styles {
  b,err:=os.ReadFile(path)
  if err!=nil {t.Fatal(err)}
  css:=strings.ToLower(string(b))
  for _,hex:=range old {if strings.Contains(css,hex) {t.Errorf("outdated brand colour %s in %s",hex,path)}}
 }
 b,err:=os.ReadFile("static/css/brand.css")
 if err!=nil {t.Fatal(err)}
 for _,want:=range []string{"--green:#23372b","--orange:#f97315","--pine:#23372b","--forest:#23372b"} {
  if !strings.Contains(string(b),want) {t.Errorf("missing design token %q",want)}
 }
 for _,path:=range []string{"static/img/logo-transparent-light.svg","static/img/logo-transparent-dark.svg","static/img/logo-white.svg","static/img/logo-dark.svg","static/img/logo-green.svg","static/img/favicon.svg","static/img/logo_liikkuva.svg"} {
  b,err:=os.ReadFile(path)
  if err!=nil {t.Fatal(err)}
  svg:=strings.ToLower(string(b))
  if !strings.Contains(svg,"#f97315") {t.Errorf("orange arrow absent in %s",path)}
  for _,v:=range []string{"#f97316","#001a4b","#123f32"} {
   if strings.Contains(svg,v) {t.Errorf("old logo colour %s in %s",v,path)}
  }
 }
 for _,path:=range []string{"landing.html","templates/landing.html","privacy.html","review.html","thankyou.html","templates/privacy.html","templates/review.html","templates/thankyou.html","admin-access.html"} {
  b,err:=os.ReadFile(path)
  if err!=nil {t.Fatal(err)}
  if !strings.Contains(string(b),"logo-white.svg") || !strings.Contains(string(b),"logo-transparent-dark.svg") {
   t.Errorf("%s should use canonical vector logos",path)
  }
 }
 for _,unapproved :=range []string{"#123456","#ff7410","#2458d6"} {
  if got:=liikkuvaProDefaults(SiteSettings{ProAccent:unapproved}).ProAccent;got!="#f97315"{
   t.Errorf("runtime accent %q was %q, not canonical orange",unapproved,got)
  }
 }
}
