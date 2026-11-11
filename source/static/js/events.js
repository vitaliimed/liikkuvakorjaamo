/* Measurement describes actions only. Never pass contacts, messages, files or coordinates. */
(function(){'use strict';
 const memory=new Set();
 const consent=()=>typeof window.liikkuvaConsentState==='function'?window.liikkuvaConsentState():{};
 const config=()=>window.LIIKKUVA_TRACKING||{};
 const enabled=name=>!config().events||config().events[name]===true;
 window.liikkuvaSafeLocation=function(marketing){const u=new URL(location.origin+location.pathname);if(marketing){const q=new URLSearchParams(location.search);['gclid','gbraid','wbraid'].forEach(k=>{const v=q.get(k)||'';if(/^[A-Za-z0-9_-]{1,240}$/.test(v))u.searchParams.set(k,v);});}return u.href;};
 function once(id,vendor,fn){const key='liikkuva_receipt_'+id+'_'+vendor;if(memory.has(key))return;try{if(sessionStorage.getItem(key))return;}catch(e){}memory.add(key);try{sessionStorage.setItem(key,'1');}catch(e){}fn();}
 function safeParams(params){const clean={};const keys=['form_position','theme','field_name','destination','action','error_type','service_id','faq_index','percent_scrolled','photo_count','has_phone','has_email','has_photos','has_location','http_status'];
  keys.forEach(key=>{const value=(params||{})[key];if(typeof value==='boolean'||(typeof value==='number'&&Number.isFinite(value)))clean[key]=value;else if(typeof value==='string'&&/^[a-zA-Z0-9_-]{1,50}$/.test(value))clean[key]=value;});return clean;}
 function event(name,params){if(!enabled(name)||!config().ga4MeasurementId||!consent().analytics||typeof window.gtag!=='function')return false;
  const payload=Object.assign({send_to:config().ga4MeasurementId,site_domain:'liikkuvakorjaamo.fi',site_brand:'Liikkuva',page_language:document.documentElement.lang,page_location:window.liikkuvaSafeLocation(!!consent().marketing)},safeParams(params));
  if(name==='generate_lead'&&/^[A-Za-z0-9_-]{32,100}$/.test(params?.event_id||'')){payload.event_id=params.event_id;payload.lead_source='website_form';}
  window.gtag('event',name,payload);return true;
 }
 window.liikkuvaTrackEvent=event;
 window.liikkuvaSavedLead=function(id,position){if(!/^[A-Za-z0-9_-]{32,100}$/.test(id||''))return;const c=consent(),t=config();
  if(c.analytics&&t.ga4MeasurementId&&window.gtag&&enabled('generate_lead'))once(id,'ga4',()=>event('generate_lead',{event_id:id,form_position:position}));
  if(!c.marketing)return;
  if(t.googleAdsTagId&&t.googleAdsLeadConversionLabel&&window.gtag)once(id,'ads',()=>window.gtag('event','conversion',{send_to:t.googleAdsTagId+'/'+t.googleAdsLeadConversionLabel,transaction_id:id}));
  if(t.metaPixelId&&window.fbq)once(id,'meta',()=>window.fbq('track','Lead',{}, {eventID:id}));
  if(t.tiktokPixelId&&window.ttq)once(id,'tiktok',()=>window.ttq.track('SubmitForm',{}, {event_id:id}));
  if(t.linkedinPartnerId&&t.linkedinLeadConversionID&&window.lintrk)once(id,'linkedin',()=>window.lintrk('track',{conversion_id:Number(t.linkedinLeadConversionID)}));
 };
 document.addEventListener('DOMContentLoaded',()=>{
  document.addEventListener('invalid',e=>{const name=e.target.name;if(['name','phone','email','address','car','message'].includes(name))event('lead_validation_error',{field_name:name,form_position:e.target.closest('form')?.dataset.formPosition});},true);
  document.addEventListener('click',e=>{const a=e.target.closest('a');if(!a)return;const href=a.getAttribute('href')||'';
   if(href.startsWith('mailto:'))event('email_click');else if(/wa\.me|whatsapp/.test(href))event('whatsapp_click');
   else if(href.startsWith('tel:'))event('phone_click');
   else if(a.closest('.languages'))event('language_switch');
   else if(/^(\/en|\/)?#lead-form$/.test(href)){event('lead_form_open',{destination:a.closest('header')?'header':a.closest('.service-entry')?'service':a.closest('.mobile-cta')?'mobile':'page'});}
   else if(a.closest('nav,.resource-links,.footer-links'))event('navigation_click',{destination:/^#[a-z-]+$/.test(href)?href.slice(1):'page'});
   if(a.matches('[data-service]'))event('service_select',{service_id:[...document.querySelectorAll('[data-service]')].indexOf(a)+1});
  });
  document.querySelectorAll('.faq-list details').forEach((node,i)=>node.addEventListener('toggle',()=>{if(node.open)event('faq_open',{faq_index:i+1});}));
  let levels=new Set();addEventListener('scroll',()=>{const height=document.documentElement.scrollHeight-innerHeight;if(height<=0)return;const n=100*scrollY/height;[25,50,75,90].forEach(v=>{if(n>=v&&!levels.has(v)&&event('scroll_depth',{percent_scrolled:v}))levels.add(v);});},{passive:true});
  if(window.IntersectionObserver){
   const watch=(selector,name)=>{const element=typeof selector==='string'?document.querySelector(selector):selector;if(!element)return;let seen=false;const record=()=>{if(!seen&&event(name,{form_position:element.dataset.formPosition})){seen=true;io.disconnect();}};const io=new IntersectionObserver(entries=>{if(entries.some(x=>x.isIntersecting))record();},{threshold:.2});io.observe(element);document.addEventListener('liikkuva:consent-updated',()=>{const b=element.getBoundingClientRect();if(b.top<innerHeight&&b.bottom>0)record();});};
   watch('#pricing,#prices,#hinnat,.pricing-section','view_pricing');watch('#areas','view_service_area');document.querySelectorAll('.request-card[data-form-position]').forEach(element=>watch(element,'lead_form_view'));
  }
 });
})();
