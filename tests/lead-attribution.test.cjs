const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('fs');
const vm = require('vm');
const source = fs.readFileSync('source/static/js/lead-attribution.js','utf8');
test('first landing survives navigation, consent revocation clears identifiers', () => {
  const data=new Map(), storage={getItem:k=>data.get(k),setItem:(k,v)=>data.set(k,v),removeItem:k=>data.delete(k)};
  let consent={analytics:true,marketing:true};
  function visit(pathname,search) {
    const window={liikkuvaConsentState:()=>consent,addEventListener(){}};
    const document={referrer:'https://google.fi/search?private=secret',addEventListener(){},querySelector(){return null;}};
    vm.runInNewContext(source,{window,document,sessionStorage:storage,URL,URLSearchParams,location:{origin:'https://liikkuvakorjaamo.fi',pathname,search}});
    const f=new Map();window.liikkuvaSetLeadAttribution(f);return f;
  }
  let result=visit('/palvelualue','?utm_source=google&gclid=ABC&email=private');
  assert.equal(result.get('gclid'),'ABC');assert.equal(result.get('referer'),'https://google.fi');
  result=visit('/hinnasto','');assert.equal(result.get('landing_page'),'/palvelualue');
  assert.equal(result.get('form_page'),'/hinnasto');assert.equal(result.get('utm_source'),'google');
  consent={analytics:true,marketing:false};result=visit('/hinnasto','');assert.equal(result.get('gclid'),'');
  consent={};result=visit('/en','');assert.equal(result.get('landing_page'),'');assert.equal(data.size,0);
});
