const fs=require('node:fs'),vm=require('node:vm'),assert=require('node:assert/strict');
for(const file of ['source/static/js/app.js','assets/static/js/app.js','source/static/js/footer-form.js']){
 const source=fs.readFileSync(file,'utf8');const code=source.slice(0,source.indexOf('\n}',source.indexOf('function maskRegistrationNumber'))+2);const c=vm.createContext({});vm.runInContext(code,c);
 for(const [input,want] of [['222222222222','222-'],['kkkkkkkkkkkk','KKK-'],['abc123456','ABC-123'],['123abcdef','123-ABC'],['abc','ABC-'],['123','123-'],['АВС123','123-'],['abc12','ABC-12'],['','']])assert.equal(c.maskRegistrationNumber(input),want,file);
}
console.log('PASS strict live registration masks');
