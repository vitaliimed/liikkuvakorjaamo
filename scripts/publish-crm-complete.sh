#!/usr/bin/env bash
# Publish both existing services, then connect their existing integrations.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
if [ -n "$(git status --porcelain)" ]; then
  echo "Есть локальные изменения Liikkuva. Сохраните их перед публикацией."
  exit 1
fi
if [ "$(git branch --show-current)" != "feat/crm-payments-completion-20261117" ]; then
  echo "Ожидается ветка feat/crm-payments-completion-20261117."
  exit 1
fi
umask 077
RELEASE_DIR="$HOME/.liikkuva-releases/complete-$(date -u +%Y%m%d-%H%M%S)"
mkdir -p "$RELEASE_DIR"
gcloud run services describe liikkuva-preview \
  --project=liikkuva-korjaamo --region=europe-north1 --format=json > "$RELEASE_DIR/liikkuva-before.json"
gcloud run services describe autohelp-landing \
  --project=seventh-chassis-247718 --region=europe-north1 --format=json > "$RELEASE_DIR/autohelp-before.json"
echo "Приватные резервные копии: $RELEASE_DIR"

# Prepare and test AutoHelp before changing either production service.
AUTOHELP_DIR="$HOME/autohelp-tariffs-20261118"
if [ ! -e "$AUTOHELP_DIR" ]; then
  git clone --branch feat/autohelp-crm-tariffs-20261118 --single-branch \
    https://github.com/vitaliimed/lending.git "$AUTOHELP_DIR"
fi
if [ "$(git -C "$AUTOHELP_DIR" remote get-url origin)" != "https://github.com/vitaliimed/lending.git" ] || \
   [ "$(git -C "$AUTOHELP_DIR" branch --show-current)" != "feat/autohelp-crm-tariffs-20261118" ] || \
   [ -n "$(git -C "$AUTOHELP_DIR" status --porcelain)" ]; then
  echo "Папка $AUTOHELP_DIR содержит другую ветку, репозиторий или несохранённые изменения."
  exit 1
fi
git -C "$AUTOHELP_DIR" pull --ff-only origin feat/autohelp-crm-tariffs-20261118
(cd "$AUTOHELP_DIR" && go test ./... && go build -o "$RELEASE_DIR/autohelp-check" .)

python3 update.py --project=liikkuva-korjaamo --region=europe-north1 --service=liikkuva-preview
echo "Публикую адаптер тарифов AutoHelp на рабочий сервис…"
gcloud run deploy autohelp-landing --source="$AUTOHELP_DIR" \
  --project=seventh-chassis-247718 --region=europe-north1 --quiet --format=json \
  > "$RELEASE_DIR/autohelp-deployed.json"
AUTOHELP_REVISION="$(python3 - "$RELEASE_DIR/autohelp-deployed.json" <<'PY'
import importlib.util,json,sys
spec=importlib.util.spec_from_file_location('setup','scripts/connect-crm-telegram.py')
setup=importlib.util.module_from_spec(spec)
spec.loader.exec_module(setup)
print(setup.ready_created_revision(json.load(open(sys.argv[1])), 'seventh-chassis-247718', 'europe-north1'))
PY
)"
gcloud run services update-traffic autohelp-landing --to-revisions="$AUTOHELP_REVISION=100" \
  --project=seventh-chassis-247718 --region=europe-north1 --quiet --format=json \
  > "$RELEASE_DIR/autohelp-traffic.json"
python3 scripts/connect-autohelp-tariffs.py --apply
bash scripts/install-autohelp-crm-sync.sh
python3 scripts/connect-crm-telegram.py --apply
echo "Публикация и подключения завершены. Откройте CRM и отправьте /start существующему боту."
