/* Used by TestAdminBrowserLoginFlow against a local HTTPS Go server only.
   No production credentials, Firestore access or real email delivery. */
const { chromium } = require('playwright');
const assert = require('node:assert/strict');
const base = process.argv[2];
assert(/^https:\/\/127\.0\.0\.1:\d+$/.test(base), 'Browser test requires the local Go fixture');

(async () => {
  const browser = await chromium.launch({headless: true,
    executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || undefined,
    args: ['--no-sandbox']});
  let password = 'test-password-long-enough';
  try {
    for (const width of [390, 1366]) {
      const context = await browser.newContext({ignoreHTTPSErrors: true,
        viewport: {width, height: 900}, isMobile: width < 600, hasTouch: width < 600});
      await context.route('**/*', route => route.request().url().startsWith(base + '/')
        ? route.continue() : route.abort());
      const page = await context.newPage();
      page.setDefaultTimeout(15000);
      const errors = [];
      page.on('pageerror', error => errors.push(error.message));
      const submit = async (locator, pathname, status = 303) => {
        const [response] = await Promise.all([
          page.waitForResponse(r => new URL(r.url()).pathname === pathname && r.request().method() === 'POST'),
          locator.click(),
        ]);
        assert.equal(response.status(), status, `${pathname}: unexpected POST response`);
        await page.waitForLoadState('load');
      };
      const login = async (value, status = 303) => {
        await page.locator('#login-password').fill(value);
        await submit(page.getByRole('button', {name: 'Войти в кабинет →'}), '/admin/login', status);
        if (status === 303) {
          await page.waitForURL(base + '/admin?tab=overview');
          assert(await page.getByRole('heading', {name: 'Кабинет Liikkuva', exact: true}).isVisible());
        }
      };

      // An already open page from the broken release must recover gracefully.
      if (width === 390) {
        await page.goto(base + '/__test/legacy-login');
        await login(password, 403);
        assert(await page.locator('#login-password').isVisible(), 'Rejected old form must offer a new login form');
      } else {
        assert.equal((await page.goto(base + '/admin')).status(), 200);
      }
      await login(password);
      const session = (await context.cookies()).find(c => c.name === '__session');
      assert(session && session.value.length === 43 && session.httpOnly && session.secure && session.sameSite === 'Lax');
      await page.reload();
      assert(await page.getByRole('heading', {name: 'Кабинет Liikkuva', exact: true}).isVisible(), 'Session must survive reload');

      await page.goto(base + '/admin?tab=general');
      await page.locator('[name="ProVisitEUR"]').fill(width === 390 ? '91' : '92');
      await submit(page.locator('.pro-save'), '/admin/save');
      await page.waitForURL(/saved=1/);
      assert.equal(await page.locator('[name="ProVisitEUR"]').inputValue(), width === 390 ? '91' : '92');

      await page.goto(base + '/admin/access');
      await page.locator('#password-current').fill(password);
      password = 'changed-browser-password-' + width;
      await page.locator('#password-new').fill(password);
      await page.locator('#password-repeat').fill(password);
      await submit(page.getByRole('button', {name: 'Изменить пароль', exact: true}), '/admin/access');
      await page.waitForURL(base + '/admin?notice=password');
      await login(password);
      await submit(page.getByRole('button', {name: 'Выйти', exact: true}), '/admin/logout');
      await page.waitForURL(base + '/admin');
      await login('deliberately-wrong-password', 401);
      await login(password);
      await submit(page.getByRole('button', {name: 'Выйти', exact: true}), '/admin/logout');
      await page.waitForURL(base + '/admin');

      if (width === 390) {
        await page.goto(base + '/admin/recovery');
        await submit(page.getByRole('button', {name: 'Отправить ссылку восстановления'}), '/admin/recovery', 200);
        const {token} = await (await context.request.get(base + '/__test/recovery-token')).json();
        assert.equal(token.length, 43, 'Local mail fixture must capture the recovery token');
        await page.goto(base + '/admin/reset-password?token=' + token);
        password = 'recovered-browser-password-390';
        await page.locator('#new-password').fill(password);
        await page.locator('#confirm-password').fill(password);
        await submit(page.getByRole('button', {name: 'Сохранить новый пароль'}), '/admin/reset-password');
        await page.waitForURL(base + '/admin?notice=password');
        await login(password);
        await submit(page.getByRole('button', {name: 'Выйти', exact: true}), '/admin/logout');
        await page.waitForURL(base + '/admin');
      }
      assert.deepEqual(errors, [], 'Admin JavaScript must not fail');
      console.log(`PASS ${width}px: native login, Hosting proxy, secure session, reload, save, password change, logout, wrong-password retry` + (width === 390 ? ', recovery and reset' : ''));
      await context.close();
    }
  } finally {
    await browser.close();
  }
})().catch(error => { console.error(error.stack); process.exitCode = 1; });
