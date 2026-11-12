#!/usr/bin/env python3
"""Promote the integrated Liikkuva site + CRM while keeping lead intake paused.

This is an explicit, reversible website/CRM publication, NOT proof that lead
notifications or WhatsApp Business Coexistence are configured.  Cloud Run must
be accessed from the owner's authenticated Cloud Shell.
"""
from __future__ import annotations

import json
import os
import pathlib
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[1]
PROJECT = "liikkuva-korjaamo"
REGION = "europe-north1"
SERVICE = "liikkuva-preview"
BRANCH = "release/production-20261112"
DOMAIN = "https://liikkuvakorjaamo.fi"
EXPECTED_VERSION = "2026.11.12-production-candidate"
TAG = "site-crm-release"
BACKUPS = pathlib.Path.home() / ".liikkuva-releases"


def cmd(*args: str, cwd: pathlib.Path = ROOT, capture: bool = False) -> str:
    print("+", " ".join(str(x) for x in args if "SECRET" not in str(x)), flush=True)
    p = subprocess.run(args, cwd=cwd, text=True, check=True,
                       stdout=subprocess.PIPE if capture else None)
    return p.stdout or ""


def cloud(*args: str, capture: bool = False) -> str:
    return cmd("gcloud", *args, "--project=" + PROJECT, capture=capture)


def service() -> dict:
    return json.loads(cloud("run", "services", "describe", SERVICE,
                            "--region=" + REGION, "--format=json", capture=True))


def allocations(d: dict) -> dict[str, int]:
    result: dict[str, int] = {}
    for t in d.get("status", {}).get("traffic", []):
        n = int(t.get("percent") or 0)
        if n:
            rev = t.get("revisionName")
            if not rev:
                raise RuntimeError("Active revision missing from traffic data")
            result[rev] = result.get(rev, 0) + n
    if sum(result.values()) != 100:
        raise RuntimeError("Current traffic is not a resolved 100% allocation")
    return result


def health(url: str, expected_revision: str | None = None) -> dict:
    b = read_url(url + "/api/health")
    j = json.loads(b.decode("utf-8"))
    if j.get("ok") is not True or j.get("version") != EXPECTED_VERSION:
        raise RuntimeError("Wrong site version at " + url + ": " +
                           str({k: j.get(k) for k in ("ok", "version", "revision")}))
    if expected_revision and j.get("revision") != expected_revision:
        raise RuntimeError("Wrong serving revision: " + str(j.get("revision")))
    return j


def read_url(url: str) -> bytes:
    with urllib.request.urlopen(
        urllib.request.Request(url, headers={"Cache-Control": "no-cache"}),
        timeout=30
    ) as resp:
        if resp.status != 200:
            raise RuntimeError(f"{url} returned HTTP {resp.status}")
        return resp.read(256 * 1024)


def rollback(before: dict[str, int]) -> None:
    target = ",".join(f"{r}={p}" for r, p in before.items())
    print("ROLLBACK: restoring saved Cloud Run allocation", before, flush=True)
    cloud("run", "services", "update-traffic", SERVICE, "--region=" + REGION,
          "--to-revisions=" + target)


def main() -> None:
    branch = cmd("git", "branch", "--show-current", capture=True).strip()
    if branch != BRANCH:
        raise RuntimeError(f"Worktree must be on {BRANCH}; found {branch}")
    if cmd("git", "status", "--porcelain", capture=True).strip():
        raise RuntimeError("Local changes present: save them before publishing")
    print("=== Validate integrated release ===", flush=True)
    cmd("go", "test", "./...", cwd=ROOT / "source")
    cmd("go", "vet", "./...", cwd=ROOT / "source")
    for name in ("crm-ui.test.cjs", "theme.test.cjs", "events.test.cjs",
                 "apps-script.test.cjs"):
        cmd("node", "--test", str(ROOT / "tests" / name))

    print("=== Preserve exact service snapshot and traffic ===", flush=True)
    before_full = service()
    before = allocations(before_full)
    BACKUPS.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(BACKUPS, 0o700)
    backup = BACKUPS / (f"cloudrun-{int(time.time())}.json")
    fd = os.open(backup, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "w", encoding="utf-8") as h:
        json.dump(before_full, h, indent=2)
    print("Baseline saved securely to:", backup, flush=True)
    print("Previous production traffic:", before, flush=True)

    # Do NOT invent Apps Script, Cloud Tasks, tokens or the worker URL.
    # This first phase publishes the site and CRM without claiming lead intake.
    print("=== Stage site + CRM, do not affect existing visitors ===", flush=True)
    cloud("run", "deploy", SERVICE,
          "--region=" + REGION,
          "--source=" + str(ROOT / "source"),
          "--no-traffic",
          "--tag=" + TAG,
          "--update-env-vars=" + ",".join((
              "LIIKKUVA_BOOTSTRAP=1",
              "CRM_OWNER_EMAIL=medynikv@gmail.com",
              "CRM_PHOTO_BUCKET=liikkuva-korjaamo-crm-private",
              "CRM_PUBLIC_URL=" + DOMAIN,
          )),
          "--quiet")

    staged = service()
    if allocations(staged) != before:
        raise RuntimeError("Traffic changed unexpectedly during staging")
    routes = [x for x in staged.get("status", {}).get("traffic", [])
              if x.get("tag") == TAG]
    if len(routes) != 1 or not routes[0].get("url") or not routes[0].get("revisionName"):
        raise RuntimeError("Could not resolve staged revision and tagged URL")
    new_revision = routes[0]["revisionName"]
    preview = routes[0]["url"].rstrip("/")
    print("New revision:", new_revision, "Preview:", preview, flush=True)

    health(preview, new_revision)
    for path in ("/", "/en", "/admin", "/crm/", "/static/css/brand.css",
                 "/static/img/favicon.svg", "/privacy?lang=en"):
        b = read_url(preview + path)
        if path.endswith("brand.css"):
            css = b.decode("utf-8")
            if "#23372b" not in css or "#f97315" not in css:
                raise RuntimeError("Two-colour brand was not served")
        print("PREVIEW OK:", path, flush=True)

    print("=== Promote staged site+CRM to 100% ===", flush=True)
    if allocations(service()) != before:
        raise RuntimeError("Existing traffic changed while staging")
    cloud("run", "services", "update-traffic", SERVICE,
          "--region=" + REGION,
          "--to-revisions=" + new_revision + "=100")

    try:
        live = allocations(service())
        if live != {new_revision: 100}:
            raise RuntimeError("Cloud Run did not confirm 100% on new revision")
        # The domain may have a pinned Firebase/Cloudflare rewrite; verify real host.
        health(DOMAIN, new_revision)
        for path in ("/", "/en", "/admin", "/crm/"):
            read_url(DOMAIN + path)
            print("DOMAIN OK:", path, flush=True)
    except Exception:
        print("POST-PROMOTION VERIFICATION FAILED; restoring previous traffic", flush=True)
        rollback(before)
        raise

    print("==============================================")
    print("SITE AND CRM LIVE:", DOMAIN)
    print("PRODUCTION REVISION:", new_revision)
    print("OLD TRAFFIC CONFIG BACKUP:", backup)
    print("IMPORTANT: BOOTSTRAP=1; POST /lead is still disabled")
    print("IMPORTANT: WHATSAPP REAL BUSINESS NUMBER IS NOT CONNECTED")
    print("NEXT: configure Cloud Tasks/Apps Script/secret grants and deploy BOOTSTRAP=0")
    print("==============================================")


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, subprocess.CalledProcessError, urllib.error.URLError,
            urllib.error.HTTPError, ValueError, OSError) as exc:
        print("RELEASE FAILED:", str(exc), file=sys.stderr)
        sys.exit(1)
