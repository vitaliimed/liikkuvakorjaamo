#!/usr/bin/env python3
"""Dedicated Firebase Hosting site -> existing Liikkuva Cloud Run. No AutoHelp edits."""
import argparse,json,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parent
p=argparse.ArgumentParser();p.add_argument('--project',default='seventh-chassis-247718');p.add_argument('--region',default='europe-north1');a=p.parse_args()
def run(*args):return subprocess.run(args,check=True,text=True)
site='liikkuva-'+a.project[:20]
folder=ROOT/'firebase-hosting';folder.mkdir(exist_ok=True);(folder/'public').mkdir(exist_ok=True)
config={'hosting':{'site':site,'public':'public','ignore':['firebase.json','**/.*','**/node_modules/**'],'rewrites':[{'source':'**','run':{'serviceId':'liikkuva-landing','region':a.region}}]}}
(folder/'firebase.json').write_text(json.dumps(config,indent=2))
print('В Firebase Console добавьте Firebase к существующему проекту '+a.project+'. Не создавайте другой Google Cloud проект.')
print('Отдельный Hosting site: '+site+'. Cloud Run и исходный AutoHelp не меняются.')
run('npx','--yes','firebase-tools','login','--no-localhost')
marker=folder/'site-created.json'
if not marker.exists():
 run('npx','--yes','firebase-tools','hosting:sites:create',site,'--project',a.project)
 marker.write_text(json.dumps({'project':a.project,'site':site}))
else:
 saved=json.loads(marker.read_text())
 if saved!={'project':a.project,'site':site}:raise SystemExit('Сохранён другой сайт Hosting; проверьте параметры.')
run('npx','--yes','firebase-tools','deploy','--only','hosting','--project',a.project,'--config',str(folder/'firebase.json'))
print('Откройте Firebase Console → Hosting → '+site+' → Add custom domain → liikkuvakorjaamo.fi.')
print('Добавьте выданные Firebase DNS-записи. www.liikkuvakorjaamo.fi добавьте как redirect на основной домен.')
print('Если Cloudflare управляет DNS, оставьте записи сайта DNS only до выпуска SSL. Записи MX для почты не удаляйте.')
