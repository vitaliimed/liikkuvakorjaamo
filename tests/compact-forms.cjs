/* Two real browser form instances. All acknowledgements, map and geocoding calls
   are local doubles; no request reaches a lead handler or external provider. */
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
module.exports=async function checkCompactForms(browser,base,artifactDir){
 const png=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+a2ioAAAAASUVORK5CYII=','base64');
 for(const width of [390,1366]){
  const context=await browser.newContext({viewport:{width,height:width===390?844:768},permissions:['geolocation'],geolocation:{latitude:60.17,longitude:24.94},isMobile:width===390,hasTouch:width===390});
  await context.addInitScript(()=>{
   window.__qaMaps={};window.__qaMapMoves=[];
   window.L={map:(element,options)=>{
    const map={id:element.id,zoom:options.zoom||0,handlers:{},on(name,callback){this.handlers[name]=callback;return this;},setView(center,zoom){this.zoom=zoom;window.__qaMapMoves.push({id:this.id,center,zoom});return this;},getZoom(){return this.zoom;},invalidateSize(){return this;},fitBounds(){throw Error('Overview map must not be reset with fitBounds');}};
    window.__qaMaps[element.id]=map;return map;
   },tileLayer:()=>({addTo:()=>{}}),marker:(position)=>({position,addTo(){return this;},on(){return this;},setLatLng(value){this.position=value;return this;},getLatLng(){return{lat:this.position[0],lng:this.position[1]};}})};
  });
  const submissions=[];
  let deferGeocode=false,releaseGeocode,geocodeSeen;
  await context.route('**/*',async route=>{
   const request=route.request(),url=new URL(request.url());
   if(url.hostname==='nominatim.openstreetmap.org'){
    const result={status:200,contentType:'application/json',body:JSON.stringify({display_name:'Test location from map'})};
    if(deferGeocode){releaseGeocode=()=>route.fulfill(result);geocodeSeen();return;}
    return route.fulfill(result);
   }
   if(request.url().startsWith(base+'/lead')&&request.method()==='POST'){
    const body=request.postDataBuffer().toString('utf8');submissions.push(body);const id=(body.includes('Bottom Client')?'b':'a').repeat(43);
    return route.fulfill({status:200,contentType:'application/json',body:JSON.stringify({ok:true,submission_id:id,message:'Synthetic request received'})});
   }
   if(request.url().startsWith(base+'/')||request.url().startsWith('data:'))return route.continue();
   return route.abort();
  });
  const page=await context.newPage();page.setDefaultTimeout(12000);const errors=[];page.on('pageerror',e=>errors.push(e.message));
  await page.goto(base+'/en');
  const necessary=page.getByRole('button',{name:'Only necessary'});if(await necessary.isVisible())await necessary.click();
  await page.evaluate(()=>{window.__qaEvents=[];window.liikkuvaConsentState=()=>({analytics:true,marketing:false});window.gtag=(...values)=>window.__qaEvents.push(values);});
  const top=page.locator('#leadForm'),bottom=page.locator('#leadFormBottom');
  await top.locator('[name=name]').fill('Top Client');await top.locator('[name=phone]').fill('0401234567');
  assert.equal(await top.locator('[name=phone]').inputValue(),'+358 40 123 45 67');
  await top.locator('[name=phone]').evaluate(input=>{const clipboardData=new DataTransfer();clipboardData.setData('text','+358 (0) 40 123 45 67');input.dispatchEvent(new ClipboardEvent('paste',{clipboardData,bubbles:true,cancelable:true}));});
  assert.equal(await top.locator('[name=phone]').inputValue(),'+358 40 123 45 67','Phone paste lost digits');
  await top.locator('[name=email]').fill('top-private@example.test');await top.locator('[name=car]').fill('abc12');
  assert.equal(await top.locator('[name=car]').inputValue(),'ABC-12');assert.equal(await top.locator('[name=car]').evaluate(x=>x.checkValidity()),false);
  await top.locator('[name=car]').fill('abc123');assert.equal(await top.locator('[name=car]').inputValue(),'ABC-123');
  await top.locator('[name=car]').fill('');await top.locator('[name=car]').pressSequentially('abc123');
  assert.equal(await top.locator('[name=car]').inputValue(),'ABC-123','Typing the plate lost a character');
  await top.locator('#useLocationBtn').click();await page.waitForFunction(()=>document.getElementById('latitude').value==='60.17');
  await page.waitForFunction(()=>document.getElementById('address').value==='Test location from map');await top.locator('[name=address]').fill('Top Address 1, Helsinki');
  const geocodePending=new Promise(resolve=>{geocodeSeen=resolve;});deferGeocode=true;
  await top.locator('#useLocationBtn').click();await geocodePending;
  await top.locator('[name=address]').fill('Top Address 1, manually corrected');await releaseGeocode();deferGeocode=false;
  await page.waitForFunction(()=>!document.getElementById('useLocationBtn').disabled);
  assert.equal(await top.locator('[name=address]').inputValue(),'Top Address 1, manually corrected','A late geocoder response overwrote a manual address');
  await top.locator('[name=message]').fill('Private top message');await top.locator('[name=photos]').setInputFiles({name:'top-photo.png',mimeType:'image/png',buffer:png});
  assert.equal(await top.locator('[data-photo-count]').textContent(),'1 / 3');assert.equal(await bottom.locator('[name=phone]').inputValue(),'');
  await bottom.locator('[name=name]').fill('Bottom Client');await bottom.locator('[name=phone]').fill('0509876543');await bottom.locator('[name=email]').fill('bottom-private@example.test');await bottom.locator('[name=car]').fill('xyz987');
  await bottom.locator('#bottom-pickMapBtn').click();await page.waitForFunction(()=>!!window.__qaMaps['bottom-mapPickerCanvas']);
  assert(await page.locator('#bottom-mapPickerModal').isVisible());assert.equal(await page.locator('#mapPickerModal').isVisible(),false);
  await page.evaluate(()=>window.__qaMaps['bottom-mapPickerCanvas'].handlers.click({latlng:{lat:60.2,lng:24.9}}));
  await page.locator('#bottom-confirmMapPoint').click();await page.waitForFunction(()=>document.getElementById('bottom-latitude').value==='60.2');
  assert.equal(await top.locator('[name=latitude]').inputValue(),'60.17','Bottom map modified the top form');
  await bottom.locator('[name=address]').fill('Bottom Address 2, Helsinki');await bottom.locator('[name=message]').fill('Private bottom message');await bottom.locator('[name=photos]').setInputFiles({name:'bottom-photo.png',mimeType:'image/png',buffer:png});
  await page.locator('#areas').scrollIntoViewIfNeeded();await page.waitForFunction(()=>!!window.__qaMaps.areasLiveMap);
  assert.equal(await page.evaluate(()=>window.__qaMaps.areasLiveMap.getZoom()),12);assert.deepEqual(await page.evaluate(()=>window.__qaMapMoves.find(x=>x.id==='areasLiveMap').center),[60.1699,24.9384]);
  await page.evaluate(()=>window.__qaMaps.areasLiveMap.zoom=14);await page.setViewportSize({width:width+20,height:width===390?844:768});await page.waitForTimeout(180);
  assert.equal(await page.evaluate(()=>window.__qaMaps.areasLiveMap.getZoom()),14,'Window resize reset the map view');await page.setViewportSize({width,height:width===390?844:768});
  await top.locator('button[type=submit]').click();await page.waitForFunction(()=>document.getElementById('formMessage').classList.contains('success'));
  assert.equal(submissions.length,1);assert(submissions[0].includes('Top Client')&&submissions[0].includes('ABC-123')&&submissions[0].includes('Top Address 1')&&submissions[0].includes('top-photo.png'));assert(!submissions[0].includes('Bottom Client'));
  assert.equal(await bottom.locator('[name=name]').inputValue(),'Bottom Client');assert.equal(await bottom.locator('[data-photo-count]').textContent(),'1 / 3');
  await bottom.locator('button[type=submit]').click();await page.waitForFunction(()=>document.getElementById('bottom-formMessage').classList.contains('success'));
  assert.equal(submissions.length,2);assert(submissions[1].includes('Bottom Client')&&submissions[1].includes('XYZ-987')&&submissions[1].includes('Bottom Address 2')&&submissions[1].includes('bottom-photo.png'),JSON.stringify({name:submissions[1].includes('Bottom Client'),plate:submissions[1].includes('XYZ-987'),address:submissions[1].includes('Bottom Address 2'),photo:submissions[1].includes('bottom-photo.png')}));assert(!submissions[1].includes('Top Client'));
  await page.evaluate(()=>{window.liikkuvaSavedLead('a'.repeat(43),'hero');window.liikkuvaSavedLead('b'.repeat(43),'bottom');});
  const events=await page.evaluate(()=>window.__qaEvents);const leads=events.filter(x=>x[1]==='generate_lead');assert.equal(leads.length,2);assert.deepEqual(leads.map(x=>x[2].form_position),['hero','bottom']);
  assert(!/Top Client|Bottom Client|private@example|Private top|Private bottom|Top Address|Bottom Address|top-photo|bottom-photo|60\.17/.test(JSON.stringify(events)),'Private contents appeared in analytics');
  assert.deepEqual(errors,[],'Unexpected browser error in independent forms');
  await page.goto(base+'/');await page.screenshot({path:path.join(artifactDir,`compact-fi-${width}.png`)});
  await context.close();
 }
 console.log('PASS: two independent forms, masks, photos, geolocation, map selection, Helsinki view, resize preservation and private GA4 events at 390 / 1366 px.');
};
