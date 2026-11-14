#!/usr/bin/env python3
"""Render reviewed Finnish service copy. Run from any directory; no dependencies."""
from pathlib import Path
import html
import json

ROOT = Path(__file__).resolve().parents[1]
BASE = 'https://liikkuvakorjaamo.fi'
PAGES = json.loads((ROOT / 'docs/seo-pages.json').read_text())
esc = html.escape

def render(page):
    slug, title, description, heading, intro, sections, faq = page
    url = BASE + '/' + slug
    graph = {
        '@context': 'https://schema.org', '@graph': [
            {'@type': 'WebPage', '@id': url + '#page', 'url': url, 'name': title,
             'description': description, 'inLanguage': 'fi', 'about': {'@id': url + '#service'}},
            {'@type': 'Service', '@id': url + '#service', 'name': heading, 'url': url,
             'provider': {'@type': 'Organization', '@id': BASE + '/#organization',
                          'name': 'Liikkuva Auto Korjaamo', 'url': BASE + '/'},
             'areaServed': [{'@type': 'City', 'name': city} for city in ('Helsinki', 'Espoo', 'Vantaa')]},
            {'@type': 'BreadcrumbList', 'itemListElement': [
                {'@type': 'ListItem', 'position': 1, 'name': 'Etusivu', 'item': BASE + '/'},
                {'@type': 'ListItem', 'position': 2, 'name': heading, 'item': url}]}]}
    blocks = ''.join(f'<section class="seo-block"><h2>{esc(h)}</h2><p>{esc(t)}</p></section>' for h,t in sections)
    questions = ''.join(f'<h3>{esc(q)}</h3><p>{esc(a)}</p>' for q,a in faq)
    links = ''.join(f'<a href="/{p[0]}">{esc(p[3])}</a>' for p in PAGES if p[0] != slug)
    return f'''<!doctype html>
<html lang="fi"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{esc(title)}</title><meta name="description" content="{esc(description, quote=True)}">
<meta name="robots" content="index,follow,max-image-preview:large"><link rel="canonical" href="{url}">
<meta property="og:type" content="website"><meta property="og:locale" content="fi_FI">
<meta property="og:title" content="{esc(title, quote=True)}"><meta property="og:description" content="{esc(description, quote=True)}"><meta property="og:url" content="{url}">
<meta property="og:image" content="{BASE}/static/img/logo-white.png"><meta name="twitter:card" content="summary_large_image">
<link rel="icon" href="/static/img/favicon.svg"><link rel="stylesheet" href="/static/css/style.css">
<style>.seo-intro{{padding:3rem 0 1.5rem;max-width:54rem}}.seo-intro h1{{font-size:clamp(2rem,5vw,3.6rem);line-height:1.12}}.seo-intro p{{font-size:1.15rem;line-height:1.7}}.seo-columns{{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1.5rem}}.seo-block{{padding:1.5rem;border:1px solid var(--line,#dce1df);border-radius:16px}}.seo-block p{{line-height:1.75}}.seo-block h2{{font-size:1.5rem;line-height:1.3}}.seo-block h3{{font-size:1.1rem;margin-top:1.2rem}}.seo-bottom{{padding:2rem 0}}.resource-links{{display:flex;flex-wrap:wrap;gap:1rem;padding:1rem 0}}.seo-breadcrumb{{margin-bottom:1rem}}@media(max-width:680px){{.seo-columns{{grid-template-columns:1fr}}.subpage-header .container{{gap:1rem;flex-wrap:wrap}}}}</style>
<script type="application/ld+json">{json.dumps(graph,ensure_ascii=False).replace('<',chr(92)+'u003c')}</script>
<script>window.dataLayer=window.dataLayer||[];window.gtag=function(){{dataLayer.push(arguments)}};gtag('consent','default',{{analytics_storage:'denied',ad_storage:'denied',ad_user_data:'denied',ad_personalization:'denied'}});window.LIIKKUVA_CONSENT_UI={{lang:'fi',privacyUrl:'/privacy'}};</script>
<script src="/tracking-config.js" defer></script><script src="/static/js/consent.js" defer></script>
</head><body>
<a class="skip-link" href="#main">Siirry sisältöön</a>
<header class="subpage-header"><div class="container"><a class="brand" href="/">Liikkuva Auto Korjaamo</a><a class="btn btn-orange" href="/#lead-form">Pyydä käynti ↗</a></div></header>
<main id="main" class="container"><div class="seo-intro"><nav class="seo-breadcrumb" aria-label="Murupolku"><a href="/">Etusivu</a> / {esc(heading)}</nav><h1>{esc(heading)}</h1><p>{esc(intro)}</p></div>
<div class="seo-columns">{blocks}<section class="seo-block"><h2>Käynti ja hinnoittelu</h2><p>Käyntimaksu on {{visit}} €. Tarvittava diagnostiikka maksaa erikseen alkaen {{diagnostic}} €, joten käynti ja diagnostiikka ovat yhteensä alkaen {{total}} €. Korjaustyö ja varaosat sovitaan erikseen. Tarkka hinta ja työn laajuus vahvistetaan ennen käyntiä.</p><a href="/hinnasto">Katso hinnasto →</a></section><section class="seo-block"><h2>Usein kysytyt kysymykset</h2>{questions}</section></div>
<section class="seo-bottom"><h2>Kerro oire ja sijainti – sovitaan seuraava vaihe</h2><p>Ilmoita rekisterinumero, osoite, yhteystiedot ja toivottu aika. Pyynnön lähettäminen ei vielä vahvista varausta.</p><a class="btn btn-orange" href="/#lead-form">Lähetä avunpyyntö ↗</a></section>
<nav class="resource-links" aria-label="Aiheeseen liittyvät palvelut">{links}<a href="/vikadiagnostiikka">Vikadiagnostiikka</a><a href="/palvelualue">Palvelualue</a></nav></main>
<footer class="footer"><div class="container resource-links"><a href="/">Etusivu</a><a href="/hinnasto">Hinnasto</a><a href="/privacy">Tietosuoja</a><a href="/en">English</a></div></footer></body></html>
'''

if __name__ == '__main__':
    for page in PAGES:
        for folder in ('source/static/seo', 'assets/static/seo'):
            (ROOT / folder / (page[0] + '.html')).write_text(render(page))
    print(f'Rendered {len(PAGES)} service pages in runtime and preparation assets.')
