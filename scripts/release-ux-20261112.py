#!/usr/bin/env python3
"""Safe staged release of Liikkuva UX. Never switches traffic during 'stage'.

Run from an authenticated Google Cloud Shell. No secrets are printed.
Usage: python3 scripts/release-ux-20261112.py stage|status|publish|rollback
"""
from __future__ import annotations

import json
import os
from pathlib import Path
import subprocess
import sys
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
PROJECT = "liikkuva-korjaamo"
REGION = "europe-north1"
SERVICE = "liikkuva-preview"
BRANCH = "fix/hero-form-theme-autofill-20261112"
EXPECTED_VERSION = "2026.11.12-production-candidate"
DOMAIN = "https://liikkuvakorjaamo.fi"
TAG = "ux-nov12-preview"
STATE_DIR = Path.home() / ".liikkuva-releases"
STATE_FILE = STATE_DIR / "ux-nov12-release.json"


def command(args, cwd=ROOT):
    result = subprocess.run(list(args), cwd=cwd, text=True,
                            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if result.returncode:
        # Do not leak environment settings from a detailed gcloud error.
        raise RuntimeError("Command failed: " + " ".join(map(str, args[:5])) +
                           " (exit " + str(result.returncode) + ")\n" +
                           result.stderr[-900:])
    return result.stdout


def cloud(*args):
    return command(["gcloud", *args, "--project=" + PROJECT])


def service():
    return json.loads(cloud("run", "services", "describe", SERVICE,
                            "--region=" + REGION, "--format=json"))


def allocation(data):
    shares = {}
    for target in data.get("status", {}).get("traffic", []):
        percent = int(target.get("percent") or 0)
        if not percent:
            continue
        revision = target.get("revisionName")
        if not revision:
            raise RuntimeError("Unresolved revision in active traffic")
        shares[revision] = shares.get(revision, 0) + percent
    if sum(shares.values()) != 100:
        raise RuntimeError("Expected resolved 100% traffic, got " + str(shares))
    return shares


def env_value(data, key):
    containers = data.get("spec", {}).get("template", {}).get("spec", {}).get("containers", [])
    for container in containers:
        for item in container.get("env", []):
            if item.get("name") == key:
                return item.get("value", "<secret reference>")
    return "<not set>"


def save_json(path, data):
    STATE_DIR.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(STATE_DIR, 0o700)
    temp = path.with_suffix(".tmp")
    fd = os.open(temp, os.O_CREAT | os.O_TRUNC | os.O_WRONLY, 0o600)
    with os.fdopen(fd, "w", encoding="utf-8") as stream:
        json.dump(data, stream, ensure_ascii=False, indent=2)
    os.replace(temp, path)


def load_state():
    if not STATE_FILE.exists():
        raise RuntimeError("No staged release found. Run 'stage' first.")
    return json.loads(STATE_FILE.read_text(encoding="utf-8"))


def request(url, *, method="GET", data=None):
    try:
        req = urllib.request.Request(
            url, data=data, method=method,
            headers={"Cache-Control": "no-cache", "Pragma": "no-cache",
                     "Content-Type": "application/x-www-form-urlencoded"})
        with urllib.request.urlopen(req, timeout=35) as resp:
            return resp.status, resp.read(768 * 1024)
    except urllib.error.HTTPError as exc:
        return exc.code, exc.read(32 * 1024)


def check(url, revision, bootstrap):
    url = url.rstrip("/")
    code, body = request(url + "/api/health?ux_release=" + str(int(time.time())))
    if code != 200:
        raise RuntimeError("Health failed: HTTP " + str(code) + " at " + url)
    health = json.loads(body)
    if (health.get("revision") != revision or
            health.get("version") != EXPECTED_VERSION or
            health.get("ok") is not True):
        raise RuntimeError("Wrong revision/version at " + url + " " +
                           str({k: health.get(k) for k in ("revision", "version", "ok")}))
    for path in ("/", "/en"):
        code, body = request(url + path + ("?ux_release=1" if path == "/" else "?ux_release=2"))
        if code != 200:
            raise RuntimeError(path + " returned HTTP " + str(code))
        html = body.decode("utf-8")
        required = ('class="hero-form" id="lead-form"', 'id="leadForm"',
                    'name="phone" required', 'name="car" required',
                    'name="email" required', 'name="address" required',
                    'data-theme-toggle', 'hero-form-ux.css',
                    'og:image', 'logo-white.png')
        if not all(s in html for s in required):
            raise RuntimeError("Missing hero-form/required-fields/OG markers at " + path)
        if html.index('class="hero-form" id="lead-form"') > html.index('class="hero-copy"'):
            raise RuntimeError("Form must appear before hero copy at " + path)
        if html.count('id="leadForm"') != 1:
            raise RuntimeError("Page must have exactly one form at " + path)
        print("PASS", path, "compact form and required fields")
    for path, marker in (
        ("/static/css/hero-form-ux.css", ".hero-form"),
        ("/static/js/theme.js", "data-theme-toggle"),
        ("/static/js/app.js", "liikkuva_lead_autofill_v2"),
        ("/static/img/logo-white.png", None),
        ("/static/img/favicon.svg", None),
        ("/crm/", None),
        ("/admin", None),
        ("/privacy?lang=fi", None),
    ):
        code, body = request(url + path)
        if code != 200:
            raise RuntimeError(path + " HTTP " + str(code))
        if marker and marker.encode("utf-8") not in body:
            raise RuntimeError("Expected marker missing in " + path)
        print("PASS", path)
    if bootstrap == "0":
        code, _ = request(url + "/lead", method="POST", data=b"language=fi")
        if code != 400:
            raise RuntimeError("Invalid lead did not get HTTP 400 (got " + str(code) + ")")
        print("PASS /lead rejects incomplete request with HTTP 400")
    else:
        print("WARNING: LIIKKUVA_BOOTSTRAP=" + bootstrap +
              ". Real leads are NOT verified; publication will be blocked.")


def github_state():
    branch = command(["git", "branch", "--show-current"]).strip()
    if branch != BRANCH:
        raise RuntimeError("Wrong git branch " + branch + ", expected " + BRANCH)
    if command(["git", "status", "--porcelain"]).strip():
        raise RuntimeError("Uncommitted changes: use a clean checkout")
    return command(["git", "rev-parse", "HEAD"]).strip()


def tag_route(data):
    routes = [r for r in data.get("status", {}).get("traffic", []) if r.get("tag") == TAG]
    if len(routes) != 1:
        raise RuntimeError("Preview traffic tag not found/ambiguous: " + TAG)
    route = routes[0]
    if not route.get("revisionName") or not route.get("url"):
        raise RuntimeError("Tag has no resolved revision or URL")
    return route["revisionName"], route["url"].rstrip("/")


def firebase_pin():
    """Read only. Return Hosting Cloud Run rewrite's tag, or None."""
    try:
        token = cloud("auth", "print-access-token").strip()
        prefix = "https://firebasehosting.googleapis.com/v1beta1/"
        def get(path):
            req = urllib.request.Request(prefix + path, headers={
                "Authorization": "Bearer " + token, "Accept": "application/json"})
            with urllib.request.urlopen(req, timeout=20) as resp:
                return json.load(resp)
        live = get("sites/liikkuva-liikkuva-korjaamo/channels/live")
        version = live.get("release", {}).get("version") or {}
        if isinstance(version, str):
            version = {"name": version}
        if version.get("name") and "config" not in version:
            version = get(version["name"].lstrip("/"))
        rewrites = version.get("config", {}).get("rewrites", [])
        runs = [r["run"] for r in rewrites if "run" in r]
        for run in runs:
            if run.get("serviceId") != SERVICE:
                raise RuntimeError("Firebase Hosting points to another Cloud Run service")
        return runs[0].get("tag") if runs else None
    except urllib.error.HTTPError as exc:
        if exc.code in (403, 404):
            print("WARNING: Firebase Hosting config not readable (HTTP " + str(exc.code) +
                  "). Public-domain revision check remains mandatory.")
            return None
        raise


def stage():
    sha = github_state()
    print("SOURCE:", BRANCH, sha)
    print("RUNNING TESTS (no Cloud Run traffic changes)")
    command(["go", "test", "./..."], cwd=ROOT / "source")
    print("PASS Go tests")
    command(["go", "vet", "./..."], cwd=ROOT / "source")
    print("PASS go vet")
    for name in ("crm-ui.test.cjs", "theme.test.cjs", "events.test.cjs", "apps-script.test.cjs"):
        command(["node", "--test", str(ROOT / "tests" / name)])
        print("PASS Node", name)
    before_service = service()
    before = allocation(before_service)
    stamp = time.strftime("%Y%m%d-%H%M%S", time.gmtime())
    backup = STATE_DIR / ("cloudrun-before-ux-" + stamp + ".json")
    save_json(backup, before_service)
    state = {"sha": sha, "branch": BRANCH, "previous": before, "backup": str(backup),
             "phase": "before_deploy", "tag": TAG}
    save_json(STATE_FILE, state)
    print("BACKUP (private):", backup)
    print("PRODUCTION BEFORE:", before)
    print("Deploying no-traffic revision; existing CRM, secrets and env settings retained")
    cloud("run", "deploy", SERVICE,
          "--region=" + REGION, "--source=" + str(ROOT / "source"),
          "--no-traffic", "--tag=" + TAG, "--quiet")
    after_service = service()
    if allocation(after_service) != before:
        raise RuntimeError("Production traffic changed unexpectedly! Inspect service before proceeding")
    revision, preview = tag_route(after_service)
    bootstrap = env_value(after_service, "LIIKKUVA_BOOTSTRAP")
    state.update({"phase": "staged", "revision": revision, "preview": preview, "bootstrap": bootstrap})
    save_json(STATE_FILE, state)
    check(preview, revision, bootstrap)
    state["phase"] = "validated"
    save_json(STATE_FILE, state)
    print("\nPREVIEW VERIFIED:", preview)
    print("PREVIEW CRM:", preview + "/crm/")
    print("LIVE DOMAIN UNCHANGED:", DOMAIN)
    print("INTAKE:", "ACTIVE (invalid-input validation checked)" if bootstrap == "0"
          else "PAUSED. Do NOT publish before activating and validating integrations.")
    print("NEXT: inspect preview on desktop/mobile; then run 'publish' only when intake is ready.")


def publish():
    state = load_state()
    if state.get("phase") != "validated":
        raise RuntimeError("Not fully validated. Run stage and fix any failing check.")
    if github_state() != state.get("sha"):
        raise RuntimeError("Git checkout changed since staging")
    now = service()
    if allocation(now) != state["previous"]:
        raise RuntimeError("Production traffic changed since staging. Do not overwrite it.")
    bootstrap = env_value(now, "LIIKKUVA_BOOTSTRAP")
    if bootstrap != "0":
        raise RuntimeError("BLOCKED: LIIKKUVA_BOOTSTRAP=" + bootstrap +
                           "; run python3 scripts/activate-leads.py --stage-only first.")
    if state.get("bootstrap") == "0":
        revision, preview = tag_route(now)
        if revision != state["revision"]:
            raise RuntimeError("The tested UX revision changed")
    else:
        # The original UX preview inherited the paused setup. Activate intake in a
        # separate no-traffic revision; only that validated revision can go live.
        candidates = [r for r in now.get("status", {}).get("traffic", [])
                      if r.get("tag") == "intake-release"]
        if len(candidates) != 1 or not candidates[0].get("revisionName") or not candidates[0].get("url"):
            raise RuntimeError("No staged intake-release revision. Run activate-leads.py --stage-only")
        revision, preview = candidates[0]["revisionName"], candidates[0]["url"].rstrip("/")
        if revision == state["revision"]:
            raise RuntimeError("Intake candidate must be a separate enabled revision")
    print("CONFIGURATION PREFLIGHT:")
    command(["python3", str(ROOT / "prod_preflight.py")])
    check(preview, revision, bootstrap)
    tag = firebase_pin()
    if tag and not any(r.get("tag") == tag for r in now.get("status", {}).get("traffic", [])):
        raise RuntimeError("Firebase Hosting points to missing Cloud Run tag " + tag)
    old_tag_revision = next((r.get("revisionName") for r in now["status"]["traffic"]
                             if r.get("tag") == tag), None) if tag else None
    print("Promoting EXACT validated revision:", revision)
    try:
        cloud("run", "services", "update-traffic", SERVICE,
              "--region=" + REGION, "--to-revisions=" + revision + "=100")
        if tag:
            cloud("run", "services", "update-traffic", SERVICE, "--region=" + REGION,
                  "--update-tags=" + tag + "=" + revision)
        for attempt in range(12):
            try:
                if allocation(service()) != {revision: 100}:
                    raise RuntimeError("Unexpected production traffic allocation")
                check(DOMAIN, revision, bootstrap)
                break
            except (RuntimeError, urllib.error.URLError) as exc:
                if attempt == 11:
                    raise
                print("Domain routing not ready yet; verification retry", attempt + 1)
                time.sleep(5)
    except Exception:
        print("PROMOTION CHECK FAILED; restoring original traffic and hosting tag")
        restore_traffic(state["previous"])
        if tag and old_tag_revision:
            cloud("run", "services", "update-traffic", SERVICE, "--region=" + REGION,
                  "--update-tags=" + tag + "=" + old_tag_revision)
        raise
    state["phase"] = "published"
    state["firebase_pin"] = tag
    state["previous_pin_revision"] = old_tag_revision
    save_json(STATE_FILE, state)
    print("\nPUBLISHED:", DOMAIN, "->", revision)
    print("ROLLBACK: python3 scripts/release-ux-20261112.py rollback")
    print("IMPORTANT: send a REAL personal test lead and verify Firestore, Cloud Tasks, SMTP/Telegram/Apps Script delivery.")


def restore_traffic(previous):
    target = ",".join(name + "=" + str(value) for name, value in previous.items())
    cloud("run", "services", "update-traffic", SERVICE, "--region=" + REGION,
          "--to-revisions=" + target)


def rollback():
    state = load_state()
    if not state.get("previous"):
        raise RuntimeError("No original allocation backed up")
    current = allocation(service())
    if state.get("phase") != "published":
        raise RuntimeError("Release is not marked published; refusing to override current traffic")
    if current != {state["revision"]: 100}:
        raise RuntimeError("Traffic already changed; manual recovery needed with original snapshot")
    restore_traffic(state["previous"])
    if state.get("firebase_pin") and state.get("previous_pin_revision"):
        cloud("run", "services", "update-traffic", SERVICE, "--region=" + REGION,
              "--update-tags=" + state["firebase_pin"] + "=" + state["previous_pin_revision"])
    state["phase"] = "rolled_back"
    save_json(STATE_FILE, state)
    print("ROLLBACK DONE:", state["previous"])


def status():
    d = service()
    print("Cloud Run production:", allocation(d))
    print("Cloud Run bootstrap:", env_value(d, "LIIKKUVA_BOOTSTRAP"))
    if STATE_FILE.exists():
        s = load_state()
        print("Last UX staging:", {k: s.get(k) for k in
                                   ("phase", "revision", "preview", "backup")})
    else:
        print("No UX staging state found")


if __name__ == "__main__":
    try:
        if len(sys.argv) != 2 or sys.argv[1] not in ("stage", "status", "publish", "rollback"):
            raise RuntimeError("Usage: release-ux-20261112.py stage|status|publish|rollback")
        {"stage": stage, "status": status, "publish": publish,
         "rollback": rollback}[sys.argv[1]]()
    except Exception as exc:
        print("RELEASE STOPPED:", str(exc), file=sys.stderr)
        sys.exit(1)
