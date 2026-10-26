package main

import (
 "bytes"
 "encoding/json"
 "fmt"
 "html"
 "reflect"
 "net/http"
 "regexp"
 "strconv"
 "strings"
)
type liikkuvaResponse struct{headers http.Header;Body bytes.Buffer;Code int}
func (w *liikkuvaResponse) Header()http.Header{return w.headers}
func (w *liikkuvaResponse) WriteHeader(code int){if w.Code==0{w.Code=code}}
func (w *liikkuvaResponse) Write(b []byte)(int,error){if w.Code==0{w.Code=200};return w.Body.Write(b)}
func liikkuvaInt(s string,d,min,max int) int {n,e:=strconv.Atoi(s);if e!=nil||n<min||n>max{return d};return n}
func liikkuvaVisitFee(s SiteSettings) int {if s.ProVisitEUR<1||s.ProVisitEUR>2000{return 70};return s.ProVisitEUR}
func liikkuvaLeadFee(l Lead) int {if l.VisitPriceEUR>0{return l.VisitPriceEUR};return 70}
func liikkuvaProDefaults(s SiteSettings) SiteSettings {s.ProVisitEUR=liikkuvaVisitFee(s);if s.ProDiagnosticEUR<1||s.ProDiagnosticEUR>2000{s.ProDiagnosticEUR=50};if s.ProFontSize<16||s.ProFontSize>22{s.ProFontSize=18};if !regexp.MustCompile(`^#[a-fA-F0-9]{6}$`).MatchString(s.ProAccent){s.ProAccent="#2458d6"};return s}
func liikkuvaPriceText(s string,c SiteSettings) string {
 c=liikkuvaProDefaults(c)
 return strings.NewReplacer("70 €",fmt.Sprintf("%d €",c.ProVisitEUR),"€70",fmt.Sprintf("€%d",c.ProVisitEUR),"50 €",fmt.Sprintf("%d €",c.ProDiagnosticEUR),"€50",fmt.Sprintf("€%d",c.ProDiagnosticEUR),"{total}",strconv.Itoa(c.ProVisitEUR+c.ProDiagnosticEUR),"{visit}",strconv.Itoa(c.ProVisitEUR),"{diagnostic}",strconv.Itoa(c.ProDiagnosticEUR)).Replace(s)
}
func liikkuvaPro(cfg Config,next http.Handler) http.Handler {return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
 if r.URL.Path=="/tracking-config.js" {
  s,err:=loadSiteSettings(r.Context(),cfg);if err!=nil{http.Error(w,"Temporarily unavailable",503);return};s=liikkuvaSettings(s)
  b,_:=json.Marshal(map[string]string{"ga4MeasurementId":s.GA4MeasurementID,"googleAdsTagId":s.GoogleAdsTagID,"googleAdsLeadConversionLabel":s.GoogleAdsLeadConversionLabel,"metaPixelId":s.MetaPixelID,"linkedinPartnerId":s.LinkedInPartnerID,"linkedinLeadConversionID":s.LinkedInLeadConversionID,"tiktokPixelId":s.TikTokPixelID})
  w.Header().Set("Content-Type","application/javascript; charset=utf-8");w.Header().Set("Cache-Control","no-cache");fmt.Fprintf(w,"window.LIIKKUVA_TRACKING=%s;",b);return
 }
 public:=r.URL.Path=="/"||r.URL.Path=="/en"||r.URL.Path=="/hinnasto"||r.URL.Path=="/palvelualue"||r.URL.Path=="/vikadiagnostiikka"||strings.HasPrefix(r.URL.Path,"/thank-you")||r.URL.Path=="/fi"
 if (r.Method!=http.MethodGet&&r.Method!=http.MethodHead)||!public {next.ServeHTTP(w,r);return}
 settings,err:=loadSiteSettings(r.Context(),cfg);if err!=nil {http.Error(w,"Service temporarily unavailable",503);return};settings=liikkuvaProDefaults(settings)
 rec:=&liikkuvaResponse{headers:make(http.Header)};next.ServeHTTP(rec,r);if rec.Code==0 {rec.Code=200}
 if !strings.Contains(rec.Header().Get("Content-Type"),"text/html") {for k,v:=range rec.Header(){w.Header()[k]=v};w.WriteHeader(rec.Code);w.Write(rec.Body.Bytes());return}
 body:=liikkuvaPriceText(rec.Body.String(),settings)
 css:=fmt.Sprintf(`<style>:root{--brand:%s;--blue:%s}html{font-size:%dpx}.btn-orange,.btn-primary{background:%s}</style>`,settings.ProAccent,settings.ProAccent,settings.ProFontSize,settings.ProAccent)
 body=strings.Replace(body,"</head>",css+"</head>",1)
 extra:="";switch r.URL.Path {case "/hinnasto":extra=settings.ProPricingText;case "/palvelualue":extra=settings.ProAreasText;case "/vikadiagnostiikka":extra=settings.ProDiagnosticsText}
 if extra!="" {block:=`<div class="seo-columns"><article class="seo-block"><p>`+strings.ReplaceAll(html.EscapeString(liikkuvaPriceText(extra,settings)),"\n","<br>")+`</p></article></div>`;body=regexp.MustCompile(`(?s)<div class="seo-columns">.*?<div class="seo-bottom">`).ReplaceAllStringFunc(body,func(string)string{return block+`<div class="seo-bottom">`})}
 section:="";switch r.URL.Path{case "/hinnasto":section="Pricing";case "/palvelualue":section="Areas";case "/vikadiagnostiikka":section="Diagnostics"}
 if section!="" {
  v:=reflect.ValueOf(settings);title:=v.FieldByName("Pro"+section+"Title").String();description:=v.FieldByName("Pro"+section+"Description").String();heading:=v.FieldByName("Pro"+section+"Heading").String()
  title=liikkuvaPriceText(title,settings);description=liikkuvaPriceText(description,settings);heading=liikkuvaPriceText(heading,settings)
  if title!="" {body=regexp.MustCompile(`<title>.*?</title>`).ReplaceAllStringFunc(body,func(string)string{return "<title>"+html.EscapeString(title)+"</title>"});body=regexp.MustCompile(`<meta property="og:title" content="[^"]*">`).ReplaceAllStringFunc(body,func(string)string{return `<meta property="og:title" content="`+html.EscapeString(title)+`">`})}
  if description!="" {for _,attribute:=range []string{`name="description"`,`property="og:description"`}{body=regexp.MustCompile(`<meta `+attribute+` content="[^"]*">`).ReplaceAllStringFunc(body,func(string)string{return `<meta `+attribute+` content="`+html.EscapeString(description)+`">`})}}
  if heading!="" {body=regexp.MustCompile(`(?s)<h1>.*?</h1>`).ReplaceAllStringFunc(body,func(string)string{return "<h1>"+html.EscapeString(heading)+"</h1>"})}
  body=regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`).ReplaceAllStringFunc(body,func(block string)string{start:=strings.Index(block,">")+1;end:=strings.LastIndex(block,"</script>");var data map[string]interface{};if json.Unmarshal([]byte(block[start:end]),&data)!=nil{return block};if title!=""{data["name"]=title};if description!=""{data["description"]=description};b,e:=json.Marshal(data);if e!=nil{return block};return `<script type="application/ld+json">`+string(b)+`</script>`})
 }

 for k,v:=range rec.Header(){w.Header()[k]=v};w.Header().Del("Content-Length");w.Header().Del("Last-Modified");w.Header().Set("Cache-Control","no-cache");w.WriteHeader(rec.Code);if r.Method!=http.MethodHead {w.Write(bytes.NewBufferString(body).Bytes())}
 })}
