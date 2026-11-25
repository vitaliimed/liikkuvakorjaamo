(() => {
  const panel=document.querySelector('[data-panel=seo]');
  if (!panel) return;
  const entries=Array.from(panel.querySelectorAll('details')).filter(el=>el.querySelector('[name=service_page_path]'));
  const field=(entry,key)=>entry.querySelector('[name=service_page_'+key+']');
  function snippet(entry) {
    const title=field(entry,'title').value, text=field(entry,'description').value;
    entry.querySelector('[data-snippet]').textContent = title+' ('+title.length+') · '+text+' ('+text.length+')';
  }
  for(const entry of entries) {
    snippet(entry);
    entry.addEventListener('input',()=>snippet(entry));
    entry.querySelector('[data-seo-copy]')?.addEventListener('click',()=>{
      const draft=entries[entries.length-1];
      if(field(draft,'path').value && !confirm('Заменить структуру новой страницы?')) return;
      field(draft,'body').value=field(entry,'body').value.split('\n---\n').map(b=>b.split('\n')[0]+'\n').join('\n---\n');
      for(const key of ['faq','reviews','kind']) field(draft,key).value=field(entry,key).value;
      field(draft,'status').value='draft';
      draft.hidden=false;draft.open=true;field(draft,'path').focus();
    });
    entry.querySelector('[data-seo-history]')?.addEventListener('click',async()=>{
      const target=entry.querySelector('[data-versions]');
      target.textContent='Загружаем историю…';
      try {
        const res=await fetch('/admin/seo/history?path='+encodeURIComponent(field(entry,'path').value));
        if(!res.ok) throw new Error('История недоступна');
        const versions=await res.json();
        target.textContent=versions.length?'Выберите версию для редактирования.':'Пока нет предыдущих версий.';
        for(const v of versions) {
          const b=document.createElement('button');
          b.type='button';b.textContent=new Date(v.SavedAt).toLocaleString('ru-RU');
          b.onclick=()=>{
            const p=v.Page;
            const keys={title:'Title',heading:'Heading',description:'Description',intro:'Intro',ogtitle:'OGTitle',ogdesc:'OGDescription',image:'Image',alt:'ImageAlt',cta:'CTA',keywords:'Keywords',kind:'Kind',canonical:'CanonicalPath',translation:'TranslationPath'};
            for(const [k,prop] of Object.entries(keys)) field(entry,k).value=p[prop]||'';
            field(entry,'body').value=(p.Blocks||[]).map(b=>b.Title+'\n'+b.Text).join('\n---\n');
            field(entry,'faq').value=p.HideFAQ?'hide':'show';field(entry,'reviews').value=p.HideReviews?'hide':'show';
            field(entry,'robots').value=p.NoIndex?'noindex':'index';
            field(entry,'status').value=p.Published?'published':'draft';
            snippet(entry);target.textContent='Версия восстановлена в редакторе. Сохраните настройки для применения.';
          };
          target.append(b);
        }
      } catch {target.textContent='Не удалось загрузить историю изменений.';}
    });
  }
  document.getElementById('seoFilter')?.addEventListener('input',e=>{
    const query=e.target.value.toLocaleLowerCase().trim();
    for(const entry of entries.slice(0,-1))entry.hidden=!['path','title','keywords','kind','status'].some(k=>field(entry,k).value.toLocaleLowerCase().includes(query));
  });
})();
