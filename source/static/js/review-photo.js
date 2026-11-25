(() => {
  const active = new WeakMap();
  document.addEventListener('submit', event => {
    if (active.get(event.target)) {
      event.preventDefault();
      alert('Дождитесь загрузки фотографии.');
    }
  },true);
  document.addEventListener('change', async event => {
    const input = event.target;
    if (!input.matches('[data-review-photo]') || !input.files.length) return;
    const group = input.closest('.field, .review-edit-photo');
    const url = group.querySelector('input[type="text"]');
    const status = group.querySelector('output');
    const form = input.form;
    active.set(form,(active.get(form)||0)+1);
    input.disabled = true;
    status.textContent = 'Обрабатываем фотографию…';
    try {
      const data = new FormData();
      data.set('photo',input.files[0]);
      data.set('access_csrf',form.elements.access_csrf.value);
      const res = await fetch('/admin/reviews/photo',{method:'POST',body:data});
      if (!res.ok) throw Error(await res.text());
      url.value = (await res.json()).url;
      status.textContent = 'Фото готово. Сохраните изменения.';
    } catch (err) { status.textContent = err.message; }
    finally { input.disabled=false; input.value=''; active.set(form,active.get(form)-1); }
  });
})();
