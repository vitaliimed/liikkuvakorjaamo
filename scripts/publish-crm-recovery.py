#!/usr/bin/env python3
"""Safe release of CRM password recovery to the existing Liikkuva Cloud Run.

Run from authenticated Google Cloud Shell in the checked-out feature branch.
Preserves ALL existing environment/secret refs and LIIKKUVA_BOOTSTRAP state,
checks exact revision and public domain, and rolls back Cloud Run traffic on
failure. Does not submit a real password reset, create accounts or send email.
"""
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import urllib.request
import urllib.error

ROOT=Path(__file__).resolve().parents[1]
PROJECT="liikkuva-korjaamo"
REGION="europe-north1"
SERVICE="liikkuva-preview"
DOMAIN="https://liikkuvakorjaamo.fi"
TAG="crm-password-reset"
EXPECTED="2026.11.12-production-candidate"

def run(*argv, capture=False, cwd=ROOT):
    p=subprocess.run(argv, cwd=cwd, text=True, stdout=subprocess.PIPE if capture else None,
                     stderr=subprocess.PIPE if capture else None)
    if p.returncode:
        raise RuntimeError("Failed: "+" ".join(argv[:4])+"; "+(p.stderr or "")[-350:])
    return p.stdout or ""

def cloud(*args, capture=False):
    return run("gcloud", *args, "--project="+PROJECT, capture=capture)

def service():
    return json.loads(cloud("run","services","describe",SERVICE,"--region="+REGION,"--format=json",capture=True))

def allocation(d):
    result={}
    for row in d.get("status",{}).get("traffic",[]):
        n=int(row.get("percent") or 0)
        if n:
            name=row.get("revisionName")
            if not name:
                raise RuntimeError("Cloud Run has unresolved traffic")
            result[name]=result.get(name,0)+n
    if sum(result.values())!=100:
        raise RuntimeError("Production traffic is not resolved to 100%")
    return result

def revision(name):
    return json.loads(cloud("run","revisions","describe",name,"--region="+REGION,"--format=json",capture=True))

def pod_spec(d):
    return d.get("spec",{})

def env_snapshot(d):
    out={}
    for container in pod_spec(d).get("containers",[]):
        for entry in container.get("env",[]):
            out[entry["name"]]={k:v for k,v in entry.items() if k!="name"}
    return out

def fetch(url, method="GET", body=None):
    req=urllib.request.Request(
        url, method=method, data=body,
        headers={"Cache-Control":"no-cache","X-CRM-Request":"1","Content-Type":"application/json"})
    with urllib.request.urlopen(req,timeout=30) as response:
        if response.status!=200:
            raise RuntimeError(url+" HTTP "+str(response.status))
        return response.read(300000).decode("utf-8")

def smoke(url, expected_revision):
    health=json.loads(fetch(url+"/api/health?crm-reset="+str(int(time.time()))))
    if health.get("revision")!=expected_revision or health.get("version")!=EXPECTED:
        raise RuntimeError("Unexpected health: "+str({k:health.get(k) for k in ("revision","version")}))
    for path in ("/","/en","/admin","/crm/","/crm/password-reset.js","/crm/app.js"):
        content=fetch(url+path)
        if path=="/crm/password-reset.js" and ("function crmResetPassword" not in content or
                                               "crmGeneratePassword" not in content):
            raise RuntimeError("CRM password recovery script is not serving")
        if path=="/crm/app.js" and "forgotlink" not in content:
            raise RuntimeError("CRM login form is not updated")
    response=json.loads(fetch(url+"/crm/api/password/forgot","POST",b'{"email":"invalid"}'))
    if "message" not in response:
        raise RuntimeError("Password recovery endpoint failed")
    print("PASS: "+url+" (site, CRM, password reset scripts, no-account response)")

def rollback(baseline):
    data=",".join(name+"="+str(percent) for name,percent in baseline.items())
    cloud("run","services","update-traffic",SERVICE,"--region="+REGION,"--to-revisions="+data)
    print("ROLLBACK: previous Cloud Run allocation restored")

def main():
    branch=run("git","branch","--show-current",capture=True).strip()
    if branch!="feat/crm-email-password-recovery":
        raise RuntimeError("Switch to feat/crm-email-password-recovery before running")
    if run("git","status","--porcelain",capture=True).strip():
        raise RuntimeError("Uncommitted local changes; commit/stash before release")
    print("===== TESTS =====")
    run("go","test","./...",cwd=ROOT/"source")
    run("go","vet","./...",cwd=ROOT/"source")
    run("node","--check","source/crm/web/app.js")
    run("node","--check","source/crm/web/password-reset.js")
    run("node","--test","tests/crm-ui.test.cjs","tests/crm-password-reset.test.cjs")
    print("===== SAVE PRODUCTION BASELINE =====")
    before=service()
    old=allocation(before)
    if len(old)!=1:
        raise RuntimeError("Multiple production revisions; manual release required")
    old_revision=next(iter(old))
    prod=revision(old_revision)
    initial_env=env_snapshot(prod)
    required=("SMTP_HOST","SMTP_USERNAME","SMTP_PASSWORD","MAIL_FROM","FIRESTORE_PROJECT_ID")
    missing=[k for k in required if k not in initial_env]
    if missing:
        raise RuntimeError("Production missing required configured values: "+", ".join(missing))
    backup=Path.home()/".liikkuva-releases"
    backup.mkdir(mode=0o700,parents=True,exist_ok=True)
    os.chmod(backup,0o700)
    file=backup/("crm-password-reset-"+str(int(time.time()))+".json")
    fd=os.open(file,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
    with os.fdopen(fd,"w") as handle:
        json.dump(before,handle,indent=2)
    print("Old allocation:",old,"private snapshot:",file)
    print("===== DEPLOY NO-TRAFFIC CRM RECOVERY =====")
    cloud("run","deploy",SERVICE,"--region="+REGION,"--source="+str(ROOT/"source"),
          "--no-traffic","--tag="+TAG,
          "--update-env-vars=CRM_PUBLIC_URL="+DOMAIN)
    staged=service()
    if allocation(staged)!=old:
        raise RuntimeError("Traffic changed during deployment; inspect manually")
    selected=[row for row in staged["status"]["traffic"] if row.get("tag")==TAG]
    if len(selected)!=1:
        raise RuntimeError("CRM password reset preview tag not found")
    row=selected[0]
    candidate=row["revisionName"]
    url=row["url"].rstrip("/")
    rev=revision(candidate)
    new_env=env_snapshot(rev)
    old_keep={k:v for k,v in initial_env.items() if k!="CRM_PUBLIC_URL"}
    new_keep={k:v for k,v in new_env.items() if k!="CRM_PUBLIC_URL"}
    if old_keep!=new_keep:
        print("Env differences (names only):",
              sorted(k for k in set(old_keep)|set(new_keep) if old_keep.get(k)!=new_keep.get(k)))
        raise RuntimeError("Candidate would change other production env/secret references")
    if new_env.get("CRM_PUBLIC_URL",{}).get("value")!=DOMAIN:
        raise RuntimeError("Canonical CRM_PUBLIC_URL not set")
    old_sa=prod.get("spec",{}).get("serviceAccountName")
    new_sa=rev.get("spec",{}).get("serviceAccountName")
    if old_sa!=new_sa:
        raise RuntimeError("Service account changed unexpectedly")
    print("===== PREVIEW SMOKE =====")
    smoke(url,candidate)
    if allocation(service())!=old:
        raise RuntimeError("Production traffic changed while preview was checked")
    print("===== PROMOTE TO 100% =====")
    cloud("run","services","update-traffic",SERVICE,"--region="+REGION,
          "--to-revisions="+candidate+"=100")
    try:
        if allocation(service())!={candidate:100}:
            raise RuntimeError("Cloud Run did not confirm promotion")
        smoke(DOMAIN,candidate)
    except Exception:
        rollback(old)
        raise
    print("SUCCESS: New CRM recovery published on 100% traffic")
    print("Public CRM: "+DOMAIN+"/crm/")
    print("Production revision: "+candidate)
    print("Previous revision for rollback: "+old_revision)
    print("Bootstrap unchanged: "+str(new_env.get("LIIKKUVA_BOOTSTRAP",{}).get("value","not set")))
    print("NEXT: use 'Забыли пароль?' with an already registered CRM email.")
    print("NOTE: SMTP inbox delivery is only proven by receiving the actual reset letter.")

if __name__=="__main__":
    try:
        main()
    except (RuntimeError,OSError,ValueError,urllib.error.URLError) as error:
        print("CRM RECOVERY RELEASE STOPPED:",str(error),file=sys.stderr)
        sys.exit(1)
