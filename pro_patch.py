"""Release-specific changes, applied to a separate copy only."""
from pathlib import Path
import re,shutil
from admin_ru import translate
ROOT=Path(__file__).resolve().parent
FIELDS={'ProVisitEUR':'int','ProDiagnosticEUR':'int','ProFontSize':'int','ProAccent':'string','ProPricingText':'string','ProAreasText':'string','ProDiagnosticsText':'string'}
for section in ['Pricing','Areas','Diagnostics']:
 for field in ['Title','Description','Heading']:FIELDS['Pro'+section+field]='string'
def apply_pro(dst):
 def edit(name,fn):
  p=dst/name;p.write_text(fn(p.read_text()))
 edit('services_gcs.go',lambda s:s.replace('120*time.Second','30*time.Second'))
 edit('models.go',lambda s:re.sub(r'const visitPriceEUR = \d+','const visitPriceEUR = 70',s).replace('type APIResponse struct {','type APIResponse struct {\n SubmissionID string `json:"submission_id,omitempty"`'))
 edit('viewmodels.go',lambda s:s.replace('type SiteSettings struct {','type SiteSettings struct {\n'+'\n'.join(k+' '+t for k,t in FIELDS.items())))
 edit('config.go',lambda s:s.replace('liikkuvaSecurity(cfg, loggingMiddleware(newRouter(cfg, renderer)))','liikkuvaPro(cfg, liikkuvaSecurity(cfg, loggingMiddleware(newRouter(cfg, renderer))))'))
 edit('handlers_admin.go',lambda s:s.replace('return t.Local().Format("2006-01-02 15:04:05")','return liikkuvaTimestamp(t)'))
 edit('handlers_admin.go',lambda s:s.replace('const adminCookieName = "liikkuva_admin"','const adminCookieName = "__session"'))
 if (dst/'release_test.go').exists():
  edit('release_test.go',lambda s:s.replace('Name:     "Test Client",','Name:     "Test Client",\n Phone: "+358401234567",').replace('"Custom email message"','"Текст письма клиенту"'))
 edit('handlers_lead.go',lambda s:s.replace('lead.Phone == "" && lead.Email == ""','lead.Phone == ""').replace('anna puhelin tai sähköposti','anna puhelinnumero').replace('phone or email is required','phone is required'))
 # Keep fee snapshots in notifications. Subjects do not quote a stale default.
 edit('services_email.go',lambda s:s.replace('fmt.Sprintf("Liikkuva: Pyyntö vastaanotettu – käynti %d €", visitPriceEUR)','"Liikkuva Auto Korjaamo: Pyyntö vastaanotettu"').replace('fmt.Sprintf("Liikkuva: Request received – €%d mechanic visit", visitPriceEUR)','"Liikkuva Auto Korjaamo: Request received"'))
 for name in ['services_email.go','services_telegram.go']:
  p=dst/name;s=p.read_text()
  # Functions with a lead parameter can use the fee recorded when the request was sent.
  from prepare import replace_function
  for match in list(re.finditer(r'func (\w+)\([^\n]*\blead Lead\)[^\n]*\{',s)):
   fn=match.group(1);start=s.find('func '+fn+'(');end=s.find('\nfunc ',start+5);chunk=s[start:end if end!=-1 else len(s)]
   if 'visitPriceEUR' in chunk:s=replace_function(s,fn,chunk.replace('visitPriceEUR','liikkuvaLeadFee(lead)'))
  s=s.replace('The price includes travel to your location, an initial on-site diagnosis and the possibility of a quick repair when it can be completed safely right away.','The visit fee covers travel. Diagnostics, repairs and spare parts are agreed and charged separately.')
  s=s.replace('Hinta sisältää saapumisen kohteeseen, alustavan vianmäärityksen ja mahdollisuuden nopeaan korjaukseen paikan päällä, jos työ voidaan tehdä turvallisesti heti.','Käyntimaksu kattaa saapumisen. Diagnostiikka, korjaukset ja varaosat sovitaan ja veloitetaan erikseen.').replace('The price includes travel to your location, initial diagnosis and the possibility of a quick repair on site if the work can be done safely right away.','The visit fee covers travel. Diagnostics, repairs and spare parts are agreed and charged separately.')
  p.write_text(s)
 # Parse validated presentation settings alongside the existing CMS fields.
 edit('handlers_admin.go',lambda s:s.replace('settings := SiteSettings{','settings := SiteSettings{\n'+''.join(k+': '+('liikkuvaInt(r.FormValue("'+k+'"),'+str({'ProVisitEUR':70,'ProDiagnosticEUR':50,'ProFontSize':18}[k])+','+('16,22' if k=='ProFontSize' else '1,2000')+')' if t=='int' else 'strings.TrimSpace(r.FormValue("'+k+'"))')+',\n' for k,t in FIELDS.items()),1))
 for name in ['landing.html','privacy.html','thankyou.html','review.html','admin.html']:
  p=dst/name
  if not p.exists():continue
  s=p.read_text().replace('</head>','<link rel="stylesheet" href="/static/css/pro.css?v=20261025"><script src="/static/js/events.js?v=20261025" defer></script></head>')
  s=s.replace('name="phone"','name="phone" required autocomplete="tel" inputmode="tel"')
  if name=='admin.html':
   s=s.replace('<html lang="en">','<html lang="ru">')
   s=s.replace('{{range .SelectedLead.PhotoURLs}}','{{range $index, $photo := .SelectedLead.PhotoURLs}}').replace('href="{{.}}" target="_blank" rel="noopener noreferrer">{{.}}</a>','href="/admin/photo?lead={{$.SelectedLead.ID}}&amp;index={{$index}}" target="_blank" rel="noopener noreferrer">Открыть фотографию</a>')
   s=s.replace('</head>','<link rel="stylesheet" href="/static/css/admin-mobile.css"><script src="/static/js/admin-mobile.js" defer></script></head>')
   fields='''<div class="pro-settings"><h2>Цены и оформление</h2><p>Выезд и диагностика оплачиваются отдельно. Значения ниже применяются к стандартным ценовым блокам и новым заявкам. Если заполнить основной текст SEO-страницы, он заменит стандартные описания этой страницы. Разделяйте абзацы пустой строкой. В собственных текстах используйте {visit} и {diagnostic}, чтобы цены обновлялись автоматически.</p>'''
   labels={'ProVisitEUR':'Выезд, €','ProDiagnosticEUR':'Диагностика от, €','ProFontSize':'Основной шрифт, px (16–22)','ProAccent':'Акцентный цвет (#2458d6)','ProPricingText':'Основной текст страницы «Цены» (финский)','ProAreasText':'Основной текст страницы «Зона выезда» (финский)','ProDiagnosticsText':'Основной текст страницы «Диагностика» (финский)'}
   for section,ru in [('Pricing','Цены'),('Areas','Зона выезда'),('Diagnostics','Диагностика')]:
    for field,label in [('Title','SEO-заголовок'),('Description','Meta description'),('Heading','Заголовок H1')]:labels['Pro'+section+field]=ru+' — '+label+' (финский)'
   for key,kind in FIELDS.items():
    fields+='<label>'+labels[key]
    if kind=='int':fields+='<input type="number" name="'+key+'" value="{{.Settings.'+key+'}}" min="'+('16' if key=='ProFontSize' else '1')+'" max="'+('22' if key=='ProFontSize' else '2000')+'">'
    elif key=='ProAccent':fields+='<input name="'+key+'" value="{{.Settings.'+key+'}}" pattern="#[0-9a-fA-F]{6}" placeholder="#2458d6">'
    else:fields+='<textarea name="'+key+'" rows="5">{{.Settings.'+key+'}}</textarea>'
    fields+='</label>'
   fields+='</div>'
   anchor='<section class="card panel {{if or (eq .ActiveTab "") (eq .ActiveTab "general")}}active{{end}}" data-panel="general">'
   if anchor not in s:raise RuntimeError('Admin panel structure has changed')
   s=s.replace(anchor,anchor+fields)
  if name=='admin.html':s=translate(s)
  p.write_text(s)
 for name in ['liikkuva_pro.go','liikkuva_pro_test.go','liikkuva_photos.go']:shutil.copy2(ROOT/'bundle'/name,dst/name)
 apply_email(dst)

 # Editor gets effective defaults too, so a normal save cannot reset fees to zero.
 edit('handlers_admin.go',lambda s:s.replace('func normalizeAdminSettingsForTemplate(settings SiteSettings) SiteSettings {','func normalizeAdminSettingsForTemplate(settings SiteSettings) SiteSettings {\n settings=liikkuvaProDefaults(settings)'))
 for p in (dst/'static/seo').glob('*.html'):
  s=p.read_text().replace('</head>','<link rel="stylesheet" href="/static/css/pro.css"><script src="/tracking-config.js"></script><script src="/static/js/events.js" defer></script></head>');p.write_text(s)


def apply_email(dst):
 from prepare import replace_function
 p=dst/'services_email.go';s=p.read_text()
 s=replace_function(s,'sendSMTPMail','func sendSMTPMail(cfg Config,to,subject,textBody,htmlBody string,attachments ...EmailAttachment) error {return liikkuvaSendSMTP(cfg,to,subject,textBody,htmlBody,attachments...)}')
 s=s.replace('func sendCustomClientEmail(cfg Config, lead Lead, subject, message string) error {','func sendCustomClientEmail(cfg Config, lead Lead, subject, message string, attachments ...EmailAttachment) error {')
 s=s.replace('return sendSMTPMail(cfg, to, subject, textBody, htmlBody)','return sendSMTPMail(cfg, to, subject, textBody, htmlBody, attachments...)') if 'func sendReadyForPickupEmail' not in s else s
 start=s.index('func sendCustomClientEmail(');end=s.index('\nfunc ',start+5);chunk=s[start:end].replace('sendSMTPMail(cfg, to, subject, textBody, htmlBody)','sendSMTPMail(cfg, to, subject, textBody, htmlBody, attachments...)');s=s[:start]+chunk+s[end:]
 for imp in ['bytes','mime','mime/multipart','mime/quotedprintable','net/smtp','net/textproto','time']:
  if not re.search(r'\b'+imp.split('/')[-1]+r'\.',s):s=s.replace('"'+imp+'"','')
 s=s.replace('\"time\"','')
 s=s.replace('valitsit Liikkuvain','valitsit palvelumme')
 s=re.sub(r'\bLiikkuva\b(?! Auto Korjaamo)','Liikkuva Auto Korjaamo',s)
 p.write_text(s)
 for name in ['liikkuva_email.go','admin_standalone_email.go']:shutil.copy2(ROOT/'bundle'/name,dst/name)
 p=dst/'router.go';s=p.read_text()
 if 'mux.HandleFunc(standaloneEmailRoute' not in s:s=s.replace('mux.HandleFunc("/admin", admin.Show)','mux.HandleFunc("/admin", admin.Show)\n mux.HandleFunc(standaloneEmailRoute,admin.StandaloneEmail)')
 p.write_text(s)
 p=dst/'handlers_admin.go';s=p.read_text();start=s.index('func (h *AdminHandler) SendCustomEmail');end=s.index('\nfunc ',start+5);chunk=s[start:end]
 needle='leadID := strings.TrimSpace(r.FormValue("lead_id"))'
 chunk=chunk.replace(needle,'''r.Body=http.MaxBytesReader(w,r.Body,maxEmailAttachmentBytes+(1<<20))
 if strings.HasPrefix(r.Header.Get("Content-Type"),"multipart/form-data") {if err:=r.ParseMultipartForm(2<<20);err!=nil {http.Error(w,"Слишком большой размер письма",400);return};if r.MultipartForm!=nil {defer r.MultipartForm.RemoveAll()}}
 attachments,attachmentErr:=readEmailAttachments(r);if attachmentErr!=nil {http.Error(w,attachmentErr.Error(),400);return}
 '''+needle)
 chunk=chunk.replace('sendCustomClientEmail(h.cfg, record.Lead, subject, message)','sendCustomClientEmail(h.cfg, record.Lead, subject, message, attachments...)');s=s[:start]+chunk+s[end:];p.write_text(s)
 p=dst/'admin.html';s=p.read_text().replace('<div class="quick-links">','<div class="quick-links"><a href="/admin/email">Новое письмо</a>',1).replace('action="/admin/leads/send-custom-email"','action="/admin/leads/send-custom-email" enctype="multipart/form-data"')
 s=s.replace('<button type="submit" class="primary">Отправить письмо</button>','<label>Вложения (до 10 файлов, всего до 15 МБ)<input type="file" name="custom_email_attachments" multiple></label><button type="submit" class="primary">Отправить письмо</button>');p.write_text(s)
