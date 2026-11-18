#!/usr/bin/env bash
# Run from the actual AutoHelp checkout in Cloud Shell. Read-only, no secrets.
set -eu
if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  echo 'Запустите из рабочей папки AutoHelp.' >&2
  exit 1
fi
git branch --show-current
git log -1 --format='%h %cs %s'
git status --short
git ls-files '*settings*.go' '*config*.go' '*pricing*.go' '*texts*.go' '*models*.go' '*handlers*.go' | while IFS= read -r file; do
  if [ -f "$file" ]; then
    rg -n 'visitPriceEUR|VisitPriceEUR|DiagnosticPrice|DiagnosticEUR|diagnostic_price|visit_price|ProVisitEUR|ProDiagnosticEUR' "$file" || true
  fi
done
