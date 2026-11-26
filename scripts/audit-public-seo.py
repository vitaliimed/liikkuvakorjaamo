#!/usr/bin/env python3
"""Read-only public page inventory for production. Writes a CSV to stdout."""
import csv
import json
import sys
import urllib.error
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
from html.parser import HTMLParser

BASE = "https://liikkuvakorjaamo.fi"


class Head(HTMLParser):
    def __init__(self):
        super().__init__()
        self.meta = {}
        self.canonicals = []
        self.alternates = []
        self.headings = []
        self.title = ""
        self.capture = ""
        self.forms = 0
        self.reviews = 0
        self.structured = []
        self.links = set()

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        if tag == "article" and "testimonial-card" in a.get("class", "").split():
            self.reviews += 1
        if tag == "br" and self.capture == "h1":
            self.headings[-1] += " "
        if tag in ("title", "h1"):
            self.capture = tag
            if tag == "h1":
                self.headings.append("")
        if tag == "meta":
            self.meta[a.get("name", a.get("property"))] = a.get("content", "")
        if tag == "link" and a.get("rel") == "canonical":
            self.canonicals.append(a.get("href", ""))
        if tag == "link" and a.get("hreflang"):
            self.alternates.append(a["hreflang"] + "=" + a.get("href", ""))
        if tag == "form" and (a.get("id") == "leadForm" or a.get("action") == "/lead"):
            self.forms += 1
        if tag == "a" and a.get("href", "").startswith("/"):
            self.links.add(a["href"].split("#")[0])
        if tag == "script" and a.get("type") == "application/ld+json":
            self.structured.append("")
            self.capture = "script"

    def handle_endtag(self, tag):
        if tag == self.capture:
            self.capture = ""

    def handle_data(self, data):
        if self.capture == "title":
            self.title += data
        elif self.capture == "h1":
            self.headings[-1] += data
        elif self.capture == "script":
            self.structured[-1] += data


def fetch(url):
    req = urllib.request.Request(url, headers={"User-Agent": "Liikkuva site audit/2"})
    try:
        response = urllib.request.urlopen(req, timeout=25)
    except urllib.error.HTTPError as err:
        response = err
    with response:
        return response.status, response.geturl(), response.headers, response.read().decode("utf-8")


def main():
    status, _, _, xml = fetch(BASE + "/sitemap.xml")
    if status != 200:
        raise SystemExit(f"Sitemap HTTP {status}")
    root = ET.fromstring(xml)
    urls = [n.text for n in root.findall("{*}url/{*}loc")]
    report = csv.writer(sys.stdout)
    report.writerow(["url", "status", "final_url", "title", "description", "h1", "canonical",
                     "robots", "x_robots_tag", "hreflang", "og_title", "og_image", "json_ld",
                     "server_lead_forms", "published_reviews", "internal_links", "issues"])
    titles = {}
    for url in urls:
        if urllib.parse.urlsplit(url).netloc != "liikkuvakorjaamo.fi":
            raise SystemExit("Unexpected sitemap host")
        status, final, headers, body = fetch(url)
        doc = Head()
        doc.feed(body)
        issues = []
        if status != 200 or final != url:
            issues.append("status/redirect")
        if len(doc.headings) != 1:
            issues.append("H1 count")
        if doc.canonicals != [url]:
            issues.append("canonical")
        if not doc.title or doc.title in titles:
            issues.append("missing/duplicate title")
        titles[doc.title] = url
        for name in ("description", "og:title", "og:image"):
            if not doc.meta.get(name):
                issues.append(name)
        if not doc.forms:
            issues.append("no inline lead form")
        for block in doc.structured:
            try:
                json.loads(block)
            except ValueError:
                issues.append("invalid JSON-LD")
        if not doc.structured:
            issues.append("missing JSON-LD")
        report.writerow([url, status, final, doc.title, doc.meta.get("description"), " | ".join(doc.headings),
                         " | ".join(doc.canonicals), doc.meta.get("robots"), headers.get("X-Robots-Tag"),
                         " | ".join(doc.alternates), doc.meta.get("og:title"), doc.meta.get("og:image"),
                         len(doc.structured), doc.forms, doc.reviews, " | ".join(sorted(doc.links)), " | ".join(issues)])


if __name__ == "__main__":
    main()
