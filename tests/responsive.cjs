/* Public layout checks plus optional synthetic admin template checks.
   No lead, SMTP or production database writes are performed. */
const { chromium } = require('playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const http = require('node:http');
const { spawn } = require('node:child_process');
const root = path.resolve(__dirname, '..');
const artifactDir = process.env.ARTIFACT_DIR || path.join(root, '.qa-artifacts');
const fixtureDir = process.env.LIIKKUVA_QA_DIR;
const localBinary = process.env.LIIKKUVA_TEST_BINARY;
let base = process.env.BASE_URL || 'http://127.0.0.1:8080';
const sizes = [[320,568],[360,800],[375,667],[390,844],[430,932],[768,1024],[1024,768],[1101,800],[1200,800],[1366,768],[1440,900],[1920,1080],[2560,1440],[3840,2160]];
const children = [];
let proxy, browser;
const pause = ms => new Promise(resolve => setTimeout(resolve,ms));
async function ready(url) { for(let i=0;i<100;i++){try{const r=await fetch(url);if(r.ok)return;}catch{} await pause(100);}throw Error('Local server did not become ready'); }
function bounds() {
 const hero=document.querySelector('.hero-copy'); const h1=document.querySelector('.hero h1'); const cta=document.querySelector('.hero-request button[type=submit]');
 const rect=h1?.getBoundingClientRect(); const range=document.createRange();if(h1)range.selectNodeContents(h1);
 return {width:innerWidth,scroll:document.documentElement.scrollWidth,hero:hero?{height:hero.clientHeight,scroll:hero.scrollHeight}:null,
  title:rect?{x:rect.x,right:rect.right,y:rect.y,bottom:rect.bottom}:null,
  textRects:h1?[...range.getClientRects()].map(r=>({x:r.x,right:r.right})):[],cta:cta?cta.getBoundingClientRect().toJSON():null};
}
(async()=>{
 fs.mkdirSync(artifactDir,{recursive:true});
 if(localBinary){
  const server=spawn(localBinary,[],{cwd:path.join(root,'source'),env:{...process.env,LIIKKUVA_BOOTSTRAP:'1',SITE_URL:'http://127.0.0.1:8080',PORT:'8080',FIRESTORE_PROJECT_ID:'',GOOGLE_CLOUD_PROJECT:''},stdio:['ignore','ignore','pipe']});children.push(server);
  let errors='';server.stderr.on('data',b=>{errors+=b});server.on('error',e=>{errors+=e.message});await ready(base+'/healthz');
 }
 if(fixtureDir){
  const upstream=new URL(base);
  proxy=http.createServer((req,res)=>{
   const u=new URL(req.url,'http://localhost');let file;
   if(u.pathname==='/admin')file='admin-'+(u.searchParams.get('tab')||'overview')+'.html';
   else if(u.pathname==='/admin/access')file='access-access.html';
   else if(u.pathname.startsWith('/qa/access-'))file=path.basename(u.pathname)+'.html';
   if(file&&fs.existsSync(path.join(fixtureDir,file))){res.writeHead(200,{'Content-Type':'text/html; charset=utf-8'});return fs.createReadStream(path.join(fixtureDir,file)).pipe(res);}
   const outgoing=http.request({hostname:upstream.hostname,port:upstream.port,path:req.url,method:req.method,headers:req.headers},r=>{res.writeHead(r.statusCode,r.headers);r.pipe(res)});outgoing.on('error',()=>{res.writeHead(502);res.end()});req.pipe(outgoing);
  });await new Promise(resolve=>proxy.listen(8081,'127.0.0.1',resolve));base='http://127.0.0.1:8081';
 }
 browser=await chromium.launch({headless:true,executablePath:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH||undefined,args:['--no-sandbox']});
 const report=[];
 for(const [width,height] of sizes){
  const context=await browser.newContext({viewport:{width,height},isMobile:width<600,hasTouch:width<600});
  await context.route('**/*',r=>r.request().url().startsWith(base)||r.request().url().startsWith('data:')?r.continue():r.abort());
  const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
  for(const lang of ['fi','en']){
   await page.goto(base+(lang==='fi'?'/':'/en'),{waitUntil:'load'});await page.evaluate(()=>document.fonts.ready);
   const b=await page.evaluate(bounds);
   assert(b.scroll<=width+1,`Public ${lang} ${width}: horizontal overflow ${b.scroll}`);
   assert(b.hero.scroll<=b.hero.height+1,`Hero content clipped at ${width}`);
   assert(b.title.x>=-1&&b.title.right<=width+1,`Title outside screen at ${width}`);
   assert(b.textRects.every(r=>r.x>=-1&&r.right<=width+1),`Title text outside screen at ${width}`);
   assert(b.cta.width>0&&b.cta.x>=0&&b.cta.right<=width+1,`CTA outside screen at ${width}`);
   if([390,1920,3840].includes(width))await page.screenshot({path:path.join(artifactDir,`landing-${lang}-${width}.png`)});
   const necessary=page.getByRole('button',{name:/Vain välttämättömät|Only necessary|Necessary only/});
   if(await necessary.count())await necessary.first().click();
   if(width<600){
    const menu=page.locator('#menuToggle');await menu.click();assert.equal(await menu.getAttribute('aria-expanded'),'true');await page.keyboard.press('Escape');assert.equal(await menu.getAttribute('aria-expanded'),'false');
   }
   report.push({page:'landing-'+lang,width,height,bounds:b});
  }
  assert.deepEqual(errors,[],`Public JS error ${width}`);
  await context.close();
 }
 if(fixtureDir){
  for(const width of [320,390,768,1440,2560]){
   const context=await browser.newContext({viewport:{width,height:900}});const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
   for(const tab of ['overview','general','assets','analytics','emails','leads','help','seo']){
    await page.goto(base+'/admin?tab='+tab);const b=await page.evaluate(()=>({scroll:document.documentElement.scrollWidth,width:innerWidth}));assert(b.scroll<=width+1,`Admin ${tab} overflows ${width}: ${b.scroll}`);
    if(width===390&&['overview','general'].includes(tab))await page.screenshot({path:path.join(artifactDir,`admin-${tab}-${width}.png`)});
   }
   await page.goto(base+'/admin?tab=general');const price=page.locator('[name="ProVisitEUR"]');await price.fill('82');
   if(width<=900)await page.getByRole('combobox',{name:'Раздел кабинета'}).selectOption(base+'/admin?tab=seo');else await page.locator('.tabs-nav a[href="/admin?tab=seo"]').click();
   assert.equal(await page.locator('[name="ProVisitEUR"]').inputValue(),'82','Tab switch lost unsaved price');assert.equal(await page.locator('.pro-save').textContent(),'Сохранить изменения');
   const search=page.getByRole('searchbox',{name:'Поиск по всем настройкам'}).filter({visible:true});await search.fill('Выезд,');
   await page.locator('.admin-search-results:visible button').first().click();assert.equal(await price.inputValue(),'82');
   for(const mode of ['login','recovery','reset','access']){
    await page.goto(base+'/qa/access-'+mode);const b=await page.evaluate(()=>({scroll:document.documentElement.scrollWidth,width:innerWidth}));assert(b.scroll<=width+1,`Access ${mode} overflows ${width}`);
    if(width===390&&mode==='login')await page.screenshot({path:path.join(artifactDir,'login-390.png')});
   }
   assert.deepEqual(errors,[],`Admin JS error ${width}`);report.push({page:'admin-and-access',width});await context.close();
  }
 }
 fs.writeFileSync(path.join(artifactDir,'responsive-report.json'),JSON.stringify(report,null,2));
 console.log(`PASS: ${report.length} layout scenarios; both languages, menu behavior, admin navigation and unsaved changes.`);
})().catch(e=>{console.error(e.stack);process.exitCode=1}).finally(async()=>{if(browser)await browser.close();if(proxy)proxy.close();children.forEach(p=>p.kill('SIGTERM'));});
