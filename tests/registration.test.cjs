const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict');
const valid={'abc123':'ABC-123',' ABC-123 ':'ABC-123','abc 123':'ABC-123','123abc':'123-ABC','123 abc':'123-ABC','123 - abc':'123-ABC','000xyz':'000-XYZ'};
const invalid=['','ABC-1','ABC-12','AB-123','123-AB','ABCD123','ABC1234','1ABC23','АВС123','ABC/123','ABC--123','ABC-123!','ABC\n123'];
for(const file of ['source/static/js/app.js','assets/static/js/app.js','source/static/js/footer-form.js','source/crm/web/app.js']){
 const text=fs.readFileSync(file,'utf8');const start=text.indexOf('function normalizeRegistrationNumber(');const match=text.slice(start).match(/^[\s\S]*?function isValidRegistrationNumber\(value\) \{[\s\S]*?\n  \}/);assert.ok(match,file);
 const c=vm.createContext({});vm.runInContext(match[0],c);
 for(const [input,want] of Object.entries(valid)){assert.equal(c.normalizeRegistrationNumber(input),want,file+': '+input);assert.equal(c.isValidRegistrationNumber(input),true,file+': '+input)}
 for(const input of invalid)assert.equal(c.isValidRegistrationNumber(input),false,file+': '+input);
 for(const [input,want] of [['123a','123-A'],['abc1','ABC-1'],['ABC1234','ABC1234'],['1234ABC','1234ABC']])assert.equal(c.normalizeRegistrationNumber(input),want);
 console.log('PASS registration:',file);
}
