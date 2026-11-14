# Production candidate — 12 November 2026

## Result: NOT DEPLOYED

Release gate is blocked. No Cloud Run revision was created, no traffic was
changed, no production data was written, and Bootstrap was not changed.

## Source

- Site: `feat/orange-on-green-site` at `506651dd3bff09f469f4ca0ce7a459748922cdaa`.
- CRM: `feat/crm-whatsapp-orange-ui` at `4e818920e198044d86188084bb9352964733ae9d`.
- CRM includes `feat/crm-complete-v2` at `99d50df4f9ca61b99a95d88e7e6596ef890d5322` (ancestry verified).
- Integrated branch: `release/production-20261112`.
- Runtime version: `2026.11.12-production-candidate`.
- Deployable Go application: `source/`, including integrated `source/crm/`.
  Do not deploy the older standalone `liikkuva-crm/` in place of this application.

The branches merged without conflicts. Owner/mechanic access, calendar,
financial ledger, private photos and Meta HMAC webhook verification are retained.
CRM logos and palette now use #23372b / #f97315. CRM emails retain the message
language from the job and use the same logo and palette as the public site.
The public server and browser require a vehicle registration number. English
privacy has its own canonical URL. Existing saved accent overrides are normalized
to the approved orange by the site branch; the actual database was not inspected.

Lead and CRM job writes remain atomic. A Firestore transaction lease serializes
concurrent processing of the same lead; completed replays do not touch the CRM job.
Confirmation SMTP failures now return an error to Cloud Tasks instead of being
acknowledged as completed. A worker-crash window between an external SMTP/Telegram
acceptance and its durable acknowledgement still requires reconciliation; these
providers do not offer a transaction shared with Firestore. Do not claim absolute
exactly-once delivery across that failure window.

## Verification completed

- Go tests with the Firestore emulator: **93 passed**, 2 optional preview/browser
  tests skipped (`TestWriteAdminPreview`, `TestAdminBrowserLoginFlow`).
- Emulator checks include mechanic/private-data access, owner authorization,
  booking concurrency, finance/ledger, atomic intake, WhatsApp signed requests,
  duplicate events, app echoes, statuses and message binding.
- Added emulator test: synthetic site lead and CRM job saved together, photo
  reference retained, second simultaneous worker rejected, two completed
  replays leave the CRM job unchanged. No production writes or external sends.
- `go vet ./...`: passed.
- `go build -buildvcs=false`: passed; same VCS flag as the Dockerfile.
- Node tests: **14 passed** (CRM UI, theme, analytics, Apps Script).
- Python tests: **28 passed** (existing release/update/integration suite).
- Local compiled server: HTTP 200 for `/`, `/en`, `/admin`, `/privacy?lang=en`,
  `/review?lang=en`, `/hinnasto`, `/palvelualue`, `/vikadiagnostiikka`, brand CSS,
  favicon and `/api/health`. Local smoke uses Bootstrap=1 and no cloud project;
  it does not prove production ingestion or CRM initialization.
- Public-page canonical regression tests passed for FI/EN and English privacy.
- Browser visual desktop/mobile/dark-mode review is **outstanding**. Node theme
  tests and CSS/logo regression tests are not a substitute for visual review.

## Actual domain observations

Read-only requests to https://liikkuvakorjaamo.fi returned:

| Route | Result |
| --- | --- |
| `/`, `/en`, `/admin`, `/privacy?lang=en`, `/hinnasto` | 200 |
| `/crm/` | **404** |
| `/api/health` | `revision=liikkuva-preview-00021-loz`, `version=2026.11.01-pine-settings` |

This proves which revision answered the domain, not the complete Cloud Run traffic
allocation or whether Firebase Hosting pins that revision. Neither was available
for authenticated inspection.

## Blockers / unverified production requirements

`python3 prod_preflight.py` exits **2**: `gcloud` is unavailable. No authenticated
Cloud SDK profile was present; the Cloud Console browser showed “Site Unavailable”.
No suitable Cloud Run connector was available.

Thus the following are UNKNOWN, not passed: current Bootstrap value; service
account permissions; database/bucket access and existing-data counts; Cloud Tasks
queue/invocation; live Telegram/Apps Script/Brevo delivery; CRM owner login on the
production domain; current Meta webhook delivery; Firebase rewrite configuration.
No real test lead, email or WhatsApp message was sent. The business number was not
migrated, coexistence/import was not started, and automatic client messaging was
not enabled.

The preflight now also blocks missing SMTP settings, inspects the named native
Firestore database, requires an explicitly identified runtime service account,
checks project-level datastore/tasks bindings and CRM owner/photo configuration.
Inherited/custom IAM that is not proven by those bindings remains a blocker for
manual verification. Preflight does not prove effective access or delivery.

## Continue only in an authenticated Cloud Shell

Check out the integration branch, run `python3 prod_preflight.py`, and resolve the
reported infrastructure issues. Do not invoke `update.py` as an unchecked next
step: its current workflow can promote traffic without all requested CRM and
real-delivery checks. Preserve the full service and Firebase Hosting snapshots
before staging; they contain operational configuration and must not be committed.

Create the candidate revision with `--no-traffic`, retain the previous revisions,
verify all required live checks with the owner's test data, then promote only the
validated revision. Verify the domain's `/api/health`, CRM, form receipt and storage
again after promotion. Bootstrap=0 is conditional on successful infrastructure
validation, not a shortcut around configuration checks.

## Rollback

No rollback is currently needed: this session made no production changes.
Before a future release, save the complete `gcloud run services describe` JSON and
Firebase Hosting active version/rewrite. Restore the exact saved revision/percent
map, rather than assuming the last domain-observed revision owns all traffic:

```bash
gcloud run services update-traffic liikkuva-preview \
  --project=liikkuva-korjaamo --region=europe-north1 \
  --to-revisions='SAVED_REVISION_1=SAVED_PERCENT_1,SAVED_REVISION_2=SAVED_PERCENT_2'
```

Substitute only entries from the verified snapshot; for a single old revision use
`SAVED_REVISION=100`. If Hosting was changed or pins a revision, restore its saved
release too. Check domain `/api/health` and a safe read of CRM after rollback.
