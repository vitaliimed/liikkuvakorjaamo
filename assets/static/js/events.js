/* Public events contain no form contents, phone, email, address or coordinates. */
(function(){'use strict';
 const memory=new Set();
 const consent=()=>typeof window.liikkuvaConsentState==='function'?window.liikkuvaConsentState():{};
 const config=()=>window.LIIKKUVA_TRACKING||{};
 window.liikkuvaSafeLocation=function(marketing){const u=new URL(location.origin+location.pathname);if(marketing){const q=new URLSearchParams(location.search);['gclid','gbraid','wbraid'].forEach(k=>{const v=q.get(k)||'';if(/^[A-Za-z0-9_-]{1,240}$/.test(v))u.searchParams.set(k,v);});}return u.href;};
 function once(id,vendor,fn){const key='liikkuva_receipt_'+id+'_'+vendor;if(memory.has(key))return;try{if(sessionStorage.getItem(key))return;}catch(e){}memory.add(key);try{sessionStorage.setItem(key,'1');}catch(e){}fn();}
 function event(name,params){if(!consent().analytics||typeof window.gtag!=='function')return;window.gtag('event',name,Object.assign({site_domain:'liikkuvakorjaamo.fi',site_brand:'Liikkuva Auto Korjaamo',page_language:document.documentElement.lang,page_location:window.liikkuvaSafeLocation(!!consent().marketing)},params||{}));}
 window.liikkuvaSavedLead=function(id){if(!/^[A-Za-z0-9_-]{32,100}$/.test(id||''))return;const c=consent(),t=config();
  if(c.analytics&&t.ga4MeasurementId&&window.gtag)once(id,'ga4',()=>event('generate_lead',{event_id:id,lead_source:'website_form'}));
  if(!c.marketing)return;
  if(t.googleAdsTagId&&t.googleAdsLeadConversionLabel&&window.gtag)once(id,'ads',()=>window.gtag('event','conversion',{send_to:t.googleAdsTagId+'/'+t.googleAdsLeadConversionLabel,transaction_id:id}));
  if(t.metaPixelId&&window.fbq)once(id,'meta',()=>window.fbq('track','Lead',{}, {eventID:id}));
  if(t.tiktokPixelId&&window.ttq)once(id,'tiktok',()=>window.ttq.track('SubmitForm',{}, {event_id:id}));
  if(t.linkedinPartnerId&&t.linkedinLeadConversionID&&window.lintrk)once(id,'linkedin',()=>window.lintrk('track',{conversion_id:Number(t.linkedinLeadConversionID)}));
 };
 document.addEventListener('DOMContentLoaded',()=>{
  document.addEventListener('invalid',e=>{const name=e.target.name;if(['name','phone','email','address','car','message'].includes(name))event('lead_validation_error',{field_name:name});},true);
  // Standard app.js handles form, telephone and language events. These are complementary.
  document.addEventListener('click',e=>{const a=e.target.closest('a');if(!a)return;const href=a.getAttribute('href')||'';if(href.startsWith('mailto:'))event('email_click');else if(/wa\.me|whatsapp/.test(href))event('whatsapp_click');else if(a.closest('nav'))event('navigation_click',{destination:href.startsWith('#')?href.slice(1):'page'});});
  let levels=new Set();addEventListener('scroll',()=>{const height=document.documentElement.scrollHeight-innerHeight;if(height<=0)return;const n=100*scrollY/height;[25,50,75,90].forEach(v=>{if(n>=v&&!levels.has(v)&&consent().analytics){levels.add(v);event('scroll_depth',{percent_scrolled:v});}});},{passive:true});
  const prices=document.querySelector('#pricing,#prices,#hinnat,.pricing-section');if(prices&&window.IntersectionObserver){const io=new IntersectionObserver(entries=>{if(entries.some(x=>x.isIntersecting)&&consent().analytics){event('view_pricing');io.disconnect();}},{threshold:.35});io.observe(prices);}
 });
})();
