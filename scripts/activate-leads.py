#!/usr/bin/env python3
"""Configure Liikkuva's real lead pipeline and promote an enabled revision.

Run only from authenticated Cloud Shell, AFTER scripts/publish-site-crm.py.
Requires the REAL deployed Google Apps Script web-app URL and its matching
token. Never creates placeholder integrations, overwrites existing secrets,
drops collections, or switches traffic before launch checks succeed.
"""
from __future__ import annotations
import getpass
import json
import os
import pathlib
import secrets
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[1]
PROJECT = "liikkuva-korjaamo"
REGION = "europe-north1"
# Cloud Tasks is not available in europe-north1; Cloud Run stays in Finland.
TASKS_REGION = "europe-west3"  # Frankfurt, Germany (EU)

SERVICE = "liikkuva-preview"
DATABASE = "liikkuvakorjaamo"
DOMAIN = "https://liikkuvakorjaamo.fi"
TAG = "intake-release"
VERSION = "2026.11.12-production-candidate"
BACKUPS = pathlib.Path.home() / ".liikkuva-releases"


def command(args: list[str], *, capture: bool = True, stdin: str | None = None) -> str:
    p = subprocess.run(args, text=True, input=stdin, stdout=subprocess.PIPE if capture else None,
                       stderr=subprocess.PIPE if capture else None)
    if p.returncode:
        raise RuntimeError("Command failed: " + " ".join(args[:4]) + " / " +
                           (p.stderr or "")[-350:])
    return p.stdout or ""


def cloud(*parts: str) -> str:
    return command(["gcloud", *parts, "--project=" + PROJECT])


def service() -> dict:
    return json.loads(cloud("run", "services", "describe", SERVICE,
                            "--region=" + REGION, "--format=json"))


def traffic(d: dict) -> dict[str, int]:
    result: dict[str, int] = {}
    for t in d.get("status", {}).get("traffic", []):
        n = int(t.get("percent") or 0)
        if n:
            name = t.get("revisionName")
            if not name:
                raise RuntimeError("Unresolved serving revision")
            result[name] = result.get(name, 0) + n
    if sum(result.values()) != 100:
        raise RuntimeError("Expected total 100% production traffic")
    return result


def environment(d: dict) -> dict:
    containers = d.get("spec", {}).get("template", {}).get("spec", {}).get("containers", [])
    return {e["name"]: e for c in containers for e in c.get("env", [])}


def configured(e: dict | None) -> bool:
    if not e:
        return False
    return bool(e.get("value") or e.get("valueFrom", {}).get("secretKeyRef"))


def ensure_secret(name: str, *, prompt: bool = False) -> str:
    ref = "liikkuva-" + name.lower().replace("_", "-")
    inspect = subprocess.run(
        ["gcloud", "secrets", "describe", ref, "--project=" + PROJECT],
        stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if inspect.returncode:
        cloud("secrets", "create", ref, "--replication-policy=automatic")
    versions = json.loads(cloud("secrets", "versions", "list", ref, "--format=json"))
    has_enabled = any(v.get("state") == "ENABLED" for v in versions)
    if not has_enabled:
        if prompt:
            value = getpass.getpass(f"Enter the EXISTING matching {name} (hidden): ").strip()
            if len(value) < 20:
                raise RuntimeError(f"{name} must match the deployed Apps Script, minimum 20 characters")
        else:
            value = secrets.token_urlsafe(48)
        command(["gcloud", "secrets", "versions", "add", ref, "--data-file=-",
                 "--project=" + PROJECT], stdin=value, capture=True)
    return ref + ":latest"


def firestore_doc(path: str, bearer: str, method: str = "GET", body: dict | None = None):
    base = (f"https://firestore.googleapis.com/v1/projects/{PROJECT}"
            f"/databases/{DATABASE}/documents/")
    req = urllib.request.Request(
        base + path,
        method=method,
        headers={"Authorization": "Bearer " + bearer, "Content-Type": "application/json"},
        data=json.dumps(body).encode("utf-8") if body is not None else None)
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            return json.load(response)
    except urllib.error.HTTPError as error:
        if error.code == 404 and method == "GET":
            return None
        raise


def migrate_site_settings() -> None:
    token = command(["gcloud", "auth", "print-access-token"], capture=True).strip()
    for old_path, new_path in (
        ("site_settings/landing", "liikkuva_site_settings/landing"),
        ("site_settings_private/admin_access",
         "liikkuva_site_settings_private/admin_access"),
    ):
        source_doc = firestore_doc(old_path, token)
        dest_doc = firestore_doc(new_path, token)
        if source_doc and not dest_doc:
            # Keep admin credentials and revision numbers; never overwrite or delete.
            collection, doc_id = new_path.rsplit("/", 1)
            firestore_doc(collection + "?documentId=" + doc_id, token,
                          method="POST", body={"fields": source_doc.get("fields", {})})
            print("Copied legacy Firestore document:", old_path, "->", new_path)
        elif dest_doc:
            print("Destination already exists; preserved:", new_path)
        elif old_path.endswith("/admin_access"):
            print("WARNING: no existing admin_access document found; verify admin login")
        else:
            print("No legacy site settings to migrate; defaults will apply")


def get_status(url: str, method: str = "GET", data: bytes | None = None) -> tuple[int, bytes]:
    req = urllib.request.Request(url, method=method, data=data,
                                 headers={"Cache-Control": "no-cache",
                                          "Content-Type": "application/x-www-form-urlencoded"})
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return r.status, r.read(128 * 1024)
    except urllib.error.HTTPError as e:
        return e.code, e.read(128 * 1024)


def check(url: str, revision: str) -> None:
    status, body = get_status(url + "/api/health")
    if status != 200:
        raise RuntimeError("Health failed: HTTP " + str(status))
    h = json.loads(body)
    if h.get("revision") != revision or h.get("version") != VERSION:
        raise RuntimeError("Unexpected deployed revision or app version: " + str(h))
    if get_status(url + "/crm/")[0] != 200:
        raise RuntimeError("CRM page unavailable")
    # A malformed, non-customer request must fail with validation, not bootstrap.
    status, body = get_status(url + "/lead", method="POST", data=b"language=fi")
    if status != 400:
        raise RuntimeError(f"Lead intake unavailable: expected HTTP 400 invalid form, got {status}")
    print("CHECK: health, CRM, unpaused lead validation passed at", url)


def main() -> None:
    snapshot = service()
    before = traffic(snapshot)
    if len(before) != 1:
        raise RuntimeError("Expected one active production revision, cannot safely promote")
    current_env = environment(snapshot)
    bootstrap = current_env.get("LIIKKUVA_BOOTSTRAP", {}).get("value")
    if bootstrap == "0" and "--stage-only" in sys.argv[1:]:
        print("Intake is already configured as Bootstrap=0 in the current service template.")
        print("No additional revision created; validate using prod_preflight.py and the UX preview.")
        return
    if bootstrap != "1":
        raise RuntimeError("Intake activation expects Bootstrap=1; got " + str(bootstrap))
    if current_env.get("SITE_URL", {}).get("value") != DOMAIN:
        raise RuntimeError("SITE_URL mismatch")
    if current_env.get("FIRESTORE_PROJECT_ID", {}).get("value") != PROJECT:
        raise RuntimeError("Firestore project mismatch")
    if not configured(current_env.get("TELEGRAM_BOT_TOKEN")) or not configured(current_env.get("TELEGRAM_CHAT_ID")):
        raise RuntimeError("Telegram must be configured before enabling leads")
    for key in ("SMTP_HOST", "SMTP_USERNAME", "SMTP_PASSWORD", "MAIL_FROM"):
        if not configured(current_env.get(key)):
            raise RuntimeError("Missing mail setting: " + key)
    bucket = current_env.get("GCS_BUCKET_NAME", {}).get("value", "")
    if not bucket.endswith("-liikkuva-files"):
        raise RuntimeError("Photo bucket must have the -liikkuva-files suffix")
    run_url = "https://liikkuva-preview-451713593869.europe-north1.run.app"
    worker = run_url + "/internal/process-lead"

    url = current_env.get("GOOGLE_APPS_SCRIPT_URL", {}).get("value", "").strip()
    if not url:
        print("Google Apps Script is NOT deployed in the Cloud Run configuration.")
        url = input("Paste the existing Google Apps Script Web App /exec URL: ").strip()
    parsed = urllib.parse.urlsplit(url)
    if (parsed.scheme != "https" or parsed.hostname != "script.google.com" or
        not parsed.path.startswith("/macros/s/") or not parsed.path.endswith("/exec")):
        raise RuntimeError("Real Apps Script /exec URL required; cannot generate it from GitHub code")
    # A valid-looking URL can still point to an undeployed/unauthorized Apps Script.
    # doGet() is read-only; require the actual Liikkuva service response.
    try:
        req = urllib.request.Request(url, headers={"Accept": "application/json"})
        with urllib.request.urlopen(req, timeout=25) as response:
            content = response.read(8192)
        handshake = json.loads(content)
        if handshake.get("ok") is not True or handshake.get("site") != "liikkuvakorjaamo.fi":
            raise RuntimeError("Apps Script /exec returned the wrong site or is not initialized")
    except (urllib.error.URLError, ValueError) as exc:
        raise RuntimeError("Apps Script /exec is not ready (GET must return Liikkuva JSON): " +
                           type(exc).__name__) from exc
    print("PASS: deployed Apps Script /exec identifies Liikkuva (GET only)")

    BACKUPS.mkdir(parents=True, exist_ok=True, mode=0o700)
    os.chmod(BACKUPS, 0o700)
    backup = BACKUPS / (f"pre-intake-{int(time.time())}.json")
    fd = os.open(backup, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o600)
    with os.fdopen(fd, "w") as f:
        json.dump(snapshot, f, indent=2)
    print("Saved reversible Cloud Run snapshot:", backup)

    service_account = snapshot.get("spec", {}).get("template", {}).get("spec", {}).get("serviceAccountName")
    if not service_account:
        number = cloud("projects", "describe", PROJECT, "--format=value(projectNumber)").strip()
        service_account = number + "-compute@developer.gserviceaccount.com"
    member = "serviceAccount:" + service_account

    print("Checking named Firestore database and migrating site settings without deleting any data")
    db = json.loads(cloud("firestore", "databases", "describe",
                          "--database=" + DATABASE, "--format=json"))
    if db.get("type") != "FIRESTORE_NATIVE":
        raise RuntimeError("Firestore native database required")
    migrate_site_settings()

    print("Ensuring Cloud Tasks queue in", TASKS_REGION, "(Cloud Run remains in", REGION + ")")
    queue = "liikkuva-lead-processing"
    p = subprocess.run(["gcloud", "tasks", "queues", "describe", queue,
                        "--location=" + TASKS_REGION, "--project=" + PROJECT],
                       stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    if p.returncode:
        cloud("tasks", "queues", "create", queue, "--location=" + TASKS_REGION,
              "--max-dispatches-per-second=2", "--max-concurrent-dispatches=2")
    q = json.loads(cloud("tasks", "queues", "describe", queue,
                         "--location=" + TASKS_REGION, "--format=json"))
    if q.get("state") != "RUNNING":
        raise RuntimeError("Queue not RUNNING; inspect before activation")

    for role in ("roles/datastore.user", "roles/cloudtasks.enqueuer"):
        cloud("projects", "add-iam-policy-binding", PROJECT, "--member=" + member,
              "--role=" + role, "--condition=None", "--format=none")
    print("Granted Firestore/Cloud Tasks runtime IAM access to existing service account")

    secrets_to_set = {}
    for name in ("INTERNAL_TASK_TOKEN", "REVIEW_TOKEN_SECRET"):
        if not configured(current_env.get(name)):
            secrets_to_set[name] = ensure_secret(name)
    if not configured(current_env.get("GOOGLE_APPS_SCRIPT_TOKEN")):
        secrets_to_set["GOOGLE_APPS_SCRIPT_TOKEN"] = ensure_secret("GOOGLE_APPS_SCRIPT_TOKEN", prompt=True)
    for ref in secrets_to_set.values():
        secret_name = ref.split(":")[0]
        cloud("secrets", "add-iam-policy-binding", secret_name,
              "--member=" + member, "--role=roles/secretmanager.secretAccessor")
    print("Secrets granted by reference; no secret values displayed")

    env = {
        "LIIKKUVA_BOOTSTRAP": "0",
        "FIRESTORE_LEADS_COLLECTION": "liikkuva_leads",
        "FIRESTORE_REVIEWS_COLLECTION": "liikkuva_reviews",
        "SITE_SETTINGS_COLLECTION": "liikkuva_site_settings",
        "CLOUD_TASKS_PROJECT_ID": PROJECT,
        "CLOUD_TASKS_LOCATION": TASKS_REGION,
        "CLOUD_TASKS_QUEUE": queue,
        "INTERNAL_PROCESS_LEAD_URL": worker,
        "GOOGLE_APPS_SCRIPT_URL": url,
        "CRM_OWNER_EMAIL": "medynikv@gmail.com",
        "CRM_PHOTO_BUCKET": "liikkuva-korjaamo-crm-private",
        "CRM_PUBLIC_URL": DOMAIN,
        "CRM_MAIL_SEND_ENABLED": "1",  # owner-only action; SMTP required above
    }
    print("Staging enabled intake at 0% traffic, production stays on Bootstrap=1")
    args = ["run", "deploy", SERVICE, "--region=" + REGION,
            "--source=" + str(ROOT / "source"), "--no-traffic", "--tag=" + TAG,
            "--update-env-vars=" + ",".join(k + "=" + v for k, v in env.items()),
            "--quiet"]
    if secrets_to_set:
        args.append("--update-secrets=" +
                    ",".join(k + "=" + v for k, v in secrets_to_set.items()))
    cloud(*args)
    updated = service()
    if traffic(updated) != before:
        raise RuntimeError("Traffic changed unexpectedly during staged intake deployment")
    routes = [r for r in updated["status"]["traffic"] if r.get("tag") == TAG]
    if len(routes) != 1:
        raise RuntimeError("Tagged intake revision not found")
    revision = routes[0]["revisionName"]
    staged_url = routes[0]["url"].rstrip("/")
    check(staged_url, revision)

    if "--stage-only" in sys.argv[1:]:
        print("INTAKE PREVIEW READY, PRODUCTION TRAFFIC UNCHANGED:", staged_url)
        print("Candidate:", revision)
        print("Next: run python3 prod_preflight.py, then")
        print("python3 scripts/release-ux-20261112.py publish")
        return

    print("Promoting enabled intake to 100% and validating the public domain")
    cloud("run", "services", "update-traffic", SERVICE, "--region=" + REGION,
          "--to-revisions=" + revision + "=100")
    try:
        if traffic(service()) != {revision: 100}:
            raise RuntimeError("Not 100% on the selected intake revision")
        check(DOMAIN, revision)
    except Exception:
        print("Health validation failed; restoring the previous revision.")
        restore = ",".join(f"{r}={p}" for r, p in before.items())
        cloud("run", "services", "update-traffic", SERVICE, "--region=" + REGION,
              "--to-revisions=" + restore)
        raise
    print("SUCCESS: website + CRM + POST /lead live at", DOMAIN)
    print("NEXT: submit one of your OWN test leads and inspect Firestore, Telegram, Apps Script and SMTP statuses")
    print("Rollback baseline:", backup)
    print("Note: CONFIGURED integrations are not proof of real message delivery")


if __name__ == "__main__":
    try:
        if any(arg != "--stage-only" for arg in sys.argv[1:]) or sys.argv[1:].count("--stage-only") > 1:
            raise RuntimeError("Usage: activate-leads.py [--stage-only]")
        main()
    except (RuntimeError, ValueError, urllib.error.URLError, OSError) as exc:
        print("ACTIVATION STOPPED:", str(exc), file=sys.stderr)
        sys.exit(1)
