#!/usr/bin/env python3
"""Configure a separate domain sender. Password never appears in process arguments."""
import argparse,email.utils,getpass,json,os,re,subprocess,tempfile
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--project',default='seventh-chassis-247718');p.add_argument('--region',default='europe-north1');a=p.parse_args()
def gc(*args,input=None):return subprocess.run(['gcloud',*args,'--project='+a.project,'--quiet'],input=input,text=True,capture_output=True,check=True).stdout
s=json.loads(gc('run','services','describe','liikkuva-landing','--region='+a.region,'--format=json'))
if s.get('metadata',{}).get('labels',{}).get('brand')!='liikkuvakorjaamo':raise SystemExit('Не найден новый сайт. Настройки не менялись.')
login=input('Brevo SMTP login (из Settings → SMTP & API): ').strip();password=getpass.getpass('Brevo SMTP key (ввод скрыт): ').strip()
address=input('Адрес отправителя [service@liikkuvakorjaamo.fi]: ').strip() or 'service@liikkuvakorjaamo.fi'
if not re.fullmatch(r'[^\s@]+@liikkuvakorjaamo\.fi',address) or not login or not password:raise SystemExit('Проверьте адрес нового домена и SMTP-данные.')
sa=s['spec']['template']['spec']['serviceAccountName']
refs={}
for key,value in [('SMTP_USERNAME',login),('SMTP_PASSWORD',password)]:
 name='liikkuva-'+key.lower().replace('_','-')
 found=subprocess.run(['gcloud','secrets','describe',name,'--project='+a.project,'--format=none'],capture_output=True).returncode==0
 if not found:gc('secrets','create',name,'--replication-policy=automatic')
 version=gc('secrets','versions','add',name,'--data-file=-','--format=value(name)',input=value).strip().split('/')[-1]
 gc('secrets','add-iam-policy-binding',name,'--member=serviceAccount:'+sa,'--role=roles/secretmanager.secretAccessor','--format=none');refs[key]=name+':'+version
# Update only these public settings; preserve every unrelated variable and secret reference.
if '~' in address:raise SystemExit('Недопустимый символ в адресе отправителя.')
updates={'SMTP_HOST':'smtp-relay.brevo.com','SMTP_PORT':'587','MAIL_FROM':email.utils.formataddr(('Liikkuva Auto Korjaamo',address)),'MAIL_REPLY_TO':address}
gc('run','services','update','liikkuva-landing','--region='+a.region,'--update-env-vars=^~^'+'~'.join(k+'='+v for k,v in updates.items()),'--update-secrets='+','.join(k+'='+v for k,v in refs.items()))
print('SMTP нового сайта настроен. В админке → Основные настройки укажите тот же адрес для связи.')
print('Проверьте реальную заявку со своей почтой и ответ на полученное письмо. Скрипт сам письма не отправляет.')
