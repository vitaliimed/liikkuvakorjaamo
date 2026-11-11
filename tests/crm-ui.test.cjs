const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const crypto=require('node:crypto').webcrypto;
function app(demo=true){
 const elements=new Map();
 const element=()=>({style:{},dataset:{},addEventListener(){},querySelector(){return element()},showModal(){},close(){}});
 const context=vm.createContext({crypto,Intl,Date,URLSearchParams,console,setTimeout(){},location:{search:demo?'?demo':''},document:{querySelector(s){if(!elements.has(s))elements.set(s,element());return elements.get(s)},querySelectorAll(){return []}}});
 let source=fs.readFileSync('source/crm/web/app.js','utf8');
 source=source.split('\n').filter(line=>!line.startsWith('(async()=>')).join('\n');
 vm.runInContext(source,context);
 vm.runInContext('let captured="";shell=html=>{captured=html};seed();',context);
 return {run:s=>vm.runInContext(s,context),html:()=>vm.runInContext('captured',context)};
}
test('mechanic detail contains technical controls without client composer',()=>{
 const a=app();a.run('me={id:"m1",name:"Mechanic",role:"mechanic",active:true};selected={job:jobs[0],events:[]};detail()');
 assert.doesNotMatch(a.html(),/id="copyDraft"|id="composeMail"|Ответ клиенту/);
 assert.match(a.html(),/technical_note/);
});
test('owner sees drafts and mail composer',()=>{
 const a=app();a.run('selected={job:jobs[0],events:[],draft:"TEST_DRAFT"};detail()');
 assert.match(a.html(),/id="copyDraft"/);assert.match(a.html(),/id="composeMail"/);
});
test('finance renders totals independent of loaded job list',()=>{
 const a=app();a.run('view="finance";jobs=[];financeData={total:{jobs:105,closed:104,reported_cents:11000,accepted_cents:10500},brands:{},mechanics:{},payments:{cash:10500}};render()');
 assert.match(a.html(),/105/);assert.match(a.html(),/Вся база/);
});
test('calendar displays records absent from the loaded job page',()=>{
 const a=app(false);a.run('calendarJobs=[{...jobs[0],plate:"CALENDAR-ONLY"}];jobs=[];view="calendar";render()');
 assert.match(a.html(),/CALENDAR-ONLY/);
 assert.doesNotMatch(a.html(),/Подтверждённых выездов пока нет/);
});
test('financial controls and WhatsApp handoff remain owner-only',()=>{
 const a=app();a.run('selected={job:jobs[0],events:[],draft:"TEST"};detail()');assert.match(a.html(),/id="openLedger"/);assert.match(a.html(),/id="openWhatsAppDraft"/);
 a.run('me={id:"m1",name:"Mechanic",role:"mechanic"};detail()');assert.doesNotMatch(a.html(),/id="openLedger"|id="openWhatsAppDraft"/);
});
test('money entry converts decimal text without floating point rounding',()=>{
 const a=app();assert.equal(a.run('centsInput("12,34")'),1234);assert.equal(a.run('centsInput("0.01")'),1);assert.throws(()=>a.run('centsInput("1.005")'));assert.throws(()=>a.run('centsInput("-1")'));
});

test('approved orange brand logo is used on sidebar and login',()=>{
 const a=app();
 a.run('shell("<p>demo</p>")');
 // Main CRM app loads the original brand asset in both light and dark contexts.
 const source=fs.readFileSync('source/crm/web/app.js','utf8');
 assert.match(source,/\/crm\/logo-dark\.svg/);
 assert.match(source,/\/crm\/logo-light\.svg/);
});
test('whatsapp inbox shows real messages separately from status-only records',async()=>{
 const a=app(false);
 a.run('api=async()=>({messages:[{id:"valid",peer:"358401234567",direction:"incoming",status:"received",at:"2026-11-11T15:00:00Z",message:{id:"wamid.real",type:"text",text:{body:"Hei testi"}}},{id:"pending",peer:"",status:"read",message:{}}],cursor:""})');
 await a.run('showWAInbox()');
 assert.match(a.html(),/Hei testi/);
 assert.match(a.html(),/служебных статусов без текста/);
 assert.match(a.html(),/id="waRefresh"/);
 assert.doesNotMatch(a.html(),/Ожидание сообщения/);
});
