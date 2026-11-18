#!/usr/bin/env bash
set -Eeuo pipefail

target=liikkuva-korjaamo
source_project=seventh-chassis-247718
run_region=europe-north1
scheduler_region=europe-west1
job=autohelp-crm-sync
worker_name=autohelp-crm-sync
scheduler_name=autohelp-crm-trigger
worker="$worker_name@$target.iam.gserviceaccount.com"
trigger="$scheduler_name@$target.iam.gserviceaccount.com"
repo="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
commit="$(git -C "$repo" rev-parse HEAD)"
build="$(mktemp -d)"
trap 'rm -rf -- "$build"' EXIT

# Use only the tracked CRM package and its module dependencies.
git -C "$repo" archive "$commit" source/go.mod source/go.sum source/crm source/internal |
  tar -x -C "$build" --strip-components=1
mkdir -p "$build/cmd/autohelp-crm-sync"
cat > "$build/cmd/autohelp-crm-sync/main.go" <<'GO'
package main

import (
	"context"
	"log"
	"time"

	"liikkuvakorjaamo.fi/landing/crm"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	if err := crm.SyncAutoHelp(ctx); err != nil {
		log.Fatal(err)
	}
}
GO

cat > "$build/Dockerfile" <<'DOCKER'
FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY crm ./crm
COPY internal ./internal
COPY cmd ./cmd
RUN go test ./crm && CGO_ENABLED=0 go build -trimpath -o /out/sync ./cmd/autohelp-crm-sync
FROM alpine:3.23
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /out/sync /usr/local/bin/autohelp-crm-sync
USER app
ENTRYPOINT ["/usr/local/bin/autohelp-crm-sync"]
DOCKER

active_env() {
  local project="$1" service="$2" name="$3" revision
  revision="$(gcloud run services describe "$service" \
    --project="$project" --region="$run_region" --format=json |
    python3 -c '
import json, sys
routes = json.load(sys.stdin)["status"]["traffic"]
active = [r["revisionName"] for r in routes if r.get("percent", 0) > 0]
if len(active) != 1:
    sys.exit("A single active revision is required to resolve bucket settings")
print(active[0])
')"
  gcloud run revisions describe "$revision" \
    --project="$project" --region="$run_region" --format=json |
    python3 -c '
import json, sys
containers = json.load(sys.stdin)["spec"]["containers"]
for container in containers:
    for entry in container.get("env", []):
        if entry["name"] == sys.argv[1]:
            if "valueFrom" in entry:
                sys.exit("Set the bucket name explicitly when configured through a secret")
            print(entry.get("value", ""))
            sys.exit()
' "$name"
}

autohelp_bucket="${AUTOHELP_PHOTO_BUCKET:-$(active_env "$source_project" autohelp-landing GCS_BUCKET_NAME)}"
crm_bucket="${CRM_PHOTO_BUCKET:-$(active_env "$target" liikkuva-preview CRM_PHOTO_BUCKET)}"
if [[ -n "$autohelp_bucket" && -z "$crm_bucket" ]]; then
  echo 'CRM_PHOTO_BUCKET is missing from the running CRM.' >&2
  exit 1
fi

gcloud services enable run.googleapis.com cloudbuild.googleapis.com \
  artifactregistry.googleapis.com cloudscheduler.googleapis.com --project="$target" --quiet

for name in "$worker_name" "$scheduler_name"; do
  email="$name@$target.iam.gserviceaccount.com"
  if ! gcloud iam service-accounts describe "$email" --project="$target" >/dev/null 2>&1; then
    gcloud iam service-accounts create "$name" --project="$target" --quiet
  fi
done

gcloud projects add-iam-policy-binding "$source_project" \
  --member="serviceAccount:$worker" --role=roles/datastore.viewer --condition=None --quiet >/dev/null
gcloud projects add-iam-policy-binding "$target" \
  --member="serviceAccount:$worker" --role=roles/datastore.user --condition=None --quiet >/dev/null

if [[ -n "$autohelp_bucket" ]]; then
  gcloud storage buckets add-iam-policy-binding "gs://$autohelp_bucket" \
    --member="serviceAccount:$worker" --role=roles/storage.objectViewer --quiet >/dev/null
  gcloud storage buckets add-iam-policy-binding "gs://$crm_bucket" \
    --member="serviceAccount:$worker" --role=roles/storage.objectCreator --quiet >/dev/null
fi

gcloud run jobs deploy "$job" --source="$build" --project="$target" --region="$run_region" \
  --service-account="$worker" --tasks=1 --max-retries=2 --task-timeout=30m \
  --set-env-vars="AUTOHELP_PHOTO_BUCKET=$autohelp_bucket,CRM_PHOTO_BUCKET=$crm_bucket" --quiet

gcloud run jobs add-iam-policy-binding "$job" --project="$target" --region="$run_region" \
  --member="serviceAccount:$trigger" --role=roles/run.invoker --quiet >/dev/null

echo 'First import: waiting for completion'
gcloud run jobs execute "$job" --project="$target" --region="$run_region" --wait

scheduler_action=create
if gcloud scheduler jobs describe "$job" --project="$target" --location="$scheduler_region" >/dev/null 2>&1; then
  scheduler_action=update
fi

gcloud scheduler jobs "$scheduler_action" http "$job" \
  --project="$target" --location="$scheduler_region" --schedule='*/5 * * * *' \
  --time-zone=Europe/Helsinki --http-method=POST \
  --uri="https://run.googleapis.com/v2/projects/$target/locations/$run_region/jobs/$job:run" \
  --oauth-service-account-email="$trigger" --oauth-token-scope=https://www.googleapis.com/auth/cloud-platform \
  --quiet

if [[ "$(gcloud scheduler jobs describe "$job" --project="$target" \
  --location="$scheduler_region" --format='value(state)')" == PAUSED ]]; then
  gcloud scheduler jobs resume "$job" --project="$target" --location="$scheduler_region" --quiet
fi

gcloud scheduler jobs describe "$job" --project="$target" --location="$scheduler_region" \
  --format='table(name,state,schedule,timeZone)'
echo 'Import summary:'
gcloud logging read \
  "resource.type=\"cloud_run_job\" AND resource.labels.job_name=\"$job\" AND textPayload:\"AutoHelp sync:\"" \
  --project="$target" --freshness=1h --limit=1 --format='value(textPayload)'
