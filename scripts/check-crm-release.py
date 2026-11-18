#!/usr/bin/env python3
"""Read-only deployment inventory. Never prints secret/environment values."""
import json, subprocess

def cloud(*args):
    p = subprocess.run(['gcloud', *args, '--format=json'], capture_output=True, text=True)
    if p.returncode:
        raise RuntimeError('Не удалось прочитать Cloud Run. Проверьте вход и права gcloud.')
    return json.loads(p.stdout)

for project, service in [('liikkuva-korjaamo','liikkuva-preview'),('seventh-chassis-247718','autohelp-landing')]:
    try:
        state = cloud('run','services','describe',service,'--project='+project,'--region=europe-north1')
        print(json.dumps({'project':project,'service':service,'traffic':state.get('status',{}).get('traffic',[])},ensure_ascii=False,indent=2))
        revisions = {r['revisionName'] for r in state.get('status',{}).get('traffic',[]) if r.get('percent') and r.get('revisionName')}
        for revision in sorted(revisions):
            r = cloud('run','revisions','describe',revision,'--project='+project,'--region=europe-north1')
            c = r.get('spec',{}).get('containers',[{}])[0]
            env = {e['name']:e for e in c.get('env',[])}
            wanted = ['TELEGRAM_BOT_TOKEN','TELEGRAM_CHAT_ID','CRM_TELEGRAM_BOT_TOKEN','CRM_TELEGRAM_WEBHOOK_SECRET','CRM_OWNER_TELEGRAM_ID','CRM_INTERNAL_TOKEN','FIRESTORE_DATABASE','SITE_SETTINGS_COLLECTION']
            print(json.dumps({'revision':revision,'image':c.get('image'),'configuration':{k:('secret' if env[k].get('valueFrom') else 'configured') if k in env else 'missing' for k in wanted}},ensure_ascii=False,indent=2))
    except RuntimeError as e:
        print(project+': '+str(e))
