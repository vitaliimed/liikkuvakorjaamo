(() => {
  const key = 'liikkuva_first_landing_v1';
  const utms = ['utm_source','utm_medium','utm_campaign','utm_content','utm_term'];
  const clicks = ['gclid','gbraid','wbraid','fbclid'];
  let first;
  function capture() {
    const consent = window.liikkuvaConsentState?.() || {};
    if (!consent.analytics && !consent.marketing) {
      first = null;
      try {sessionStorage.removeItem(key);} catch {}
      return;
    }
    if (!first) {
      try { first = JSON.parse(sessionStorage.getItem(key)); } catch {}
    }
    const query = new URLSearchParams(location.search);
    if (!first) {
      first = {landing_page:location.pathname, referer:''};
      try {first.referer = new URL(document.referrer).origin;} catch {}
    }
    for (const name of [...utms,...clicks]) {
      const permitted = clicks.includes(name) ? consent.marketing : consent.analytics || consent.marketing;
      if (!permitted) delete first[name];
      else if (first.landing_page === location.pathname && !(name in first)) {
        const value = query.get(name) || '';
        first[name] = (clicks.includes(name) && !/^[a-z0-9_-]{1,240}$/i.test(value)) ? '' : value.slice(0,240);
      }
    }
    try {sessionStorage.setItem(key,JSON.stringify(first));} catch {}
  }
  window.addEventListener('liikkuva:consent-updated',capture);
  document.addEventListener('DOMContentLoaded',capture);
  window.liikkuvaSetLeadAttribution = data => {
    capture();
    data.set('page_url',location.origin+location.pathname);
    data.set('form_page',location.pathname);
    data.set('service_type',document.querySelector('.hero-copy h1')?.textContent.trim().slice(0,160) || '');
    for (const name of ['landing_page','referer',...utms,...clicks]) {
      data.set(name,first?.[name] || '');
    }
  };
})();
