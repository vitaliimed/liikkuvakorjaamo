const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const crypto=require('node:crypto').webcrypto;

const resetScript=fs.readFileSync('source/crm/web/password-reset.js','utf8');
const crmScript=fs.readFileSync('source/crm/web/app.js','utf8');
const html=fs.readFileSync('source/crm/web/index.html','utf8');

test('registered-email reset UI is loaded before the CRM app',()=>{
 const i=html.indexOf('/crm/password-reset.js');
 const j=html.indexOf('/crm/app.js');
 assert.ok(i>=0&&j>i);
 assert.match(crmScript,/Забыли пароль\?/);
 assert.match(crmScript,/crmForgotPassword/);
 assert.match(crmScript,/crmResetPassword/);
 assert.match(crmScript,/location\.hash/);
 assert.match(crmScript,/crmAttachPasswordTools/);
 assert.match(crmScript,/crmPasswordTools/);
});

test('password generator yields strong distinct 20-character values',()=>{
 const ctx=vm.createContext({crypto,console});
 vm.runInContext(resetScript,ctx);
 const seen=new Set();
 for(let i=0;i<100;i++){
  const password=vm.runInContext('crmGeneratePassword()',ctx);
  assert.equal(password.length,20);
  assert.match(password,/^[A-Za-z2-9!@#$%&*?]+$/);
  seen.add(password);
 }
 assert.equal(seen.size,100);
});

test('verified password link shows server-registered email as read-only',async()=>{
 const nodes=new Map();
 const createNode=()=>({innerHTML:'',value:'',style:{},querySelector:()=>null,addEventListener(){},focus(){},select(){}});
 const find=s=>{if(!nodes.has(s))nodes.set(s,createNode());return nodes.get(s)};
 const ctx=vm.createContext({
  crypto,console,
  document:{querySelector:find},
  api:async(path,method,body)=>{
    assert.equal(method,'POST');
    assert.equal(path,'password/verify');
    assert.equal(body.token,'verified');
    return {email:'owner@example.test'};
  },
  esc:x=>String(x).replaceAll('&','&amp;').replaceAll('"','&quot;'),
 });
 vm.runInContext('const $ = document.querySelector.bind(document);',ctx);
 vm.runInContext(resetScript,ctx);
 await vm.runInContext('crmResetPassword("verified")',ctx);
 assert.match(find('#app').innerHTML,/owner@example\.test/);
 assert.match(find('#app').innerHTML,/readonly/);
 assert.match(find('#app').innerHTML,/name="password"/);
 assert.match(find('#app').innerHTML,/Сгенерировать пароль/);
});
