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
  const fi = document.documentElement.lang === 'fi';
  const key = 'exit-offer-' + dialog.dataset.until;
  let shown = false, card, anchor, timer;
  try { shown = sessionStorage.getItem(key) === '1'; } catch {}
  function remaining() {
    const seconds = Math.max(0, Math.ceil(Number(dialog.dataset.until) - Date.now()/1000));
    const d = Math.floor(seconds/86400), h = Math.floor(seconds/3600)%24, m = Math.floor(seconds/60)%60;
    dialog.querySelector('[data-countdown]').textContent = (fi?'Aikaa jäljellä ':'Time remaining ') + d + (fi?' pv ':' d ') + h + ' h ' + m + ' min ' + seconds%60 + ' s';
    if (!seconds && dialog.open) dialog.close();
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
  dialog.querySelector('.offer-close').onclick = () => dialog.close();
  // On touch devices the browser exposes neither a close-button event nor a
  // pointer leaving the viewport. A deliberate return to the top after
  // reading the page provides an exit-intent signal without trapping Back.
  let furthest = window.scrollY, swipe, fastUntil = 0;
  function editingRequest() {
    return !!document.activeElement?.closest('form, dialog') ||
      Array.from(document.querySelectorAll('form input[name="phone"], form textarea[name="message"]'))
        .some(field => field.value.trim());
  }
  const coarse = matchMedia('(pointer:coarse)');
  function checkReturn() {
    if (!shown && coarse.matches && performance.now() < fastUntil &&
        furthest > innerHeight * 1.5 && scrollY < 64 && !editingRequest()) open();
  }
  window.addEventListener('scroll', () => {
    furthest = Math.max(furthest, scrollY);
    checkReturn();
  }, {passive:true});
  window.addEventListener('touchstart', e => {
    fastUntil = 0;
    const t = e.touches[0];
    swipe = e.touches.length === 1 && scrollY >= 64 &&
      {x:t.clientX, y:t.clientY, time:performance.now()};
  }, {passive:true});
  window.addEventListener('touchcancel', () => { swipe = null; }, {passive:true});
  window.addEventListener('touchend', e => {
    if (!swipe) return;
    const t = e.changedTouches[0], dy = t.clientY - swipe.y;
    if (dy >= 90 && dy > 2 * Math.abs(t.clientX - swipe.x) &&
        dy >= 0.7 * (performance.now() - swipe.time)) {
      fastUntil = performance.now() + 550;
      checkReturn();
    }
    swipe = null;
  }, {passive:true});
  let moves = [];
  document.addEventListener('mousemove', e => {
    if (shown || coarse.matches) return;
    moves = moves.filter(p => e.timeStamp - p.time <= 180);
    moves.push({x:e.clientX, y:e.clientY, time:e.timeStamp});
  });
  document.addEventListener('mouseout', e => {
    if (e.relatedTarget) return;
    const upward = moves.some(p => {
      const dy = p.y - e.clientY, ms = e.timeStamp - p.time;
      return ms <= 180 && dy >= 65 && dy >= 0.7 * ms && dy > Math.abs(e.clientX - p.x);
    });
    moves = [];
    if (upward && e.clientY <= 0 && !shown && !coarse.matches && !editingRequest()) open();
  });
})();
