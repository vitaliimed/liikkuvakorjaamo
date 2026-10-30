#!/usr/bin/env python3
import email.utils
import argparse, getpass, hashlib, json, os, re, secrets, shutil, subprocess, sys, tempfile, time, urllib.request, urllib.error
from pathlib import Path
from prepare import prepare, fingerprint

ROOT=Path(__file__).resolve().parent
DOMAIN='liikkuvakorjaamo.fi'; SERVICE='liikkuva-landing'; DATABASE='liikkuvakorjaamo'; QUEUE='liikkuva-lead-processing'
SHARED={'TELEGRAM_BOT_TOKEN','TELEGRAM_CHAT_ID','SMTP_HOST','SMTP_PORT','SMTP_USERNAME','SMTP_PASSWORD','MAIL_FROM','MAIL_REPLY_TO','MAIL_COPY_TO'}
CONTACTS={'Phone','ServiceEmail','WhatsAppNumber','BusinessName','YTunnus','BaseCity','WorkingHoursFI','WorkingHoursEN'}

class Installer:
 def __init__(self,args):
  self.a=args;self.project=args.project;self.region=args.region;self.source_service='autohelp-landing';self.account='liikkuva-runtime@'+self.project+'.iam.gserviceaccount.com';self.bucket=self.project+'-liikkuva-files';self.redact=[]
 def run(self,args,input=None,optional=False,cwd=None):
  r=subprocess.run(args,input=input,text=True,capture_output=True,cwd=cwd)
  permission=any(word in (r.stderr+r.stdout).lower() for word in ['permission_denied','permission denied','do not have permission','does not have','not authorized','access denied','forbidden','403'])
  if r.returncode and (not optional or permission):
   detail=r.stderr or r.stdout
   for value in self.redact:detail=detail.replace(value,'[скрыто]')
   raise RuntimeError(detail[-1800:] or 'Команда завершилась с ошибкой')
  return r
 def gc(self,*args,**kw):return self.run(['gcloud',*args,'--project='+self.project,'--quiet'],**kw)
 def describe(self,service,optional=False):
  r=self.gc('run','services','describe',service,'--region='+self.region,'--format=json',optional=optional)
  return json.loads(r.stdout) if r.returncode==0 else None
 def ensure(self,read,create,label):
  if self.gc(*read,optional=True).returncode:
   print(label,flush=True);self.gc(*create)
 def acl(self,reference):
  match=re.fullmatch(r'projects/([^/]+)/secrets/([^/]+)',reference);p,n=match.groups() if match else (self.project,reference)
  self.run(['gcloud','secrets','add-iam-policy-binding',n,'--project='+p,'--member=serviceAccount:'+self.account,'--role=roles/secretmanager.secretAccessor','--quiet','--format=none'])
 def secret(self,name,value=None,rotate=False):
  if not name.startswith('liikkuva-'):raise RuntimeError('Запись секрета другого сайта заблокирована')
  exists=self.gc('secrets','describe',name,'--format=none',optional=True).returncode==0
  if not exists:self.gc('secrets','create',name,'--replication-policy=automatic')
  versions=self.gc('secrets','versions','list',name,'--filter=state:ENABLED','--sort-by=~createTime','--limit=1','--format=value(name)').stdout.strip()
  if rotate or not versions:
   value=value or secrets.token_urlsafe(40);self.redact.append(value)
   self.gc('secrets','versions','add',name,'--data-file=-',input=value)
   versions=self.gc('secrets','versions','list',name,'--filter=state:ENABLED','--sort-by=~createTime','--limit=1','--format=value(name)').stdout.strip()
  if not versions:raise RuntimeError('Не удалось создать версию секрета '+name)
  self.acl(name);return name+':'+versions.split('/')[-1]
 def api(self,url,method='GET',data=None,missing=False):
  token=self.gc('auth','print-access-token').stdout.strip();self.redact.append(token)
  req=urllib.request.Request(url,data=json.dumps(data).encode() if data is not None else None,method=method,headers={'Authorization':'Bearer '+token,'Content-Type':'application/json'})
  try:
   with urllib.request.urlopen(req,timeout=45) as r:return json.load(r)
  except urllib.error.HTTPError as e:
   if missing and e.code==404:return None
   raise RuntimeError('Google Cloud API: HTTP '+str(e.code)) from None
 def check_sheet(self):
  url=input('URL НОВОГО Apps Script (заканчивается /exec): ').strip();token=getpass.getpass('WEBHOOK_TOKEN новой таблицы (ввод скрыт): ').strip();self.redact.append(token)
  if not re.fullmatch(r'https://script\.google\.com/macros/s/[A-Za-z0-9_-]+/exec',url) or len(token)<32:raise RuntimeError('Некорректный URL или токен новой таблицы')
  payload={'site':DOMAIN,'script_token':token,'action':'check_connection'}
  req=urllib.request.Request(url,data=json.dumps(payload).encode(),headers={'Content-Type':'application/json'})
  with urllib.request.urlopen(req,timeout=45) as r:result=json.load(r)
  if result.get('ok') is not True or result.get('configured') is not True or result.get('site')!=DOMAIN:raise RuntimeError('Новая таблица не прошла проверку. Убедитесь, что setupNewLiikkuva выполнен и Web app развёрнут')
  return url,token
 def source_env(self,description):
  containers=description['spec']['template']['spec']['containers'];container=next((c for c in containers if c.get('name')=='autohelp-landing-1'),None)
  if container is None:
   matches=[c for c in containers if any(x.get('name')=='TELEGRAM_BOT_TOKEN' for x in c.get('env',[]))]
   if len(matches)!=1:raise RuntimeError('Не удалось однозначно определить контейнер AutoHelp')
   container=matches[0]
  env={x['name']:x for x in container.get('env',[])}
  if not all(k in env for k in ['TELEGRAM_BOT_TOKEN','TELEGRAM_CHAT_ID']):raise RuntimeError('В AutoHelp не найдены настройки Telegram; изменений не выполнено')
  aliases={}
  for pair in description['spec']['template'].get('metadata',{}).get('annotations',{}).get('run.googleapis.com/secrets','').split(','):
   if ':' in pair:k,v=pair.split(':',1);aliases[k]=v
  return env,aliases
 def install(self):
  if not shutil.which('gcloud'):raise RuntimeError('Запускайте этот файл в Google Cloud Shell: https://shell.cloud.google.com')
  original=self.describe(self.source_service);env,aliases=self.source_env(original);before=json.dumps(original['spec'],sort_keys=True)
  existing=self.describe(SERVICE,optional=True)
  if existing:raise RuntimeError('Сервис уже существует. Для обновления используйте update.py; повторная первичная установка заблокирована.')
  sheet_url,sheet_token=self.check_sheet()
  target=Path(self.a.target).expanduser().resolve()
  if target.exists():
   raise RuntimeError('Новая папка уже существует. Используйте --target с другим именем; чужие изменения не перезаписываются')
  print('1/6 Создаю отдельную копию актуального AutoHelp…',flush=True)
  if self.a.source:prepare(Path(self.a.source),target)
  else:shutil.copytree(ROOT/"source",target)
  if not shutil.which('go'):raise RuntimeError('В Cloud Shell не найден Go. Копия сохранена; установите Go перед повторным запуском')
  self.run(['gofmt','-w',*[str(p) for p in target.glob('*.go')]])
  print('2/6 Проверяю Go-код. Облачные ресурсы ещё не изменяются…',flush=True)
  self.run(['go','test','./...'],cwd=target)
  self.run(['go','build','-buildvcs=false','-o',str(target/'server-check'),'.'],cwd=target)
  (target/'server-check').unlink()
  print('3/6 Создаю отдельные ресурсы нового сайта…',flush=True)
  self.gc('services','enable','run.googleapis.com','cloudbuild.googleapis.com','artifactregistry.googleapis.com','firestore.googleapis.com','cloudtasks.googleapis.com','storage.googleapis.com','secretmanager.googleapis.com')
  self.ensure(['iam','service-accounts','describe',self.account,'--format=none'],['iam','service-accounts','create','liikkuva-runtime','--display-name=Liikkuva runtime'],'Отдельная сервисная учётная запись…')
  self.ensure(['firestore','databases','describe','--database='+DATABASE,'--format=none'],['firestore','databases','create','--database='+DATABASE,'--location='+self.region,'--type=firestore-native','--delete-protection'],'Отдельная база Firestore…')
  self.gc('projects','add-iam-policy-binding',self.project,'--member=serviceAccount:'+self.account,'--role=roles/datastore.user','--condition=expression=resource.name=="projects/'+self.project+'/databases/'+DATABASE+'",title=LiikkuvaDatabaseOnly','--format=none')
  self.ensure(['storage','buckets','describe','gs://'+self.bucket,'--format=none'],['storage','buckets','create','gs://'+self.bucket,'--location='+self.region,'--uniform-bucket-level-access','--public-access-prevention'],'Закрытое хранилище фотографий…')
  self.gc('storage','buckets','add-iam-policy-binding','gs://'+self.bucket,'--member=serviceAccount:'+self.account,'--role=roles/storage.objectAdmin','--format=none')
  self.ensure(['tasks','queues','describe',QUEUE,'--location='+self.region,'--format=none'],['tasks','queues','create',QUEUE,'--location='+self.region,'--max-concurrent-dispatches=1','--max-attempts=20'],'Отдельная очередь…')
  self.gc('tasks','queues','update',QUEUE,'--location='+self.region,'--max-concurrent-dispatches=1','--max-dispatches-per-second=1','--max-attempts=20','--min-backoff=10s','--max-backoff=300s')
  self.gc('projects','add-iam-policy-binding',self.project,'--member=serviceAccount:'+self.account,'--role=roles/cloudtasks.enqueuer','--condition=None','--format=none')
  self.ensure(['artifacts','repositories','describe','liikkuva-images','--location='+self.region,'--format=none'],['artifacts','repositories','create','liikkuva-images','--location='+self.region,'--repository-format=docker'],'Хранилище сборок…')
  plain={};refs={}
  for key in SHARED:
   entry=env.get(key,{})
   if 'value' in entry and entry['value']:
    value=entry['value']
    if key in {'TELEGRAM_BOT_TOKEN','SMTP_PASSWORD','SMTP_USERNAME'}:refs[key]=self.secret('liikkuva-'+key.lower().replace('_','-'),value,True)
    else:
     if key=='MAIL_FROM':
      _,address=email.utils.parseaddr(value)
      if address:value=email.utils.formataddr(('Liikkuva Auto Korjaamo',address))
     plain[key]=value
   elif 'secretKeyRef' in entry.get('valueFrom',{}):
    ref=entry['valueFrom']['secretKeyRef'];name=aliases.get(ref['name'],ref['name']);self.acl(name);refs[key]=name+':'+ref.get('key','latest')
  refs.update({'ADMIN_PASSWORD':self.secret('liikkuva-admin-password'),'INTERNAL_TASK_TOKEN':self.secret('liikkuva-internal-task-token'),'REVIEW_TOKEN_SECRET':self.secret('liikkuva-review-token-secret'),'GOOGLE_APPS_SCRIPT_TOKEN':self.secret('liikkuva-sheets-token',sheet_token,True)})
  image=self.region+'-docker.pkg.dev/'+self.project+'/liikkuva-images/site:'+time.strftime('%Y%m%d-%H%M%S')
  print('4/6 Сборка Cloud Build — может занять несколько минут…',flush=True)
  self.gc('builds','submit','--tag='+image,'--timeout=1200s',str(target))
  common=['run','deploy',SERVICE,'--region='+self.region,'--image='+image,'--service-account='+self.account,'--allow-unauthenticated','--port=8080','--memory=512Mi','--cpu=1','--min-instances=0','--max-instances=5','--concurrency=20','--timeout=180s','--labels=brand=liikkuvakorjaamo','--format=json']
  with tempfile.TemporaryDirectory(prefix='liikkuva-install-') as folder:
   file=Path(folder)/'env.json';file.touch(mode=0o600)
   if not existing:
    file.write_text(json.dumps({'LIIKKUVA_BOOTSTRAP':'1','SITE_URL':'https://'+DOMAIN}))
    self.gc(*common,'--env-vars-file='+str(file))
   url=self.describe(SERVICE)['status']['url'].rstrip('/')
   if not url.startswith('https://liikkuva-landing-') or not url.endswith('.run.app'):raise RuntimeError('Неожиданный адрес нового сервиса')
   plain.update({'SITE_URL':'https://'+DOMAIN,'DEFAULT_LANGUAGE':'fi','FIRESTORE_PROJECT_ID':self.project,'FIRESTORE_LEADS_COLLECTION':'liikkuva_leads','FIRESTORE_REVIEWS_COLLECTION':'liikkuva_reviews','SITE_SETTINGS_COLLECTION':'liikkuva_settings','GCS_BUCKET_NAME':self.bucket,'GCS_PHOTO_PREFIX':'liikkuva-photos','GCS_MAX_PHOTOS':'0','CLOUD_TASKS_PROJECT_ID':self.project,'CLOUD_TASKS_LOCATION':self.region,'CLOUD_TASKS_QUEUE':QUEUE,'INTERNAL_PROCESS_LEAD_URL':url+'/internal/process-lead','GOOGLE_APPS_SCRIPT_URL':sheet_url})
   file.write_text(json.dumps(plain));print('5/6 Запускаю рабочий сервис с новым хранением…',flush=True)
   deployed=json.loads(self.gc(*common,'--env-vars-file='+str(file),'--set-secrets='+','.join(k+'='+v for k,v in sorted(refs.items()))).stdout)
   revision=deployed['status']['latestReadyRevisionName']
   self.gc('run','services','update-traffic',SERVICE,'--region='+self.region,'--to-revisions='+revision+'=100')
  base='https://firestore.googleapis.com/v1/projects/'+self.project+'/databases/'
  source_collection=env.get('SITE_SETTINGS_COLLECTION',{}).get('value','site_settings');source_db=env.get('FIRESTORE_DATABASE_ID',{}).get('value','(default)')
  target_doc=base+DATABASE+'/documents/liikkuva_settings/landing'
  if not self.api(target_doc,missing=True):
   try:old=self.api(base+source_db+'/documents/'+source_collection+'/landing',missing=True)
   except RuntimeError:old={}
   fields={k:v for k,v in (old or {}).get('fields',{}).items() if k in CONTACTS}
   fields.update({'SectionVisibilityConfigured':{'booleanValue':True},'StatsEnabled':{'booleanValue':False}})
   for key in ['Services','Process','Cases','Capability','Areas','Testimonials','FAQ','FinalCTA']:fields[key+'Enabled']={'booleanValue':True}
   self.api(target_doc,'PATCH',{'fields':fields})
  seo=json.loads((ROOT/'bundle/seo-defaults.json').read_text())
  from urllib.parse import urlencode
  update=urlencode([('updateMask.fieldPaths',key) for key in seo])
  self.api(target_doc+'?'+update,'PATCH',{'fields':{key:{'stringValue':value} for key,value in seo.items()}})
  print('6/6 Проверка адреса и исходного AutoHelp…',flush=True)
  with urllib.request.urlopen(url+'/',timeout=45) as response:
   landing=response.read().decode('utf-8')
   if 'liikkuvakorjaamo.fi' not in landing or 'G-MXX9XCNCRB' not in landing:raise RuntimeError('Главная страница не прошла проверку')
  if json.dumps(self.describe(self.source_service)['spec'],sort_keys=True)!=before:raise RuntimeError('AutoHelp изменился параллельно. Этот установщик его не изменяет; проверьте параллельные действия')
  with urllib.request.urlopen(url+'/api/health',timeout=45) as r:health=json.load(r)
  if health.get('site')!=DOMAIN or health.get('ok') is not True:raise RuntimeError('Проверка нового сервиса не прошла')
  (ROOT/'installation-result.json').write_text(json.dumps({'project':self.project,'region':self.region,'service':SERVICE,'url':url,'target':str(target),'image':image,'admin_secret':refs['ADMIN_PASSWORD']},indent=2))
  print('\nСайт работает: '+url+'\nАдминка: '+url+'/admin')
  print('Пароль: gcloud secrets versions access '+refs['ADMIN_PASSWORD'].split(':')[1]+' --secret=liikkuva-admin-password --project='+self.project)
  print('Следующий шаг: python3 hosting.py (домен через Firebase Hosting)')
 def domain(self):
  current=json.loads(self.gc('beta','run','domain-mappings','list','--region='+self.region,'--format=json').stdout)
  mapping=next((x for x in current if x.get('metadata',{}).get('name')==DOMAIN),None)
  if mapping and mapping.get('spec',{}).get('routeName')!=SERVICE:raise RuntimeError('Домен связан с другим сервисом. Перезапись запрещена')
  print('Если Google требует подтверждение владения: gcloud domains verify '+DOMAIN)
  if not mapping:self.gc('beta','run','domain-mappings','create','--domain='+DOMAIN,'--service='+SERVICE,'--region='+self.region)
  data=json.loads(self.gc('beta','run','domain-mappings','describe','--domain='+DOMAIN,'--region='+self.region,'--format=json').stdout)
  for row in data.get('status',{}).get('resourceRecords',[]):print(row.get('name') or '@',row.get('type',''),row.get('rrdata',''))
  print('Добавьте эти записи в DNS домена. HTTPS включится после распространения DNS и выдачи сертификата Google.')
  print('Если Google требует подтверждение владения: gcloud domains verify '+DOMAIN)
  print('Cloud Run domain mapping имеет статус Preview Google; для SLA используйте HTTPS Load Balancer.')

def main():
 p=argparse.ArgumentParser();p.add_argument('action',nargs='?',default='install',choices=['install','domain']);p.add_argument('--source',default=None);p.add_argument('--target',default=str(Path.home()/'lending/LiikkuvaKorjaamo'));p.add_argument('--project',default='seventh-chassis-247718');p.add_argument('--region',default='europe-north1');args=p.parse_args()
 try:
  if not re.fullmatch(r'[a-z][a-z0-9-]+',args.project):raise RuntimeError('Некорректный ID проекта')
  getattr(Installer(args),args.action)()
 except Exception as e:print('Остановлено: '+str(e),file=sys.stderr);print('Исходный AutoHelp не изменялся. Пароли и токены не отправляйте в чат.',file=sys.stderr);sys.exit(1)
if __name__=='__main__':main()
