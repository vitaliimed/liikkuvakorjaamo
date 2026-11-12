const fs=require('node:fs');
const vm=require('node:vm');
const assert=require('node:assert/strict');

const secret='a'.repeat(64);
const props={
  WEBHOOK_TOKEN:secret,
  SITE_DOMAIN:'liikkuvakorjaamo.fi',
  SPREADSHEET_ID:'14HHvj7skyve3K9ugFFgdlobD9De-2BICFdkSZdrVHKU'
};
const ctx={
  PropertiesService:{getScriptProperties:()=>({
    getProperty:k=>props[k]||null,
    setProperties:o=>Object.assign(props,o)
  })},
  ContentService:{MimeType:{JSON:'json'},createTextOutput:s=>({setMimeType:()=>JSON.parse(s)})},
  console:{log:()=>{},error:()=>{}}
};
const source=fs.readFileSync('apps-script/Code.gs','utf8');
vm.createContext(ctx);
vm.runInContext(source,ctx);

assert.equal(vm.runInContext('HEADERS.length',ctx),23,'monthly headers must have a channel column');
assert.equal(vm.runInContext('SUMMARY_HEADERS.length',ctx),13,'income summary must include ROAS CR CPL CPA');
assert.equal(vm.runInContext('SHEET_ID',ctx),'14HHvj7skyve3K9ugFFgdlobD9De-2BICFdkSZdrVHKU');
assert.ok(!source.includes('SpreadsheetApp.create('),'never create a second Google Sheet');
assert.ok(source.includes('function setupExistingLiikkuva'));
assert.ok(source.includes('SpreadsheetApp.openById(SHEET_ID)'),'Web Apps must use openById, not getActiveSpreadsheet');
assert.ok(source.includes('script_token'),'webhook must require a private token');
assert.ok(source.includes("l.submission_id"),'lead retries must be deduplicated');
assert.ok(source.includes('tryLock(20000)'),'parallel webhook writes must be serialized');

for(const formula of [
  'SUMIF(', 'COUNTIF(', 'COUNTIFS(', 'SUMIFS(',
  'H15/D15','F15/E15','D15/F15','D15/G15'
]){
  assert.ok(source.includes(formula),'missing marketing formula '+formula);
}
for(const value of [
  ['cpc','','Реклама'],['paid_social','','Реклама'],
  ['organic','','Органика'],['','google_ads','Реклама'],
  ['','google','Не определён'],['','','Не определён']
]){
  assert.equal(ctx.channel_(value[0],value[1]),value[2],JSON.stringify(value));
}
for(const dangerous of ['=IMPORTXML("x")','+123','@SUM(A1:A2)',' -4']){
  assert.equal(ctx.safe_(dangerous),"'"+dangerous);
}
assert.equal(ctx.safe_('Normal text'),'Normal text');
assert.equal(ctx.safe_(null),'');
assert.equal(ctx.doPost({postData:{contents:'{}'}}).error,'Unauthorized');
assert.equal(ctx.doPost({postData:{contents:JSON.stringify({script_token:secret,site:'autohelp.fi'})}}).error,'Wrong site');
assert.equal(ctx.doPost({postData:{contents:'x'.repeat(100001)}}).error,'Invalid payload');
assert.equal(ctx.doGet().site,'liikkuvakorjaamo.fi');
console.log('Liikkuva Apps Script: existing Sheet, data integrity, marketing metrics, UTM attribution, auth, dedupe and spreadsheet-injection checks passed.');
