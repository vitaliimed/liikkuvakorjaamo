// Local-only UI checks. Lead submissions are intercepted; no customer data is sent.
const {chromium}=require('playwright');
const assert=require('node:assert/strict');
(async()=>{
 const browser=await chromium.launch({headless:true,executablePath:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH||'/usr/bin/chromium',args:['--no-sandbox']});
 try {
 const page=await browser.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const base=process.env.BASE_URL||'http://127.0.0.1:8087';
 for(const lang of ['/','/en']){
  await page.goto(base+lang);await page.waitForSelector('input[name=car]');
  const inputs=page.locator('input[name=car]');assert.ok(await inputs.count()>=2,'hero and footer');
  for(let i=0;i<await inputs.count();i++){
   const field=inputs.nth(i);
   for(const [input,want] of [['abc123','ABC-123'],['123abc','123-ABC'],['123 abc','123-ABC'],['ABC1234','ABC1234'],['1ABC23','1ABC23']]){
    await field.fill(input);assert.equal(await field.inputValue(),want,lang+' form '+i);
   }
  }
  assert.ok((await page.locator('body').innerText()).includes('123-ABC'),'visible format hint');
 }
 await page.goto(base+'/crm/?demo');await page.waitForSelector('#search');
 const search=page.locator('#search input');await search.fill('123abc');assert.equal(await search.inputValue(),'123-ABC');
 await page.locator('#search button[type=submit]').click();assert.ok((await page.locator('h1').innerText()).includes('123-ABC'));
 await page.locator('#clearsearch').click();assert.equal(await page.locator('#search input').inputValue(),'');
 await page.locator('#create').click();const plate=page.locator('#modal input[name=plate]');
 await plate.fill('abc123');assert.equal(await plate.inputValue(),'ABC-123');
 await plate.fill('ABC1234');assert.equal(await plate.evaluate(el=>el.checkValidity()),false);
 await plate.fill('123abc');assert.equal(await plate.evaluate(el=>el.checkValidity()),true);
 await page.locator('#cancelmodal').click();
 for(const width of [375,1440]){
  await page.setViewportSize({width,height:900});
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'CRM width '+width);
 }
 assert.deepEqual(errors,[]);console.log('PASS browser: FI/EN hero/footer; CRM search/reset/create; desktop/mobile; no JS errors');
 } finally {await browser.close()}
})().catch(e=>{console.error(e);process.exit(1)});
