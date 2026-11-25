package main

import (
    "bytes"
    "fmt"
    "io"
    "strings"

    "golang.org/x/net/html"
)

// Markers are controlled by the landing template. Only the direct children of
// main participate, so rearranging sections cannot detach forms or their inputs.
func renderServiceLayout(dst io.Writer, source io.Reader, layout string) error {
    doc,err:=html.Parse(source)
    if err!=nil{return err}
    var main *html.Node
    for n:=range doc.Descendants(){if n.Type==html.ElementNode && n.Data=="main"{main=n;break}}
    if main==nil{return fmt.Errorf("landing template is missing main")}
    blocks:=map[string]*html.Node{}
    var footerForm *html.Node
    for n:=main.FirstChild;n!=nil; {
        next:=n.NextSibling
        for _,a:=range n.Attr {
            if a.Key=="id" && a.Val=="footer-lead"{footerForm=n}
            if a.Key=="data-service-block"{blocks[a.Val]=n;main.RemoveChild(n);break}
        }
        n=next
    }
    for _,name:=range strings.Fields(layout) {
        if n:=blocks[name];n!=nil{main.InsertBefore(n,footerForm);delete(blocks,name)}
    }
    return html.Render(dst,doc)
}

func (tr *TemplateRenderer) renderOrderedServicePage(w io.Writer,data LandingPageData) error {
    var page bytes.Buffer
    if err:=tr.landing.Execute(&page,data);err!=nil{return err}
    return renderServiceLayout(w,&page,data.ServicePage.Layout)
}
