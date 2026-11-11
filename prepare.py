#!/usr/bin/env python3
"""Create a separate fork of the user's CURRENT AutoHelp source; never edit it."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shutil
import subprocess
from pro_patch import apply_pro

ROOT=Path(__file__).resolve().parent
DOMAIN='liikkuvakorjaamo.fi'

def replace_function(source,name,new):
    pattern=r'func\s+(?:\([^\n]+\)\s+)?'+re.escape(name)+r'\s*\('
    match=re.search(pattern,source)
    if not match:raise RuntimeError('Не найдена функция '+name+'; исходники не изменены.')
    start=match.start();brace=source.index('{',match.end());depth=0;quote=None;comment=None;i=brace
    while i<len(source):
        c=source[i];n=source[i:i+2]
        if comment=='line':
            if c=='\n':comment=None
        elif comment=='block':
            if n=='*/':comment=None;i+=1
        elif quote:
            if c=='\\' and quote!='`':i+=1
            elif c==quote:quote=None
        elif n=='//':comment='line';i+=1
        elif n=='/*':comment='block';i+=1
        elif c in ['"',"'",'`']:quote=c
        elif c=='{':depth+=1
        elif c=='}':
            depth-=1
            if depth==0:return source[:start]+new+source[i+1:]
        i+=1
    raise RuntimeError('Не удалось безопасно разобрать функцию '+name)

def fingerprint(source):
    result={}
    for file in sorted(source.rglob('*')):
        if file.is_file() and file.suffix in ('.go','.html','.js','.css') and '.git' not in file.parts:
            result[str(file.relative_to(source))]=hashlib.sha256(file.read_bytes()).hexdigest()
    return result

def prepare(source,destination):
    source=source.expanduser().resolve();destination=destination.expanduser().resolve()
    if not (source/'go.mod').is_file():raise RuntimeError('В исходной папке не найден go.mod AutoHelp.')
    if destination==source or source in destination.parents or destination in source.parents:raise RuntimeError('Новая папка должна быть отдельной, вне исходной.')
    if destination.exists():raise RuntimeError('Новая папка уже существует. Используйте другое имя; перезапись запрещена.')
    required=['config.go','main.go','models.go','handlers_lead.go','services_firestore.go','services_appsscript.go','services_site_settings.go','texts.go','viewmodels.go','handlers_pages.go','landing.html','services_reviews.go','services_telegram.go','handlers_admin.go','landing_visuals.go']
    for name in required:
        if not (source/name).is_file():raise RuntimeError('Отсутствует '+name+'; установка остановлена.')
    before=fingerprint(source)
    destination.mkdir(parents=True)
    try:
        # Copy only source code, never .env, passwords, logs, customer uploads or Git state.
        for file in source.iterdir():
            if file.is_file() and (file.suffix in ('.go','.html') or file.name in ('go.mod','go.sum')):
                shutil.copy2(file,destination/file.name)
        (destination/'templates').mkdir()
        if (source/'templates').is_dir():
            for file in (source/'templates').glob('*.html'):shutil.copy2(file,destination/'templates'/file.name)
        shutil.copytree(ROOT/'assets/static',destination/'static')
        for file in destination.rglob('*'):
            if not file.is_file() or file.suffix not in ('.go','.html','.js','.css','.mod'):continue
            text=file.read_text()
            text=text.replace('AutoHelp Uusimaa','Liikkuva Auto Korjaamo').replace('AutoHelp','Liikkuva').replace('AUTOHELP','LIIKKUVA').replace('autoHelp','liikkuva').replace('autohelp.fi',DOMAIN).replace('autohelp','liikkuva')
            # Keep the existing verified sender address, not an invented mailbox.
            text=text.replace('service@'+DOMAIN,'service@autohelp.fi')
            text=re.sub(r'firestore\.NewClient\(\s*ctx\s*,\s*cfg\.FirestoreProjectID\s*\)', 'firestore.NewClientWithDatabase(ctx, cfg.FirestoreProjectID, "liikkuvakorjaamo")',text)
            text=re.sub(r'firestore\.NewClientWithDatabase\(\s*ctx\s*,\s*cfg\.FirestoreProjectID\s*,\s*[^)]+\)', 'firestore.NewClientWithDatabase(ctx, cfg.FirestoreProjectID, "liikkuvakorjaamo")',text)
            file.write_text(text)
        for name in ['services_firestore.go','services_site_settings.go','services_reviews.go']:
            text=(destination/name).read_text()
            if 'firestore.NewClient(' in text:raise RuntimeError('Не удалось изолировать все подключения Firestore в '+name)
        main=destination/'main.go';text=main.read_text();needle='cfg := loadConfig()'
        if needle not in text:raise RuntimeError('Неизвестный запуск приложения; подготовка остановлена.')
        text=text.replace(needle,needle+'\n if err:=validateLiikkuva(cfg);err!=nil{log.Fatal(err)}',1);main.write_text(text)
        config=destination/'config.go';text=config.read_text()
        needle='loggingMiddleware(newRouter(cfg, renderer))'
        if needle not in text:raise RuntimeError('Неизвестный router; подготовка остановлена.')
        text=text.replace(needle,'liikkuvaSecurity(cfg, loggingMiddleware(newRouter(cfg, renderer)))');config.write_text(text)
        # Preserve first-run section defaults in the editor as well.
        admin=destination/'handlers_admin.go'
        if admin.is_file():admin.write_text(admin.read_text().replace('settings.StatsEnabled = true','settings.StatsEnabled = false',1))
        model=destination/'models.go';text=model.read_text().replace('type Lead struct {','type Lead struct {\n DiagnosticFromEUR int `json:"diagnostic_from_eur" firestore:"diagnostic_from_eur"`\n Site string `json:"site" firestore:"site"`\n SubmissionID string `json:"submission_id" firestore:"submission_id"`',1);model.write_text(text)
        texts=destination/'texts.go';text=texts.read_text().replace('return landingContentFI()','return liikkuvaContent(landingContentFI(), "fi")',1).replace('return landingContentEN()','return liikkuvaContent(landingContentEN(), "en")',1)
        text=text.replace('func applySectionVisibility(content *LandingContent, settings SiteSettings) {\n\tcontent.ShowStats = true','func applySectionVisibility(content *LandingContent, settings SiteSettings) {\n\tcontent.ShowStats = false');texts.write_text(text)
        lead=destination/'handlers_lead.go';text=lead.read_text()
        text=replace_function(text,'Submit',(ROOT/'bundle/submit.go.txt').read_text())
        text=replace_function(text,'handleAsyncSubmit',(ROOT/'bundle/async.go.txt').read_text())
        text=text.replace('Name:               strings.TrimSpace(r.FormValue("name")),','Site: liikkuvaDomain,\n SubmissionID: liikkuvaSubmissionID(),\n Name: strings.TrimSpace(r.FormValue("name")),',1)
        if 'SubmissionID: liikkuvaSubmissionID()' not in text:raise RuntimeError('Неизвестный конструктор заявки.')
        text=text.replace('ScriptToken:        cfg.GoogleAppsScriptToken','ScriptToken:        ""').replace('time.Now().Format("2006-01-02 15:04:05")','liikkuvaTimestamp(time.Now())');lead.write_text(text)
        sheets=destination/'services_appsscript.go';text=sheets.read_text().replace('func sendLeadToGoogleAppsScript(cfg Config, lead Lead) error {','func sendLeadToGoogleAppsScript(cfg Config, lead Lead) error {\n lead.ScriptToken = cfg.GoogleAppsScriptToken',1);sheets.write_text(text)
        telegram=destination/'services_telegram.go';text=telegram.read_text()
        text=text.replace('buildTelegramMessage(lead)', '("🔧 <b>liikkuvakorjaamo.fi</b>\\n" + buildTelegramMessage(lead))')
        if 'liikkuvakorjaamo.fi</b>' not in text:raise RuntimeError('Не удалось добавить метку домена в Telegram.')
        telegram.write_text(text)
        for page,body in [('landing.html','landing-body.html'),('privacy.html','privacy-body.html'),('thankyou.html','thankyou-body.html')]:
            for file in [destination/page,destination/'templates'/page]:
                if not file.is_file():continue
                original=file.read_text();head=original.split('</head>',1)[0]
                head=re.sub(r'<script[^>]*src="https://www\.googletagmanager\.com/gtag/js[^"]*"[^>]*>\s*</script>', '',head,flags=re.S)
                if '<head>' not in head:raise RuntimeError('Некорректный шаблон '+page)
                if 'surface.js' not in head:head+='<script src="/static/js/surface.js" defer></script>'
                if page=='landing.html':
                    head=re.sub(r'<script[^>]*type="application/ld\+json"[^>]*>.*?</script>', '',head,flags=re.S)
                    head+=(ROOT/'bundle/seo-head.html').read_text()
                head=re.sub(r'style.css[^"\s]*','style.css?v=20261025',head)
                file.write_text(head+'</head>'+ (ROOT/'bundle'/body).read_text())
        for test in destination.glob('*_test.go'):
            content=test.read_text()
            if 'func TestTestimonialCardsHaveUnifiedLayout(' in content:
                content=replace_function(content,'TestTestimonialCardsHaveUnifiedLayout','// The original visual layout test is obsolete: the user requested a different layout.')
                test.write_text(content)
        for name in ['liikkuva_brand.go','liikkuva-content.json','seo-defaults.json']:shutil.copy2(ROOT/'bundle'/name,destination/name)
        # Safe CSS values generated by existing numeric clamp functions.
        views=destination/'viewmodels.go';text=views.read_text()
        if '"html/template"' not in text:text=text.replace('package main','package main\nimport "html/template"',1)
        keys=['HeroStyleVars','HeroSceneImageStyle','HeroPersonLayerStyle','HeroPersonImageStyle','AreasStyleVars','AreasMapImageStyle','ServiceImageStyle']
        for key in keys:text=re.sub(r'(\b'+key+r'\s+)string',r'\1template.CSS',text)
        views.write_text(text)
        pages=destination/'handlers_pages.go';text=pages.read_text()
        text=text.replace('settings, err := loadSiteSettings(r.Context(), h.cfg)', 'settings, err := loadSiteSettings(r.Context(), h.cfg)\n settings=liikkuvaSettings(settings)')
        text=text.replace('pageTitle := texts["PAGE_TITLE"]\n\tmetaDescription := texts["META_DESCRIPTION"]', 'seoDefaults:=liikkuvaSettings(SiteSettings{})\n pageTitle,metaDescription:=seoDefaults.LandingSEOTitleEN,seoDefaults.LandingSEODescriptionEN\n if lang=="fi" {pageTitle=seoDefaults.LandingSEOTitleFI;metaDescription=seoDefaults.LandingSEODescriptionFI}',1)
        if '"html/template"' not in text:text=text.replace('"log"','"log"\n"html/template"',1)
        for key in keys:text=re.sub(r'('+key+r':\s*)(build'+key+r'\(settings\))',r'\1template.CSS(\2)',text)
        pages.write_text(text)
        visuals=destination/'landing_visuals.go';text=visuals.read_text()
        text=text.replace('func buildHeroSceneImageStyle(settings SiteSettings) string {','func buildHeroSceneImageStyle(settings SiteSettings) string {\n if settings.HeroScenePreset=="" && settings.HeroSceneBrightness=="" && settings.HeroSceneContrast=="" && settings.HeroSceneSaturation=="" && settings.HeroSceneBlur=="" && settings.HeroSceneOpacity=="" && settings.HeroSceneScale=="" {return "filter:none;"}',1)
        text=text.replace('func buildHeroStyleVars(settings SiteSettings) string {','func buildHeroStyleVars(settings SiteSettings) string {\n if settings.HeroOverlayOpacity=="" {return "--hero-scene-overlay-opacity:0;"}',1);visuals.write_text(text)
        for file in [destination/'admin.html',destination/'templates/admin.html']:
            if file.is_file():file.write_text(file.read_text().replace('#2563eb','#315ab6').replace('#1d4ed8','#10213a').replace('#f3f6fb','#f0f3f7').replace('#0f172a','#10213a'))

        for file in [destination/'review.html',destination/'templates/review.html']:
            if not file.is_file():continue
            html=file.read_text()
            header='<header class="subpage-header"><div class="container"><a class="brand" href="{{if eq .Lang "fi"}}/{{else}}/en{{end}}"><img src="{{.LogoURL}}" alt="" width="44" height="44"><span>Liikkuva<span class="brand-small">Auto Korjaamo</span></span></a><div class="languages">{{if .LeadID}}<a href="/review?lead={{.LeadID}}&token={{.Token}}&lang=fi">FI</a><a href="/review?lead={{.LeadID}}&token={{.Token}}&lang=en">EN</a>{{end}}</div></div></header>'
            file.write_text(re.sub(r'<header\b.*?</header>',lambda _:header,html,count=1,flags=re.S))
        for name in ['Dockerfile','.dockerignore','.gcloudignore']:shutil.copy2(ROOT/name,destination/name)
        apply_pro(destination)
        # Both public forms use the same named template, with independent IDs.
        for name in ['lead_form.go', 'lead-form.html']:
            shutil.copy2(ROOT/'source'/name, destination/name)
        renderer_file = destination/'templates.go'
        renderer_text = renderer_file.read_text()
        if '"requestForm": requestFormData' not in renderer_text:
            old = 'landing, err := template.ParseFiles(landingPath)'
            if old not in renderer_text:
                raise RuntimeError('Не найдено подключение шаблона формы. Исходники AutoHelp не изменены.')
            renderer_text = renderer_text.replace(old, 'formPath, err := resolveTemplatePath("lead-form.html")\n if err != nil { return nil, err }\n landing, err := template.New("landing.html").Funcs(template.FuncMap{"requestForm": requestFormData}).ParseFiles(landingPath, formPath)', 1)
            renderer_file.write_text(renderer_text)
        # The runtime ships both lookup locations; never depend on an empty directory
        # surviving Cloud Build's source archive.
        for name in ['landing.html','lead-form.html','privacy.html','thankyou.html','admin.html','review.html']:
            if (destination/name).is_file():shutil.copy2(destination/name,destination/'templates'/name)
        if fingerprint(source)!=before:raise RuntimeError('Исходники AutoHelp изменились параллельно. Этот скрипт их не редактирует.')
        print('Копия подготовлена: '+str(destination));print('Исходный AutoHelp не изменён.')
        return destination
    except Exception:
        print('Подготовка не завершена. Новая папка сохранена для проверки; исходный AutoHelp не менялся.')
        raise

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--source',required=True);parser.add_argument('--target',required=True)
    args=parser.parse_args();prepare(Path(args.source),Path(args.target))
