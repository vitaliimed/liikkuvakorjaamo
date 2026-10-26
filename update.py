#!/usr/bin/env python3
"""Build a release, deploy at zero traffic, check it, then promote that exact revision."""
import argparse,json,subprocess,time,urllib.request
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--project',default='seventh-chassis-247718');p.add_argument('--region',default='europe-north1');p.add_argument('--source',default=str(Path(__file__).resolve().parent/'source'));a=p.parse_args();src=Path(a.source).resolve()
def run(*args,cwd=None):return subprocess.run(args,check=True,cwd=cwd,text=True,capture_output=True).stdout
base=['--project='+a.project,'--region='+a.region,'--quiet']
def service():return json.loads(run('gcloud','run','services','describe','liikkuva-landing',*base,'--format=json'))
s=service()
if s.get('metadata',{}).get('labels',{}).get('brand')!='liikkuvakorjaamo':raise SystemExit('Метка сервиса не совпадает. Обновление заблокировано.')
if not (src/'liikkuva_brand.go').is_file() or 'liikkuvakorjaamo.fi' not in (src/'liikkuva_brand.go').read_text():raise SystemExit('Это не исходники Liikkuva Korjaamo.')
print('Проверка исходников…',flush=True);run('go','test','./...',cwd=src)
stamp=time.strftime('%Y%m%d-%H%M%S');image=a.region+'-docker.pkg.dev/'+a.project+'/liikkuva-images/site:'+stamp
print('Сборка Cloud Build…',flush=True);run('gcloud','builds','submit','--project='+a.project,'--tag='+image,'--timeout=1200s',str(src),'--quiet')
tag='check-'+stamp.lower()
print('Новая ревизия пока без пользовательского трафика…',flush=True)
run('gcloud','run','deploy','liikkuva-landing',*base,'--image='+image,'--no-traffic','--tag='+tag,'--format=json')
current=service();entry=next(x for x in current['status']['traffic'] if x.get('tag')==tag);url=entry['url'];revision=entry['revisionName']
with urllib.request.urlopen(url+'/healthz',timeout=45) as response:data=json.load(response)
if data.get('site')!='liikkuvakorjaamo.fi' or data.get('ok') is not True:raise SystemExit('Проверка не пройдена. Старый сайт продолжает получать трафик.')
with urllib.request.urlopen(url+'/',timeout=45) as response:
 landing=response.read().decode('utf-8')
 if 'liikkuvakorjaamo.fi' not in landing:raise SystemExit('Главная страница не прошла проверку. Старый сайт продолжает работать.')
run('gcloud','run','services','update-traffic','liikkuva-landing',*base,'--to-revisions='+revision+'=100')
result=service()
if sum(x.get('percent',0) for x in result['status']['traffic'] if x.get('revisionName')==revision)!=100:raise SystemExit('Проверьте распределение трафика в Cloud Run.')
run('gcloud','run','services','update-traffic','liikkuva-landing',*base,'--remove-tags='+tag)
print('Обновлено. 100% трафика: '+revision+'. Настройки, секреты и заявки сохранены.')
print('Предыдущие ревизии доступны в Cloud Run → Revisions → Manage traffic для отката.')
