#!/usr/bin/env python3
"""Read-only checks. Does not create leads, send email or trigger advertising events."""
import argparse,json,subprocess,urllib.request
p=argparse.ArgumentParser();p.add_argument('--url',default='https://liikkuvakorjaamo.fi');p.add_argument('--project',default='seventh-chassis-247718');p.add_argument('--region',default='europe-north1');a=p.parse_args();base=a.url.rstrip('/');failed=False
for path in ['/healthz','/','/en','/hinnasto','/vikadiagnostiikka','/palvelualue','/robots.txt','/sitemap.xml','/admin']:
 try:
  with urllib.request.urlopen(base+path,timeout=30) as r:body=r.read().decode('utf-8');status=r.status
  if status!=200:raise ValueError('HTTP '+str(status))
  if path=='/healthz':
   data=json.loads(body)
   if not data.get('ok') or data.get('site')!='liikkuvakorjaamo.fi':raise ValueError('Неверный сервис')
   print('Версия:',data.get('version'),'Ревизия:',data.get('revision'))
  if 'liikkuvautokorjaamo.fi' in body.lower():raise ValueError('Старый домен в HTML')
  if 'ZgotmplZ' in body:raise ValueError('Ошибка безопасного рендеринга шаблона')
  print('OK',path)
 except Exception as e:failed=True;print('ОШИБКА',path,str(e))
try:
 raw=subprocess.check_output(['gcloud','run','services','describe','liikkuva-landing','--project='+a.project,'--region='+a.region,'--format=json'],text=True)
 data=json.loads(raw)
 for traffic in data.get('status',{}).get('traffic',[]):
  if traffic.get('percent',0):print('Трафик:',traffic.get('percent'),'% →',traffic.get('revisionName'))
except (OSError,subprocess.CalledProcessError):print('Cloud Run трафик не прочитан. Запустите проверку в Cloud Shell под своим аккаунтом.')
raise SystemExit(1 if failed else 0)
