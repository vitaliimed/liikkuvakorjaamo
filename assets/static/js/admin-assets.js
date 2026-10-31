document.addEventListener('DOMContentLoaded',()=>{
 'use strict';const form=document.querySelector('#adminSettingsForm');if(!form)return;
 const names=['logo_url','logo_dark_url','email_logo_url','favicon_svg_url','favicon_32_url','favicon_16_url','favicon_ico_url','apple_touch_icon_url','hero_background_url','hero_person_desktop_url','areas_map_image_url','services_image_1_url','services_image_2_url','services_image_3_url','services_image_4_url','services_image_5_url','services_image_6_url','service_image_url[]','testimonial_photo_url[]'];
 form.querySelectorAll('input[type=text]').forEach(input=>{
  if(!names.includes(input.name))return;
  const group=document.createElement('div');group.className='asset-upload-control';const button=document.createElement('button');button.type='button';button.textContent='Загрузить с компьютера';
  const file=document.createElement('input');file.type='file';file.accept='.png,.jpg,.jpeg,.webp,.svg,.ico';file.tabIndex=-1;
  const status=document.createElement('span');status.className='asset-upload-status';status.setAttribute('role','status');group.append(button,file,status);input.after(group);
  button.addEventListener('click',()=>file.click());file.addEventListener('change',async()=>{
   const image=file.files[0];if(!image)return;status.dataset.error='false';
   if(image.size>8*1024*1024){status.textContent='Файл больше 8 МБ. Сожмите изображение.';status.dataset.error='true';file.value='';return;}
   const body=new FormData();body.append('asset',image);body.append('access_csrf',form.querySelector('[name=access_csrf]').value);button.disabled=true;status.textContent='Загрузка…';
   try{const response=await fetch('/admin/assets/upload',{method:'POST',body,credentials:'same-origin',headers:{Accept:'application/json'}});const data=await response.json();if(!response.ok||!data.ok)throw new Error(data.message||'Не удалось загрузить файл. Обновите страницу и повторите.');
    input.value=data.url;input.dispatchEvent(new Event('input',{bubbles:true}));input.dispatchEvent(new Event('change',{bubbles:true}));status.textContent='Загружено. Сохраните изменения, чтобы применить к сайту.';
   }catch(error){status.dataset.error='true';status.textContent=error.message||'Ошибка соединения. Повторите загрузку.';}finally{button.disabled=false;file.value='';}
  });
 });
});
