#!/usr/bin/env python3
import json,pathlib,subprocess,tempfile,time,urllib.parse
ROOT=pathlib.Path(__file__).resolve().parents[1]
PROJECT='liikkuva-korjaamo';DATABASE='liikkuvakorjaamo';SERVICE='liikkuva-preview';REGION='europe-north1';TAG='crm-access-v2'
def run(args,capture=False,cwd=ROOT):
 return subprocess.run(args,cwd=cwd,check=True,text=True,stdout=subprocess.PIPE if capture else None).stdout
def cloud(args,capture=False):return run(['gcloud',*args,'--project',PROJECT],capture)
def service():return json.loads(cloud(['run','services','describe',SERVICE,'--region',REGION,'--format=json'],True))
def traffic(s):
 result={}
 for x in s['status'].get('traffic',[]):
  if x.get('percent',0):
   if not x.get('revisionName'):raise SystemExit('Unresolved production revision')
   result[x['revisionName']]=result.get(x['revisionName'],0)+x['percent']
 if sum(result.values())!=100:raise SystemExit('Expected existing traffic 100%')
 return result
def key(x):return tuple((f['fieldPath'],f.get('order','')) for f in x.get('fields',[]) if f['fieldPath']!='__name__')
def indexes():return json.loads(cloud(['firestore','indexes','composite','list','--database',DATABASE,'--format=json'],True))
def match(a,b):return '/collectionGroups/'+b['collectionGroup']+'/indexes/' in a.get('name','') and key(a)==key(b)
def main():
 if run(['git','status','--porcelain'],True).strip():raise SystemExit('Save local changes before deployment; working tree must be clean')
 before=service();baseline=traffic(before);print('Preserving traffic:',baseline,flush=True)
 run(['go','test','./...'],cwd=ROOT/'source')
 with tempfile.TemporaryDirectory() as tmp:run(['go','build','-o',str(pathlib.Path(tmp)/'server'),'.'],cwd=ROOT/'source')
 wanted=json.loads((ROOT/'source/crm/firestore.indexes.json').read_text())['indexes'];existing=indexes()
 for item in wanted:
  if any(match(x,item) for x in existing):continue
  args=['firestore','indexes','composite','create','--database',DATABASE,'--collection-group',item['collectionGroup'],'--query-scope=collection','--quiet','--async']
  for f in item['fields']:args+=['--field-config','field-path='+f['fieldPath']+',order='+f['order'].lower()]
  cloud(args)
 for attempt in range(90):
  existing=indexes()
  if all(any(match(x,item) and x.get('state')=='READY' for x in existing) for item in wanted):break
  print('Waiting for indexes; documents are not deleted',flush=True);time.sleep(20)
 else:raise SystemExit('Indexes pending; rerun later. Service not deployed')
 if traffic(service())!=baseline:raise SystemExit('Traffic changed during checks; inspect before continuing')
 old_url=next((x['url'] for x in before['status'].get('traffic',[]) if x.get('tag')=='crm-preview' and x.get('url')),before['status']['url'])
 host=urllib.parse.urlsplit(old_url).netloc.split('---')[-1];url='https://'+TAG+'---'+host
 cloud(['run','deploy',SERVICE,'--region',REGION,'--source',str(ROOT/'source'),'--no-traffic','--tag',TAG,'--quiet','--update-env-vars','CRM_MAIL_SEND_ENABLED=0,CRM_PUBLIC_URL='+url])
 after=service()
 if traffic(after)!=baseline:raise SystemExit('Traffic differs from baseline. Inspect Cloud Run immediately; no automatic traffic command issued')
 tagged=next(x for x in after['status']['traffic'] if x.get('tag')==TAG)
 print('Preview revision:',tagged.get('revisionName'));print('CRM:',tagged['url']+'/crm/');print('Production traffic unchanged; email sending disabled')
if __name__=='__main__':main()
