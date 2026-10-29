(function () {
 'use strict';
 document.addEventListener('DOMContentLoaded', () => {
  const form = document.querySelector('#adminSettingsForm'); if (!form) return;
  const sidebar = document.querySelector('.tabs-nav');
  const links = [...document.querySelectorAll('.tab-button')];
  const panels = [...document.querySelectorAll('[data-panel]')];
  const activeField = form.querySelector('[name="active_tab"]');
  const editable = new Set(panels.filter(p => p.closest('form') === form && !['overview','help'].includes(p.dataset.panel)).map(p => p.dataset.panel));
  let dirty = false, active = activeField.value || 'overview';
  const toolbar = document.createElement('div'); toolbar.className = 'pro-mobile-nav';
  const label = document.createElement('label'); label.textContent = 'Раздел кабинета';
  const select = document.createElement('select'); select.setAttribute('aria-label','Раздел кабинета');
  links.forEach(a => { const o = document.createElement('option'); o.value = a.href; o.textContent = a.childNodes[0].textContent.trim(); o.selected = a.classList.contains('active'); select.append(o); });
  label.append(select); toolbar.append(label); document.querySelector('.tabs-shell').before(toolbar);
  const save = document.createElement('button'); save.type='submit'; save.setAttribute('form',form.id); save.className='pro-save'; document.body.append(save);
  const status = document.createElement('span'); status.className='pro-save-status'; status.setAttribute('role','status'); save.before(status);
  function updateSave() { save.hidden = !dirty && !editable.has(active); save.dataset.dirty=String(dirty); save.textContent=dirty?'Сохранить изменения':'Сохранить настройки'; status.textContent=dirty?'Есть несохранённые изменения':''; }
  function activate(tab, push) {
   if (!panels.some(p => p.dataset.panel===tab)) return false;
   active=tab; panels.forEach(p=>p.classList.toggle('active',p.dataset.panel===tab));
   links.forEach(a=>{const selected=new URL(a.href).searchParams.get('tab')===tab; a.classList.toggle('active',selected); if(selected)a.setAttribute('aria-current','page');else a.removeAttribute('aria-current');});
   activeField.value=tab; const link=links.find(a=>a.classList.contains('active')); if(link)select.value=link.href;
   if(push){const target=new URL(location.href); target.searchParams.set('tab',tab); history.pushState(null,'',target);}
   updateSave(); return true;
  }
  document.addEventListener('click',e=>{
   const a=e.target.closest('a'); if(!a||e.defaultPrevented||e.button!==0||e.metaKey||e.ctrlKey||e.shiftKey||e.altKey||a.target)return;
   const u=new URL(a.href); if(u.origin!==location.origin||u.pathname!=='/admin'||!u.searchParams.has('tab')||[...u.searchParams.keys()].some(k=>k!=='tab'))return;
   if(activate(u.searchParams.get('tab'),true)){e.preventDefault();document.querySelector('.panel.active')?.scrollIntoView({block:'start',behavior:'auto'});}
  });
  select.addEventListener('change',()=>{const u=new URL(select.value);if(u.pathname==='/admin'&&activate(u.searchParams.get('tab'),true))return;location.assign(u);});
  addEventListener('popstate',()=>activate(new URL(location.href).searchParams.get('tab')||'overview',false));
  const markDirty=()=>{dirty=true;updateSave();}; form.addEventListener('input',markDirty);form.addEventListener('change',markDirty);
  form.addEventListener('submit',()=>{dirty=false;save.disabled=true;save.textContent='Сохраняем…';});
  addEventListener('beforeunload',e=>{if(dirty){e.preventDefault();e.returnValue='';}});
  const fields=[...form.querySelectorAll('.field,.pro-settings>label')].map((node,i)=>{
   const input=node.querySelector('input:not([type="hidden"]),textarea,select');const title=(node.querySelector('label')?.textContent||node.childNodes[0]?.textContent||'').trim();const panel=node.closest('[data-panel]');
   if(!input||!panel||!title)return null;if(!input.id)input.id='admin-setting-'+i;const fieldLabel=node.matches('label')?node:node.querySelector('label');if(fieldLabel)fieldLabel.htmlFor=input.id;
   return{node,input,title,panel:panel.dataset.panel,text:(title+' '+input.name).toLocaleLowerCase('ru')};
  }).filter(Boolean);
  function createSearch(parent){
   if(!parent)return;const wrap=document.createElement('div');wrap.className='admin-search';const input=document.createElement('input');input.type='search';input.placeholder='Найти настройку…';input.setAttribute('aria-label','Поиск по всем настройкам');input.className='pro-field-search';
   const results=document.createElement('div');results.className='admin-search-results';results.hidden=true;results.setAttribute('aria-live','polite');wrap.append(input,results);parent.append(wrap);
   input.addEventListener('input',()=>{const q=input.value.trim().toLocaleLowerCase('ru');results.replaceChildren();results.hidden=!q;if(!q)return;const found=fields.filter(f=>f.text.includes(q)).slice(0,12);if(!found.length){results.textContent='Настройки не найдены. Попробуйте другое слово.';return;}
    found.forEach(f=>{const b=document.createElement('button');b.type='button';b.textContent=f.title;b.addEventListener('click',()=>{activate(f.panel,true);results.hidden=true;input.value='';f.node.scrollIntoView({block:'center',behavior:'auto'});f.input.focus({preventScroll:true});});results.append(b);});
   });
  }
  createSearch(toolbar);if(sidebar){const slot=document.createElement('div');sidebar.prepend(slot);createSearch(slot);}activate(active,false);
 });
})();
