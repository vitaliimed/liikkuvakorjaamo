package main

import (
    _ "embed"
    "encoding/json"
    "fmt"
    "net/http"
    "regexp"
    "sort"
    "strings"
    "sync"
    "time"
)

type ServiceBlock struct { Title string; Text string }
type ServicePage struct {
    Path, Language, Title, Description, Heading, Intro string
    Blocks []ServiceBlock
    Published bool
}

//go:embed service-pages.json
var defaultServiceJSON []byte
var servicePages = func() map[string]ServicePage {
    var pages map[string]ServicePage
    if err := json.Unmarshal(defaultServiceJSON, &pages); err != nil { panic(err) }
    return pages
}()

var serviceRenderer = sync.OnceValues(newTemplateRenderer)
var extraServicePath = regexp.MustCompile(`^/(palvelut|en/services)/[a-z0-9]+(-[a-z0-9]+)*$`)
func campaignDeadline(date string) int64 {
    loc, err := time.LoadLocation("Europe/Helsinki")
    if err != nil { return 0 }
    last, err := time.ParseInLocation("2006-01-02",date,loc)
    if err != nil { return 0 }
    return last.AddDate(0,0,1).Unix()
}
func serviceCatalog(settings SiteSettings) []ServicePage {
    catalog := make(map[string]ServicePage,len(servicePages))
    for path,p := range servicePages {
        p.Path = path
        var title,description,heading,body string
        switch path {
        case "/palvelualue": title,description,heading,body = settings.ProAreasTitle,settings.ProAreasDescription,settings.ProAreasHeading,settings.ProAreasText
        case "/vikadiagnostiikka": title,description,heading,body = settings.ProDiagnosticsTitle,settings.ProDiagnosticsDescription,settings.ProDiagnosticsHeading,settings.ProDiagnosticsText
        case "/hinnasto": title,description,heading,body = settings.ProPricingTitle,settings.ProPricingDescription,settings.ProPricingHeading,settings.ProPricingText
        }
        if title!="" {p.Title=title}; if description!="" {p.Description=description}; if heading!="" {p.Heading=heading}; if body!="" {p.Intro=body}
        catalog[path]=p
    }
    for _,p := range settings.ServicePages {catalog[p.Path]=p}
    out:=make([]ServicePage,0,len(catalog))
    for _,p:=range catalog {out=append(out,p)}
    sort.Slice(out,func(i,j int)bool{return out[i].Path<out[j].Path})
    return out
}
func configureServicePage(data *LandingPageData, page ServicePage, path string) {
    data.PageTitle, data.OGTitle = page.Title,page.Title
    data.MetaDescription,data.OGDescription=page.Description,page.Description
    data.CanonicalURL="https://"+liikkuvaDomain+path
    data.OGURL=data.CanonicalURL
    data.Content.Hero.Title,data.Content.Hero.Subtitle=page.Heading,page.Intro
    data.HeroTitle,data.HeroSubtitle=page.Heading,page.Intro
    data.ServicePage=&page
}
func parseServiceEdits(r *http.Request) ([]ServicePage,error) {
    paths:=r.Form["service_page_path"]
    seen:=map[string]bool{}
    out:=make([]ServicePage,0,len(paths))
    for i,path:=range paths {
        path=strings.TrimSpace(path)
        if path=="" {continue}
        _,builtin:=servicePages[path]
        if (!builtin && !extraServicePath.MatchString(path)) || seen[path] { return nil,fmt.Errorf("Недопустимый или повторяющийся URL: %s",path) }
        seen[path]=true
        field:=func(key string)string {values:=r.Form["service_page_"+key];if i<len(values){return strings.TrimSpace(values[i])};return ""}
        p:=ServicePage{Path:path,Language:"fi",Title:field("title"),Description:field("description"),Heading:field("heading"),Intro:field("intro"),Published:builtin || field("status")=="published"}
        if strings.HasPrefix(path,"/en/"){p.Language="en"}
        if p.Title=="" || p.Description=="" || p.Heading=="" || p.Intro=="" {return nil,fmt.Errorf("%s: заполните заголовки, описание и введение",path)}
        for _,block:=range strings.Split(strings.ReplaceAll(field("body"),"\r\n","\n"),"\n---\n"){
            title,body,ok:=strings.Cut(strings.TrimSpace(block),"\n")
            if title==""{continue};if !ok{return nil,fmt.Errorf("%s: после заголовка раздела нужен текст",path)}
            p.Blocks=append(p.Blocks,ServiceBlock{Title:title,Text:strings.TrimSpace(body)})
        }
        out=append(out,p)
    }
    return out,nil
}
