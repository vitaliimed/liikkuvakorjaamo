'use strict';
// CRM account recovery UI. Loaded before app.js; does not create any user.
function crmGeneratePassword(){
 const alphabet='ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%&*?';
 const limit=Math.floor(256/alphabet.length)*alphabet.length;
 const out=[];
 while(out.length<20){
  const bytes=crypto.getRandomValues(new Uint8Array(32));
  for(const b of bytes){if(b<limit)out.push(alphabet[b%alphabet.length]);if(out.length===20)break}
 }
 return out.join('');
}
function crmPasswordTools(){
 return '<div class="password-tools" data-password-tools>'+
 '<button type="button" data-generate-password>Сгенерировать пароль</button>'+
 '<button type="button" data-toggle-password>Показать</button>'+
 '<span class="password-strength" data-password-strength>Любой непустой пароль до 72 байт; рекомендуем 12+ символов.</span></div>';
}
function crmAttachPasswordTools(form){
 if(!form)return;
 const input=form.querySelector('input[name="password"]');
 const tools=form.querySelector('[data-password-tools]');
 if(!input||!tools)return;
 const strength=tools.querySelector('[data-password-strength]');
 const update=()=>{
  const p=input.value||'';
  strength.textContent=!p?'Любой непустой пароль до 72 байт. Рекомендуем генератор.':
   p.trim().length===0?'Пароль не должен состоять только из пробелов.':
   p.length<12?'Короткий пароль допустим, но его легко угадать. Рекомендуем 12+ символов.':
   'Длина пароля: '+p.length+' символов. Храните его в менеджере паролей.';
 };
 tools.querySelector('[data-generate-password]').onclick=()=>{
  input.value=crmGeneratePassword();input.type='text';
  tools.querySelector('[data-toggle-password]').textContent='Скрыть';
  update();input.focus();input.select();
 };
 tools.querySelector('[data-toggle-password]').onclick=e=>{
  input.type=input.type==='password'?'text':'password';
  e.target.textContent=input.type==='password'?'Показать':'Скрыть';
 };
 input.addEventListener('input',update);update();
}
function crmForgotPassword(prefill){
 $('#app').innerHTML='<form class="login" id="forgot-form">'+
 '<div class="logo"><img src="/crm/logo-light.svg" alt="Liikkuva auto korjaamo"></div>'+
 '<h2>Восстановление пароля</h2><p class="muted">Введите адрес электронной почты, используемый как логин CRM. Если учётная запись активна, мы отправим на этот адрес одноразовую ссылку для установки нового пароля.</p>'+
 '<label>Email — логин CRM<input name="email" type="email" autocomplete="username" value="'+esc(prefill||'')+'" required></label>'+
 '<div class="error" id="forgot-error" role="alert"></div><p id="forgot-result" class="muted" role="status"></p>'+
 '<button class="primary" type="submit">Отправить ссылку</button>'+
 '<button class="text-action" type="button" id="forgot-back">Назад ко входу</button></form>';
 $('#forgot-back').onclick=()=>login();
 $('#forgot-form').onsubmit=async e=>{
  e.preventDefault();const b=e.target.querySelector('[type=submit]');
  b.disabled=true;$('#forgot-error').textContent='';$('#forgot-result').textContent='';
  try{
   const result=await api('password/forgot','POST',{email:new FormData(e.target).get('email')});
   $('#forgot-result').textContent=result.message;
  }catch(error){$('#forgot-error').textContent=error.message}
  finally{b.disabled=false}
 };
}
async function crmResetPassword(token){
 $('#app').innerHTML='<div class="login"><h2>Проверяем ссылку…</h2><p class="muted">Восстановление Liikkuva CRM</p></div>';
 let info;
 try{info=await api('password/verify','POST',{token})}
 catch(error){
  $('#app').innerHTML='<div class="login"><h2>Ссылка недействительна</h2><p class="error">'+esc(error.message)+'</p>'+
   '<button class="primary" type="button" id="ask-new">Запросить новую ссылку</button></div>';
  $('#ask-new').onclick=()=>crmForgotPassword('');return;
 }
 $('#app').innerHTML='<form class="login" id="reset-form">'+
  '<div class="logo"><img src="/crm/logo-light.svg" alt="Liikkuva auto korjaamo"></div>'+
  '<h2>Новый пароль CRM</h2>'+
  '<p class="muted">Это зарегистрированная учётная запись, подтверждённая ссылкой из письма.</p>'+
  '<label>Email<input type="email" value="'+esc(info.email)+'" autocomplete="username" readonly aria-readonly="true"></label>'+
  '<label>Новый пароль<input name="password" type="password" autocomplete="new-password" required></label>'+
  crmPasswordTools()+
  '<div class="error" id="reset-error" role="alert"></div>'+
  '<button class="primary" type="submit">Установить пароль</button>'+
  '<small>Ссылка действует 30 минут и только один раз. После смены старые сеансы будут закрыты.</small></form>';
 crmAttachPasswordTools($('#reset-form'));
 $('#reset-form').onsubmit=async e=>{
  e.preventDefault();const b=e.target.querySelector('[type=submit]');
  b.disabled=true;$('#reset-error').textContent='';
  try{
   await api('password/reset','POST',{token,password:new FormData(e.target).get('password')});
   if(typeof history!=='undefined'&&history.replaceState)history.replaceState(null,'','/crm/');
   $('#app').innerHTML='<div class="login"><h2>Пароль изменён</h2>'+
    '<p class="muted">Теперь можно войти в CRM с новым паролем.</p>'+
    '<button class="primary" type="button" id="reset-login">Войти в CRM</button></div>';
   $('#reset-login').onclick=()=>login();
  }catch(error){$('#reset-error').textContent=error.message}
  finally{b.disabled=false}
 };
}
