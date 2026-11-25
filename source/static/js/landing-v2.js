(() => {
  for (const block of document.querySelectorAll('.reviews')) {
    const track = block.querySelector('.review-grid');
    for (const circle of track.querySelectorAll('.author-circle')) {
      const name = circle.nextElementSibling.textContent.trim().split(/\s+/);
      circle.textContent = name.slice(0,2).map(word => Array.from(word)[0]).join('').toLocaleUpperCase();
    }
    block.querySelectorAll('[data-slide]').forEach(button => button.onclick = () => {
      track.scrollBy({left: (track.firstElementChild.offsetWidth + parseFloat(getComputedStyle(track).columnGap)) * Number(button.dataset.slide), behavior:'smooth'});
    });
  }
  const dialog = document.getElementById('exitOffer');
  if (!dialog) return;
  const trigger = document.querySelector('[data-offer-open]');
  const fi = document.documentElement.lang === 'fi';
  const key = 'exit-offer-' + dialog.dataset.until;
  let shown = false, card, anchor, timer;
  try { shown = sessionStorage.getItem(key) === '1'; } catch {}
  function remaining() {
    const seconds = Math.max(0, Math.ceil(Number(dialog.dataset.until) - Date.now()/1000));
    const d = Math.floor(seconds/86400), h = Math.floor(seconds/3600)%24, m = Math.floor(seconds/60)%60;
    dialog.querySelector('[data-countdown]').textContent = (fi?'Aikaa jäljellä ':'Time remaining ') + d + (fi?' pv ':' d ') + h + ' h ' + m + ' min ' + seconds%60 + ' s';
    if (!seconds) { trigger.hidden = true; if (dialog.open) dialog.close(); }
    return seconds;
  }
  function open() {
    if (dialog.open || !remaining() || document.querySelector('.liikkuva-consent-modal-backdrop.is-open') || document.querySelector('.liikkuva-consent-root:not([style*="none"])')) return;
    card = document.querySelector('#footerLeadFormMount .request-card');
    if (!card) return;
    anchor = document.createComment('offer-form');
    card.replaceWith(anchor);
    dialog.querySelector('[data-offer-form]').append(card);
    dialog.showModal();
    shown = true;
    try { sessionStorage.setItem(key,'1'); } catch {}
    timer = setInterval(remaining,1000);
  }
  dialog.addEventListener('close', () => {clearInterval(timer); if(anchor) anchor.replaceWith(card);});
  trigger.onclick = open;
  dialog.querySelector('.offer-close').onclick = () => dialog.close();
  // On touch devices the browser exposes neither a close-button event nor a
  // pointer leaving the viewport. A deliberate return to the top after
  // reading the page provides an exit-intent signal without trapping Back.
  let furthest = window.scrollY, previous = furthest, touchScrolling = false;
  function editingRequest() {
    return !!document.activeElement?.closest('form, dialog') ||
      Array.from(document.querySelectorAll('form input[name="phone"], form textarea[name="message"]'))
        .some(field => field.value.trim());
  }
  window.addEventListener('touchmove', () => {touchScrolling=true;},{passive:true});
  window.addEventListener('scroll', () => {
    if (!matchMedia('(pointer:coarse)').matches || !touchScrolling || shown) return;
    const y=window.scrollY;
    furthest=Math.max(furthest,y);
    if (furthest>innerHeight*1.5 && previous>y && y<64 && !editingRequest()) open();
    previous=y;
  },{passive:true});
  document.addEventListener('mouseout', e => {
    if (!shown && !editingRequest() && !e.relatedTarget && e.clientY <= 0 && matchMedia('(pointer:fine)').matches) open();
  });
})();
