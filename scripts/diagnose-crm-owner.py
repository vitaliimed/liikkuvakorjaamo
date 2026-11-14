#!/usr/bin/env python3
"""Read-only CRM owner-account check (never displays stored password hashes).
Requires authenticated gcloud. Does not send emails, change Firestore or
modify Cloud Run; it verifies whether the configured CRM owner login exists.
"""
import hashlib
import json
import subprocess
import sys
import urllib.error
import urllib.request

PROJECT="liikkuva-korjaamo"
REGION="europe-north1"
SERVICE="liikkuva-preview"
DATABASE="liikkuvakorjaamo"

def cli(*args):
    p=subprocess.run(["gcloud",*args,"--project="+PROJECT],
                     capture_output=True,text=True,check=False)
    if p.returncode:
        raise RuntimeError("Cloud permission/configuration error: "+p.stderr[-220:])
    return p.stdout.strip()

def main():
    revision=json.loads(cli("run","services","describe",SERVICE,
                            "--region="+REGION,"--format=json"))
    env={}
    for container in revision.get("spec",{}).get("template",{}).get("spec",{}).get("containers",[]):
        for item in container.get("env",[]):
            env[item.get("name")]=item
    email=env.get("CRM_OWNER_EMAIL",{}).get("value","").strip().lower()
    if not email or "@" not in email:
        print("BLOCKER: CRM_OWNER_EMAIL is missing or invalid (value intentionally hidden)")
        return 2
    if not env.get("SMTP_HOST") or not env.get("SMTP_USERNAME") or not env.get("SMTP_PASSWORD"):
        print("BLOCKER: at least one shared SMTP setting is missing")
        return 2
    print("CONFIG: CRM owner login and SMTP environment names configured")
    account_hash=hashlib.sha256(email.encode()).hexdigest()
    token=cli("auth","print-access-token")
    url=(f"https://firestore.googleapis.com/v1/projects/{PROJECT}/databases/{DATABASE}"
         f"/documents/crm_users/{account_hash}")
    req=urllib.request.Request(url,headers={"Authorization":"Bearer "+token})
    try:
        with urllib.request.urlopen(req,timeout=30) as response:
            data=json.load(response)
    except urllib.error.HTTPError as error:
        if error.code==404:
            print("BLOCKER: configured CRM_OWNER_EMAIL does not exist in crm_users")
            print("No automatic account creation or password changes attempted")
            return 2
        print("BLOCKER: Firestore read denied HTTP",error.code)
        return 2
    fields=data.get("fields",{})
    active=fields.get("Active",fields.get("active",{})).get("booleanValue",False)
    role=fields.get("Role",fields.get("role",{})).get("stringValue","")
    password_field=fields.get("Hash",fields.get("hash",{})).get("stringValue","")
    print("CRM owner: record exists")
    print("CRM owner active:",bool(active))
    print("CRM role:",role or "missing")
    print("CRM password hash present:",bool(password_field))
    if not active or role!="owner" or not password_field:
        print("BLOCKER: password recovery will not send to this account until owner record is active")
        return 2
    print("PASS: CRM registered owner is eligible for password reset")
    print("NEXT: inspect Brevo SMTP transactional logs; account presence does NOT prove email delivery")
    return 0

if __name__=="__main__":
    try:
        sys.exit(main())
    except (OSError,ValueError,RuntimeError,urllib.error.URLError) as error:
        print("DIAGNOSTIC STOPPED:",str(error)[:260],file=sys.stderr)
        sys.exit(2)
