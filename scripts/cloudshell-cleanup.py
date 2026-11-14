#!/usr/bin/env python3
"""Safe, reversible Cloud Shell cleanup for Liikkuva.

Default is READ-ONLY. --apply archives only top-level Liikkuva .log files and
removes Python __pycache__/bytecode inside the selected repository. Never
touches Cloud Run, Firestore, Secret Manager, credentials, other projects,
source trees, git objects, release snapshots, or Cloud SDK metadata.
"""
from __future__ import annotations

import argparse
import datetime as dt
import os
from pathlib import Path
import shutil
import subprocess
import sys
import zipfile


def size(path: Path) -> int:
    try:
        if path.is_file():
            return path.stat().st_size
        return sum(item.stat().st_size for item in path.rglob("*") if item.is_file())
    except OSError:
        return 0


def fmt(value: int) -> str:
    for unit in ("B", "KiB", "MiB", "GiB"):
        if value < 1024 or unit == "GiB":
            return f"{value:.1f} {unit}"
        value /= 1024
    return "unknown"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--apply", action="store_true", help="Archive logs and remove only disposable Python bytecode")
    args = parser.parse_args()
    home = Path.home().resolve()
    repo = (home / "liikkuva").resolve()
    if not (repo / ".git").exists() or repo.parent != home:
        raise SystemExit("Expected existing ~/liikkuva repository")
    print("===== SPACE =====")
    usage = shutil.disk_usage(home)
    print("home filesystem: total", fmt(usage.total), "used", fmt(usage.used), "free", fmt(usage.free))
    print("===== REPOSITORY STATUS =====")
    subprocess.run(["git", "-C", str(repo), "status", "--short"], check=True)
    print("===== TOP-LEVEL DIRECTORIES — INVENTORY ONLY =====")
    for p in sorted(home.iterdir()):
        if p.is_dir() and not p.name.startswith("."):
            print(f"{fmt(size(p)):>10}  {p.name}/")
    logs = sorted(p for p in home.glob("liikkuva-*.log") if p.is_file() and not p.is_symlink())
    cache_dirs = [p for p in repo.rglob("__pycache__") if p.is_dir()
                  and ".git" not in p.parts and "node_modules" not in p.parts]
    print("===== SAFE CANDIDATES =====")
    for p in logs:
        print("ARCHIVE:", p.name, fmt(size(p)))
    for p in cache_dirs:
        print("REBUILDABLE:", p.relative_to(home), fmt(size(p)))
    print("===== PROTECTED, NEVER TOUCHED =====")
    print("~/liikkuva, ~/lending, ~/projects, ~/sites, ~/mediaok, ~/go, ~/.config/gcloud,")
    print("~/.liikkuva-releases, Firestore, Cloud Run revisions, secrets and all Git data")
    if not args.apply:
        print("DRY-RUN ONLY. To archive logs and delete local Python bytecode: --apply")
        return 0
    if logs:
        archives = home / ".liikkuva-log-archives"
        archives.mkdir(mode=0o700, exist_ok=True)
        stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%d-%H%M%SZ")
        target = archives / f"liikkuva-cloudshell-logs-{stamp}.zip"
        with zipfile.ZipFile(target, "x", zipfile.ZIP_DEFLATED, compresslevel=6) as dest:
            for p in logs:
                dest.write(p, arcname=p.name)
        with zipfile.ZipFile(target) as verify:
            if verify.testzip() is not None or set(verify.namelist()) != {p.name for p in logs}:
                raise SystemExit("Archive validation failed; original logs retained")
        os.chmod(target, 0o600)
        print("ARCHIVED:", target)
        for p in logs:
            p.unlink()
    for p in cache_dirs:
        if p.is_relative_to(repo) and not p.is_symlink():
            shutil.rmtree(p)
            print("CLEARED:", p.relative_to(home))
    print("DONE. No repositories, secrets, or release backups were deleted.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
