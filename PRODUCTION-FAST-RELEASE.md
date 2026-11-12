# Liikkuva: expedited Cloud Shell release (12 November 2026)

The release branch is `release/production-20261112`, which combines the
latest two-colour site and existing Go CRM. **Use this branch, not the old
`feat/orange-on-green-site` or `liikkuva-crm/` standalone copy.**

## Phase 1: publish site and CRM

In the already-open Cloud Shell folder `~/liikkuva-production-20261112`:

```bash
cd ~/liikkuva-production-20261112
git fetch origin
git pull --ff-only origin release/production-20261112
python3 scripts/publish-site-crm.py
```

This script runs Go/Node tests, writes a private backup of the full Cloud Run
service configuration, deploys the **existing integrated Go application** with
`--no-traffic`, validates its pages, CRM and exact two-colour stylesheet,
switches 100% only after preview validation, then verifies the real public
domain and rolls back Cloud Run traffic if that verification fails.

It sets `LIIKKUVA_BOOTSTRAP=1`: **POST /lead remains blocked**, and this is
declared in the result. The CRM /crm routes are exempt from the site's bootstrap
block, but actual owner login requires an existing CRM user/password; GET 200
by itself does not prove account access. This script deliberately does not
change the configured Meta webhook callback, migrate the business number or
send customer messages.

## Phase 2: activate lead processing with Bootstrap=0

```bash
cd ~/liikkuva-production-20261112
python3 scripts/activate-leads.py
```

To continue, you MUST provide a real deployed Google Apps Script Web App
`https://script.google.com/macros/s/.../exec` URL and matching script token if
they are not already configured. A GitHub source file is not a deployed Web App.
Never paste tokens into ChatGPT or GitHub.

The script creates the separate Cloud Tasks queue only if absent, adds the
existing runtime account's Firestore and Cloud Tasks IAM roles, creates
unconfigured task/review secrets with secure random values, grants access to the
runtime, migrates the site's legacy `site_settings/landing` document only when
the new `liikkuva_site_settings/landing` document is missing, stages a
Bootstrap=0 revision with 0% traffic, performs health/CRM/invalid-form checks,
then promotes the verified revision. It rolls back to the previous 100% revision
if the public domain doesn't serve the expected build or form validation.

This test establishes that the form is **not blocked by Bootstrap**. You must
also send an actual test lead with your own contact details and verify the
Firestore `liikkuva_leads` and `crm_jobs` records plus Telegram, Google Sheets
and email delivery. Missing Apps Script URL or nonmatching token blocks full
pipeline delivery. Cloud Tasks replays require a real terminal status check.

## Notes

- Current Cloud Shell default project may be `seventh-chassis-247718`:
  both scripts explicitly target `liikkuva-korjaamo`.
- The real WhatsApp Business number must not be registered/migrated outside
  verified Coexistence; live callback remains on its existing tagged revision.
- Full service configuration/secret references are in private backups under
  `~/.liikkuva-releases/`, never committed.
- These scripts run in the user's authenticated Cloud Shell, not from GitHub.
  Merely committing a script does not modify Cloud Run.
- If Cloudflare or Firebase Hosting pins an old Cloud Run revision, public-domain
  verification will fail and Cloud Run traffic is restored. Inspect the
  Firebase Hosting rewrite before making any separate routing change.
