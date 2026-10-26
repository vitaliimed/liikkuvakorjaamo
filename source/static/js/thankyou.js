(function () {
  'use strict';
  document.addEventListener('DOMContentLoaded', function () {
    var countdown = document.getElementById('thankyouCountdown');
    var seconds = 60;
    if(countdown){var timer=setInterval(function(){seconds=Math.max(0,seconds-1);countdown.textContent=String(seconds);if(!seconds)clearInterval(timer);},1000);}
    var key = 'liikkuva_lead_success_v1';
    var payload;
    try { payload = JSON.parse(sessionStorage.getItem(key) || 'null'); } catch (_) {}
    if (!payload) return;
    var event = {
      page_language: payload.pageLanguage || document.documentElement.lang,
      has_photos: !!payload.hasPhotos,
      has_location: !!payload.hasLocation,
      page_path: window.location.pathname
    };
    var consent = typeof window.liikkuvaConsentState === 'function' ? window.liikkuvaConsentState() : {};
    var tracking = window.LIIKKUVA_TRACKING || {};
    if (typeof window.gtag === 'function') {
      // generate_lead is emitted once at the successful server acknowledgement.

    }
    if (typeof window.liikkuvaTrackCustomEvent === 'function') window.liikkuvaTrackCustomEvent('lead_thankyou_view', event);
    try { sessionStorage.removeItem(key); } catch (_) {}
  });
})();
