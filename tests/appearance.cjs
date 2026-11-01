const {chromium}=require('playwright');
const fs=require('fs'),path=require('path'),http=require('http'),assert=require('node:assert/strict');
const root=path.resolve(__dirname,'..'),qa=process.env.LIIKKUVA_QA_DIR;
const out=process.env.ARTIFACT_DIR||path.join(root,'.qa-artifacts/appearance');fs.mkdirSync(out,{recursive:true});
const server=http.createServer((req,res)=>{const u=new URL(req.url,'http://localhost');let p;
if(u.pathname.startsWith('/static/'))p=path.join(root,'source',u.pathname);
else p=path.join(qa,path.basename(u.pathname)+'.html');
try{res.setHeader('Content-Type',p.endsWith('.css')?'text/css':p.endsWith('.js')?'application/javascript':p.endsWith('.svg')?'image/svg+xml':p.endsWith('.html')?'text/html':'application/octet-stream');res.end(fs.readFileSync(p));}catch{res.statusCode=404;res.end();}});
function luminance(s){const c=s.match(/[\d.]+/g).slice(0,3).map(x=>{x=Number(x)/255;return x<=.04045?x/12.92:((x+.055)/1.055)**2.4});return c[0]*.2126+c[1]*.7152+c[2]*.0722;}
(async()=>{await new Promise(r=>server.listen(0,'127.0.0.1',r));const base='http://127.0.0.1:'+server.address().port;const b=await chromium.launch({executablePath:process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,args:['--no-sandbox']});let count=0;let measured={};
try{for(const width of [390,1280,1920])for(const size of [16,18,20,22])for(const theme of ['light','dark']){
const c=await b.newContext({viewport:{width,height:width===1280?720:900}});await c.route('**/*',r=>r.request().url().startsWith(base)?r.continue():r.abort());const p=await c.newPage();await p.goto(`${base}/appearance-${size}-${theme}`);await p.getByRole('button',{name:'Vain välttämättömät',exact:true}).click();
const values=await p.evaluate(()=>{const f=document.querySelector('.hero-request'),a=document.querySelector('.capability-grid article'),t=a.querySelector('li'),logo=document.querySelector('.header .brand img'),btn=document.querySelector('.btn-submit');return {scroll:document.documentElement.scrollWidth,font:parseFloat(getComputedStyle(document.querySelector('.hero-description')).fontSize),form:f.getBoundingClientRect().toJSON(),bg:getComputedStyle(a).backgroundColor,text:getComputedStyle(t).color,heading:getComputedStyle(a.querySelector('h3')).color,accent:getComputedStyle(btn).backgroundColor,logo:logo.getBoundingClientRect().toJSON(),brandText:document.querySelector('.header .brand').textContent.trim()};});
assert(values.scroll<=width+1,`Overflow ${width} ${size} ${theme}: ${values.scroll}`);assert.equal(values.brandText,'');assert(values.logo.width>100&&values.logo.height<80);assert.equal(values.accent,'rgb(32, 56, 50)');
if(theme==='dark'){for(const fg of [values.text,values.heading]){const l1=luminance(fg),l2=luminance(values.bg),contrast=(Math.max(l1,l2)+.05)/(Math.min(l1,l2)+.05);assert(contrast>=4.5,`Contrast ${contrast}`);}}
if(width===1280&&size<=20)assert(values.form.bottom<=720,`Form extends below screen ${size}: ${values.form.bottom}`);
measured[`${width}-${size}-${theme}`]=values.font;
if(size===20&&[390,1280].includes(width)){await p.screenshot({path:path.join(out,`pine-${width}-${theme}.jpg`),type:'jpeg',quality:75});await p.locator('.capability').scrollIntoViewIfNeeded();await p.waitForTimeout(250);await p.locator('.capability').screenshot({path:path.join(out,`pine-capability-${width}-${theme}.jpg`),type:'jpeg',quality:75});}
await c.close();count++;
}assert(measured['1280-20-light']>measured['1280-18-light'],'Font setting did not change rendered text');console.log(`PASS ${count} appearance scenarios; custom logo, accent, 16–22 font scaling and dark contrast.`);}finally{await b.close();server.close();}})().catch(e=>{console.error(e);server.close();process.exitCode=1});
