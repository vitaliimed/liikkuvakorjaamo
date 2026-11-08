const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict');
const src=fs.readFileSync('crm/web/app.js','utf8').split('(async()=>{if(demo)')[0];
const ctx=vm.createContext({Intl,Date,URLSearchParams,location:{search:''},crypto:require('node:crypto').webcrypto,document:{querySelector:()=>null}});vm.runInContext(src,ctx);
assert.equal(vm.runInContext("helsinkiISO('2026-11-08T15:30')",ctx),'2026-11-08T13:30:00.000Z');
assert.equal(vm.runInContext("helsinkiISO('2026-07-08T15:30')",ctx),'2026-07-08T12:30:00.000Z');
assert.throws(()=>vm.runInContext("helsinkiISO('2026-03-29T03:30')",ctx));
assert.throws(()=>vm.runInContext("helsinkiISO('2026-10-25T03:30')",ctx));
assert.equal(vm.runInContext("esc('<img src=x onerror=alert(1)>')",ctx),'&lt;img src=x onerror=alert(1)&gt;');
console.log('PASS: winter/summer Helsinki time, DST gap/fold rejection, HTML escaping');
