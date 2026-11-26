/* Run against Go-rendered fixtures served on localhost. No live leads. */
const { chromium } = require('playwright');
const assert = require('node:assert/strict');

(async () => {
  const browser = await chromium.launch({channel:'chrome', args:['--no-sandbox']});
  try {
    const page = await browser.newPage();
    await page.route('**/*', route => {
      const host = new URL(route.request().url()).hostname;
      return host === '127.0.0.1' ? route.continue() : route.abort();
    });
    const url = 'http://127.0.0.1:8091/en.html';
    for (const action of ['accept','reject','settings']) {
      await page.goto(url);
      await page.evaluate(() => localStorage.clear());
      await page.reload();
      await page.locator('[data-consent-action="'+action+'"]').click({trial:true});
      assert(await page.locator('main').evaluate(el => el.inert));
      await page.evaluate(() => { window.gtag = () => { throw new Error('Tracking unavailable'); }; });
      await page.locator('[data-consent-action="'+action+'"]').click();
      if (action === 'settings') await page.locator('[data-consent-modal-action="save"]').click();
      assert.equal(await page.locator('main').evaluate(el => el.inert), false);
      await page.locator('#leadForm input[name=name]').fill('Unlocked');
      assert.equal(await page.locator('#leadForm input[name=name]').inputValue(),'Unlocked');
      await page.reload();
      assert.equal(await page.locator('.liikkuva-consent-root:visible').count(),0);
    }
    const overflow = [];
    for (const file of ['index.html','en.html','area.html']) {
      await page.goto('http://127.0.0.1:8091/'+file);
      assert.equal(await page.locator('h1').count(),1);
      for (const width of [320,375,390,430,768,1024,1280,1366,1440,1920]) {
        await page.setViewportSize({width,height:850});
        if(width>=1024) {
          const form=await page.locator('.hero-form').boundingBox();
          const copy=await page.locator('.hero-copy').boundingBox();
          assert(form.x>=copy.x+copy.width-1,'desktop order '+width);
        }
        const wide = await page.evaluate(() => {
          if (document.documentElement.scrollWidth <= innerWidth+1) return [];
          return Array.from(document.querySelectorAll('body *')).filter(el =>
            el.getBoundingClientRect().right > innerWidth+1 && !el.closest('.review-grid')
          ).map(el=>el.tagName+'.'+el.className).slice(0,12);
        });
        if(wide.length) overflow.push({file,width,wide});
      }
    }
    await page.locator('[data-slide="1"]').click();
    await page.waitForFunction(()=>document.querySelector('.review-grid').scrollLeft>0);
    assert.equal(await page.locator('[data-offer-open]').count(),0);
    await page.evaluate(()=>{
      const mouse=(type,y)=>document.dispatchEvent(new MouseEvent(type,{clientX:150,clientY:y}));
      mouse('mousemove',10); mouse('mouseout',0);
      if(document.querySelector('#exitOffer').open) throw Error('Unexpected offer');
      mouse('mousemove',180); mouse('mouseout',0);
    });
    await page.locator('#exitOffer input[name=name]').fill('Fixture');
    await page.keyboard.press('Escape');
    assert.equal(await page.locator('#footerLeadFormMount input[name=name]').inputValue(),'Fixture');
    assert(!overflow.length, JSON.stringify(overflow));
    const touch=await browser.newContext({isMobile:true,hasTouch:true,viewport:{width:390,height:844}});
    const mobile=await touch.newPage();
    await mobile.route('**/*',r=>new URL(r.request().url()).hostname==='127.0.0.1'?r.continue():r.abort());
    await mobile.goto(url);
    await mobile.locator('[data-consent-action="reject"]').click();
    await mobile.evaluate(()=>{document.documentElement.style.scrollBehavior='auto';window.scrollTo(0,2000);});
    await mobile.waitForFunction(()=>scrollY>1500);
    await mobile.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>{window.scrollTo(0,0);resolve();}))));
    assert.equal(await mobile.locator('#exitOffer').evaluate(el=>el.open),false);
    await mobile.evaluate(()=>{
      window.scrollTo(0,200);
      const touch=(type,y)=>{
        const point=new Touch({identifier:1,target:document.body,clientX:120,clientY:y});
        window.dispatchEvent(new TouchEvent(type,{touches:type==='touchstart'?[point]:[],changedTouches:[point]}));
      };
      touch('touchstart',80); window.scrollTo(0,0); touch('touchend',300);
    });
    await mobile.waitForFunction(()=>document.querySelector('#exitOffer').open);
    await touch.close();
    console.log('FI, EN and service landing: 10 viewport widths, consent, carousel and offer passed.');
  } finally { await browser.close(); }
})().catch(err=>{console.error(err);process.exitCode=1;});
