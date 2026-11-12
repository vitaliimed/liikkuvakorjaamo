#!/usr/bin/env python3
"""Correct Brevo SMTP on Liikkuva Cloud Run without rebuilding the Docker image.

Run from authenticated Cloud Shell:
    python3 scripts/brevo-no-build.py stage
    # Test an actual email using 'Forgot password?' on the printed PREVIEW URL.
    python3 scripts/brevo-no-build.py publish

This creates a configuration-only Cloud Run revision using the currently
serving image. It preserves old traffic until email receipt is confirmed.
"""
from __future__ import annotations

import getpass
import json
import os
from pathlib import Path
import smtplib
import subprocess
import sys
import time
import urllib.error
import urllib.request

PROJECT="liikkuva-korjaamo"
SERVICE="liikkuva-preview"
REGION="europe-north1"
TAG="brevo-smtp-preview"
DOMAIN="https://liikkuvakorjaamo.fi"
DIR=Path.home()/".liikkuva-releases"
STATE=DIR/"brevo-no-build.json"
SECRETS={"SMTP_USERNAME":"liikkuva-brevo-smtp-login",
         "SMTP_PASSWORD":"liikkuva-brevo-smtp-key"}

def run(*cmd, stdin=None):
    p=subprocess.run(cmd,input=stdin,text=True,stdout=subprocess.PIPE,
                     stderr=subprocess.PIPE)
    if p.returncode:
        raise RuntimeError("Command failed: "+ " ".join(cmd[:5]) +
                           " (exit "+str(p.returncode)+") "+p.stderr[-360:])
    return p.stdout.strip()

def cloud(*args):
    return run("gcloud",*args,"--project="+PROJECT)

def data(*args):
    return json.loads(cloud(*args,"--format=json"))

def service():
    return data("run","services","describe",SERVICE,"--region="+REGION)

def shares(s):
    result={}
    for row in s["status"].get("traffic",[]):
        p=int(row.get("percent") or 0)
        if p:
            n=row.get("revisionName")
            if not n: raise RuntimeError("Unresolved Cloud Run revision")
            result[n]=result.get(n,0)+p
    if sum(result.values())!=100:
        raise RuntimeError("Production traffic must total 100%")
    return result

def revision(name):
    return data("run","revisions","describe",name,"--region="+REGION)

def vars(rev):
    return {e["name"]:{k:v for k,v in e.items() if k!="name"}
            for c in rev.get("spec",{}).get("containers",[])
            for e in c.get("env",[])}

def save(state):
    DIR.mkdir(parents=True,exist_ok=True,mode=0o700)
    os.chmod(DIR,0o700)
    fd=os.open(STATE,os.O_WRONLY|os.O_CREAT|os.O_TRUNC,0o600)
    with os.fdopen(fd,"w") as handle:json.dump(state,handle,indent=2)

def read_state():
    if not STATE.is_file():
        raise RuntimeError("No staged SMTP revision. Run stage first.")
    return json.loads(STATE.read_text())

def secret_exists(name):
    p=subprocess.run(["gcloud","secrets","describe",name,"--project="+PROJECT],
                     stdout=subprocess.DEVNULL,stderr=subprocess.PIPE,text=True)
    if p.returncode==0:return True
    if "NOT_FOUND" in p.stderr or "not found" in p.stderr.lower():
        return False
    raise RuntimeError("Cannot inspect secret "+name+"; permissions may be missing")

def secret_write(name,value,sa):
    if not secret_exists(name):
        cloud("secrets","create",name,"--replication-policy=automatic")
    cloud("secrets","add-iam-policy-binding",name,
          "--member=serviceAccount:"+sa,
          "--role=roles/secretmanager.secretAccessor","--quiet")
    # The secret value is passed on stdin, never in process arguments or shell history.
    run("gcloud","secrets","versions","add",name,"--data-file=-",
        "--project="+PROJECT,stdin=value)

def smtp_check(login,key):
    try:
        with smtplib.SMTP("smtp-relay.brevo.com",587,timeout=14) as smtp:
            smtp.ehlo()
            smtp.starttls()
            smtp.ehlo()
            smtp.login(login,key)
        print("PASS: Brevo accepted SMTP credentials from Cloud Shell")
    except smtplib.SMTPAuthenticationError as exc:
        raise RuntimeError("Brevo rejected the SMTP login/key (code "+
                           str(exc.smtp_code)+"). Verify SMTP login, SMTP key and IP allowlist.")
    except (OSError,smtplib.SMTPException) as exc:
        print("WARNING: Cloud Shell SMTP connectivity could not be validated ("+
              type(exc).__name__+"). The real email test from Cloud Run is mandatory.")

def fetch(url):
    with urllib.request.urlopen(
        urllib.request.Request(url,headers={"Cache-Control":"no-cache"}),
        timeout=35) as res:
        if res.status!=200:raise RuntimeError("HTTP "+str(res.status)+": "+url)
        return res.read(260000).decode("utf-8")

def check(url,expected):
    h=json.loads(fetch(url+"/api/health"))
    if h.get("revision")!=expected or not h.get("ok"):
        raise RuntimeError("Preview does not serve the expected revision")
    app=fetch(url+"/crm/app.js")
    if "forgotlink" not in app:raise RuntimeError("CRM forgot-password UI missing")
    reset=fetch(url+"/crm/password-reset.js")
    if "crmResetPassword" not in reset:raise RuntimeError("Password recovery JS missing")
    print("PASS: CRM recovery and health at",url)

def stage():
    existing=service()
    prev=shares(existing)
    if len(prev)!=1:raise RuntimeError("Multiple serving revisions; inspect before proceeding")
    old=next(iter(prev))
    rev=revision(old)
    cs=rev.get("spec",{}).get("containers",[])
    image=cs[0].get("image") if cs else ""
    sa=rev.get("spec",{}).get("serviceAccountName")
    if not image or not sa:
        raise RuntimeError("Cannot resolve existing image and runtime service account")
    username=input("Brevo SMTP Login (from Brevo SMTP page): ").strip()
    secret=getpass.getpass("New SMTP key (hidden, NOT API key): ").strip()
    if "@" not in username or not secret:
        raise RuntimeError("SMTP login/key missing. Create them in Brevo SMTP & API settings.")
    smtp_check(username,secret)
    for env,name in SECRETS.items():
        secret_write(name,username if env=="SMTP_USERNAME" else secret,sa)
    secret=None
    state={"before":prev,"old":old,"phase":"secrets_saved",
           "previous_env":vars(rev),"previous_sa":sa}
    save(state)
    print("Deploying EXISTING image as a configuration-only revision (0% traffic)")
    cloud("run","deploy",SERVICE,"--region="+REGION,"--image="+image,
          "--no-traffic","--tag="+TAG,
          "--update-secrets="+",".join(k+"="+v+":latest" for k,v in SECRETS.items()),
          "--update-env-vars=SMTP_HOST=smtp-relay.brevo.com,SMTP_PORT=587",
          "--quiet")
    updated=service()
    if shares(updated)!=prev:raise RuntimeError("Production traffic changed unexpectedly")
    matches=[r for r in updated["status"]["traffic"] if r.get("tag")==TAG]
    if len(matches)!=1:raise RuntimeError("Brevo tagged revision not found")
    candidate=matches[0]["revisionName"]
    url=matches[0]["url"].rstrip("/")
    new=revision(candidate)
    newimage=new.get("spec",{}).get("containers",[{}])[0].get("image")
    if newimage!=image:raise RuntimeError("Container image unexpectedly changed")
    if new.get("spec",{}).get("serviceAccountName")!=sa:
        raise RuntimeError("Runtime service account changed")
    expected={**vars(rev),
              "SMTP_HOST":{"value":"smtp-relay.brevo.com"},
              "SMTP_PORT":{"value":"587"}}
    actual=vars(new)
    for key,value in expected.items():
        if key not in SECRETS and actual.get(key)!=value:
            raise RuntimeError("Unrelated environment variable changed: "+key)
    for k in SECRETS:
        if not actual.get(k,{}).get("valueFrom",{}).get("secretKeyRef"):
            raise RuntimeError("SMTP secret is not mounted: "+k)
    check(url,candidate)
    state.update({"phase":"staged","revision":candidate,"url":url})
    save(state)
    print("\nSTAGED SMTP (NO rebuild):",url+"/crm/")
    print("OLD PRODUCTION: unchanged",prev)
    print("TEST NOW: click 'Забыли пароль?' on preview and request an email to a registered CRM user.")
    print("Only after RECEIVING the letter: python3 scripts/brevo-no-build.py publish")

def publish():
    state=read_state()
    if state.get("phase")!="staged":raise RuntimeError("SMTP preview is not fully validated")
    current=service()
    if shares(current)!=state["before"]:
        raise RuntimeError("Active production revision changed. Refusing to overwrite it.")
    matched=[x for x in current["status"]["traffic"] if x.get("tag")==TAG]
    if len(matched)!=1 or matched[0].get("revisionName")!=state["revision"]:
        raise RuntimeError("SMTP preview tag moved since stage")
    check(state["url"],state["revision"])
    answer=input("Was a REAL CRM recovery email received? Type RECEIVED: ").strip()
    if answer!="RECEIVED":raise RuntimeError("Email receipt not confirmed")
    rev=state["revision"]
    cloud("run","services","update-traffic",SERVICE,"--region="+REGION,
          "--to-revisions="+rev+"=100")
    try:
        if shares(service())!={rev:100}:raise RuntimeError("100% not confirmed")
        check(DOMAIN,rev)
    except Exception:
        print("Post-promote verification failed: restoring old Cloud Run traffic.")
        cloud("run","services","update-traffic",SERVICE,"--region="+REGION,
              "--to-revisions="+",".join(k+"="+str(v) for k,v in state["before"].items()))
        raise
    state["phase"]="published";save(state)
    print("LIVE with working Brevo SMTP:",DOMAIN+"/crm/")
    print("Production revision:",rev)
    print("Reused the EXACT SAME container image. No code rebuild.")
    print("Bootstrap remains unchanged; if 1, site /lead is still disabled.")

if __name__=="__main__":
    try:
        if len(sys.argv)!=2 or sys.argv[1] not in ("stage","publish"):
            raise RuntimeError("Usage: python3 scripts/brevo-no-build.py stage|publish")
        {"stage":stage,"publish":publish}[sys.argv[1]]()
    except Exception as error:
        print("BREVO STAGE/PUBLISH STOPPED:",str(error),file=sys.stderr)
        sys.exit(1)
