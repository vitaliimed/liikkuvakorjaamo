#!/usr/bin/env python3
"""Read-only Liikkuva production gate.

Does not read or display secret VALUES, alter traffic, send messages,
change Firebase rewrites, or create data. Call before any rollout.
"""
import json
import subprocess
import sys
from urllib.parse import urlparse

PROJECT = "liikkuva-korjaamo"
REGION = "europe-north1"
SERVICE = "liikkuva-preview"
DATABASE = "liikkuvakorjaamo"
REQUIRED = (
    "SITE_URL", "FIRESTORE_PROJECT_ID", "FIRESTORE_LEADS_COLLECTION",
    "FIRESTORE_REVIEWS_COLLECTION", "SITE_SETTINGS_COLLECTION",
    "CLOUD_TASKS_PROJECT_ID", "CLOUD_TASKS_LOCATION", "CLOUD_TASKS_QUEUE",
    "INTERNAL_TASK_TOKEN", "REVIEW_TOKEN_SECRET", "INTERNAL_PROCESS_LEAD_URL",
    "GCS_BUCKET_NAME", "TELEGRAM_BOT_TOKEN", "TELEGRAM_CHAT_ID",
    "GOOGLE_APPS_SCRIPT_URL", "GOOGLE_APPS_SCRIPT_TOKEN",
)
EMAIL = ("SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "MAIL_FROM")
SECRET = {"INTERNAL_TASK_TOKEN", "REVIEW_TOKEN_SECRET",
          "TELEGRAM_BOT_TOKEN", "TELEGRAM_CHAT_ID",
          "GOOGLE_APPS_SCRIPT_TOKEN", "SMTP_PASSWORD", "SMTP_USERNAME"}


def gcloud(*args):
    p = subprocess.run(["gcloud", *args, f"--project={PROJECT}"],
                       capture_output=True, text=True, check=False)
    if p.returncode:
        raise RuntimeError(
            "gcloud " + " ".join(args[:3]) + " failed (no credentials shown): " +
            p.stderr[-400:])
    return p.stdout


def configured(entry):
    if not isinstance(entry, dict):
        return False
    return bool(entry.get("value") or
                entry.get("valueFrom", {}).get("secretKeyRef"))


def getval(env, name):
    return env.get(name, {}).get("value", "")


def main():
    errors = []
    warnings = []
    data = json.loads(gcloud("run", "services", "describe", SERVICE,
                             f"--region={REGION}", "--format=json"))
    traffic = data.get("status", {}).get("traffic", [])
    prod = [(r.get("revisionName"), r.get("percent")) for r in traffic
            if r.get("percent", 0) > 0]
    print("===== CURRENT TRAFFIC =====")
    print("Live revisions:", prod)
    template = data.get("spec", {}).get("template", {}).get("spec", {})
    env = {}
    for container in template.get("containers", []):
        env.update({e.get("name", ""): e for e in container.get("env", [])})
    print("===== ENVIRONMENT READINESS (NAMES ONLY) =====")
    print("LIIKKUVA_BOOTSTRAP:", getval(env, "LIIKKUVA_BOOTSTRAP") or "not set")
    for name in REQUIRED:
        state = "OK" if configured(env.get(name)) else "MISSING"
        print(name + ":", state)
        if state == "MISSING":
            errors.append("missing " + name)
    for name in EMAIL:
        state = "OK" if configured(env.get(name)) else "MISSING"
        print(name + ":", state)
        if state == "MISSING":
            errors.append("SMTP: " + name)
    if getval(env, "LIIKKUVA_BOOTSTRAP") == "1":
        errors.append("bootstrap currently blocks POST /lead on this template")
    if getval(env, "SITE_URL") != "https://liikkuvakorjaamo.fi":
        errors.append("SITE_URL must be https://liikkuvakorjaamo.fi")
    if getval(env, "FIRESTORE_PROJECT_ID") != PROJECT:
        errors.append("FIRESTORE_PROJECT_ID is not the Liikkuva project")
    for name in ("FIRESTORE_LEADS_COLLECTION", "FIRESTORE_REVIEWS_COLLECTION",
                 "SITE_SETTINGS_COLLECTION"):
        value = getval(env, name)
        if value and (not value.startswith("liikkuva_") or "/" in value):
            errors.append(name + " must use the liikkuva_ prefix")
    queue = getval(env, "CLOUD_TASKS_QUEUE")
    if queue and queue != "liikkuva-lead-processing":
        errors.append("CLOUD_TASKS_QUEUE has unexpected name")
    worker = getval(env, "INTERNAL_PROCESS_LEAD_URL")
    if worker:
        u = urlparse(worker)
        allowed = {
            "liikkuva-preview-451713593869.europe-north1.run.app",
            "liikkuva-preview-knrjxcx2bq-lz.a.run.app",
        }
        if u.scheme != "https" or u.hostname not in allowed or (
            u.path != "/internal/process-lead") or u.query or u.fragment:
            errors.append("worker URL should target the UNTAGGED Liikkuva service")
    if queue == "liikkuva-lead-processing":
        try:
            state = json.loads(gcloud("tasks", "queues", "describe", queue,
                                      f"--location={REGION}", "--format=json"))
            print("Cloud Tasks queue state:", state.get("state", "UNKNOWN"))
            if state.get("state") not in ("RUNNING",):
                errors.append("Cloud Tasks queue is not RUNNING")
        except Exception as exc:
            errors.append("cannot inspect Cloud Tasks queue: " + str(exc)[:90])
    bucket = getval(env, "GCS_BUCKET_NAME")
    if bucket:
        try:
            gcloud("storage", "buckets", "describe", "gs://" + bucket, "--format=json")
            print("Photo bucket: exists")
        except Exception as exc:
            errors.append("photo bucket unavailable: " + str(exc)[:90])
    # These read-only checks verify resource existence. Effective IAM may include
    # inherited or conditional bindings; a real candidate test remains mandatory.
    try:
        db = json.loads(gcloud("firestore", "databases", "describe", f"--database={DATABASE}", "--format=json"))
        print("Firestore database:", db.get("name", "UNKNOWN"))
        if db.get("type") != "FIRESTORE_NATIVE":
            errors.append("Firestore must use native mode")
    except Exception:
        errors.append("cannot inspect named Firestore database")
    account = template.get("serviceAccountName", "")
    print("Runtime service account:", account or "DEFAULT (must be resolved)")
    if not account:
        errors.append("runtime service account not explicitly identified")
    else:
        try:
            policy = json.loads(gcloud("projects", "get-iam-policy", PROJECT, "--format=json"))
            roles = {b.get("role") for b in policy.get("bindings", [])
                     if "serviceAccount:" + account in b.get("members", []) and not b.get("condition")}
            for role in ("roles/datastore.user", "roles/cloudtasks.enqueuer"):
                if role not in roles:
                    errors.append("runtime IAM role not proven by project bindings: " + role)
            print("Runtime project IAM:", sorted(roles))
        except Exception:
            errors.append("cannot inspect runtime project IAM")
    if not configured(env.get("CRM_OWNER_EMAIL")):
        errors.append("CRM_OWNER_EMAIL missing")
    if not configured(env.get("CRM_PHOTO_BUCKET")):
        errors.append("CRM_PHOTO_BUCKET missing")
    print("NOTE: service template is a candidate configuration, not proof of every serving revision.")
    print("NOTE: delivery, secret access, bucket object access, OIDC invocation and Firebase routing require live tests.")
    print("===== RESULT =====")
    for e in warnings:
        print("WARNING:", e)
    for e in errors:
        print("BLOCKER:", e)
    if errors:
        print("BLOCKED: do not switch production traffic or set bootstrap=0 blindly.")
        return 2
    print("CONFIG CHECK PASSED — still run Go tests, tagged revision smoke tests,")
    print("real test lead and Firebase Hosting rewrite verification before rollout.")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Exception as exc:
        print("AUDIT ERROR:", str(exc)[:240], file=sys.stderr)
        sys.exit(2)
