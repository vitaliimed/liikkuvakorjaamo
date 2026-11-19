# SEO validation — 2026-11-14

Base: `3aff9d1`, branch `fix/hero-form-theme-autofill-20261112`. SEO branch: `seo`.

Passed:

- Go build using Go 1.25 toolchain with automatic module toolchain resolution.
- `go test ./... -run '^TestSEO' -count=1`: all SEO tests passed.
- Coverage includes 6 editorial routes, sitemap XML (8 URLs), unique titles, canonical, single H1, JSON-LD parsing, admin-driven 75/60/135 prices, GET/HEAD/POST, slash redirects preserving UTM, preview noindex, unknown URL 404, emergency noindex switch, legacy metadata migration, FI/EN rendered homepages with stored noindex and preserved lead form.
- Python tests: 28 tests under tests/, 3 test_prepare tests.
- Node: 10 analytics scenarios; Apps Script schema/authentication/isolation/escaping; 4 theme tests.
- `git diff --check`.

Full Go suite: three failures, all reproduced unchanged on the clean base `3aff9d1`:

1. `TestThemeSettingsSaveAndRenderWithoutAttributeInjection`: missing data-theme-picker.
2. `TestConfirmationEmailContainsRequiredMessage`: confirmation email missing 75 €.
3. `TestHeroFormAndRequiredFieldsInBothLanguages`: form ordering assertion.

These failures are not caused by the SEO diff; they remain unresolved. Do not represent the entire suite as green.

Visual browser QA was attempted but not completed: installed Chromium cannot start in this environment (socket operation not permitted); Playwright browser download also failed. Responsive CSS is provided, but viewport rendering still needs review on the deployed preview.

No production deployment, no Cloud Run traffic change, no Firestore settings write, no messages sent, no paid ads enabled.

Before production:

1. Deploy the seo branch source/ to a no-traffic preview using the established deployment workflow; retain CRM integration and current secrets/settings.
2. Check all sitemap URLs, mobile/desktop layout, lead form, consent UI, active metadata overrides and admin prices.
3. Confirm the actual Host reaching the app through Firebase/custom domain. Only liikkuvakorjaamo.fi and www.liikkuvakorjaamo.fi are indexable; previews intentionally have noindex.
4. Check `LIIKKUVA_SEO_NOINDEX` is not 1 for production. With 1, all pages retain an HTTP noindex header. LandingRobotsMeta's old noindex no longer controls the public launch; this is an intentional migration.
5. Confirm robots.txt does not block intended public pages in the stored admin settings and CDN bot rules.
6. Review the three baseline test failures before using a deploy workflow that gates on the full suite. Do not bypass test gates silently.
7. After publishing, fetch production HTML and HTTP headers again, submit sitemap in Search Console and request indexing. Branch creation alone cannot remove the live noindex.

Prices and metadata: new prices are rendered through existing admin settings, not copied from the live site into hardcoded HTML. Existing custom SEO overrides remain authoritative and need preview verification. Service pages retain their language (FI); no false EN hreflang pair is advertised.
