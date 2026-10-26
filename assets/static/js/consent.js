(function () {
  "use strict";

  var tracking = window.LIIKKUVA_TRACKING || {};
  var ui = window.LIIKKUVA_CONSENT_UI || {};
  var consentKey = window.LIIKKUVA_CONSENT_KEY || "liikkuva_consent_v1";

  var pendingLeadKey = "liikkuva_pending_lead_v1";
  var sentLeadKey = "liikkuva_sent_lead_v1";

  var state = {
    ga4Configured: false,
    adsConfigured: false,
    metaInitialized: false,
    metaPageViewSent: false,
    tiktokInitialized: false,
    tiktokPageViewSent: false,
    linkedinInitialized: false,
    stylesInjected: false,
    bannerEl: null,
    modalEl: null,
    manageButtonEl: null
  };

  function getTexts() {
    var lang = (ui.lang || "en").toLowerCase();

    var defaults = {
      fi: {
        badge: "Evästeet",
        title: "Käytämme välttämättömiä, analytiikka- ja mainosevästeitä",
        text: "Välttämättömät evästeet pitävät sivun toiminnassa. Analytiikka auttaa ymmärtämään sivun käyttöä ja kampanjoiden tehokkuutta. Mainonnan tagit aktivoituvat vain valintasi mukaan.",
        settingsTitle: "Evästeasetukset",
        modalText: "Voit sallia analytiikan ja mainonnan erikseen. Välttämättömät evästeet ovat aina käytössä, koska ilman niitä sivu ei toimi oikein.",
        necessaryTitle: "Välttämättömät",
        necessaryText: "Aina käytössä sivun perustoimintoihin.",
        analyticsTitle: "Analytiikka",
        analyticsText: "GA4-mittaus ja sivuston suorituskyvyn analyysi.",
        marketingTitle: "Mainonta",
        marketingText: "Google Ads-, Meta-, LinkedIn- ja TikTok-tagit remarketingiin ja mainonnan mittaukseen.",
        acceptAll: "Hyväksy kaikki",
        rejectAll: "Vain välttämättömät",
        settings: "Asetukset",
        save: "Tallenna valinta",
        manage: "Evästeasetukset",
        privacy: "Tietosuoja",
        close: "Sulje"
      },
      en: {
        badge: "Cookies",
        title: "We use necessary, analytics and advertising cookies",
        text: "Necessary cookies keep the site working. Analytics helps understand site usage and campaign performance. Advertising tags activate only according to your choice.",
        settingsTitle: "Cookie settings",
        modalText: "You can allow analytics and advertising separately. Necessary cookies are always enabled because the site cannot work properly without them.",
        necessaryTitle: "Necessary",
        necessaryText: "Always enabled for core site functionality.",
        analyticsTitle: "Analytics",
        analyticsText: "GA4 measurement and site performance analytics.",
        marketingTitle: "Advertising",
        marketingText: "Google Ads, Meta, LinkedIn and TikTok tags for remarketing and ad measurement.",
        acceptAll: "Accept all",
        rejectAll: "Only necessary",
        settings: "Settings",
        save: "Save selection",
        manage: "Cookie settings",
        privacy: "Privacy policy",
        close: "Close"
      }
    };

    var fallback = defaults[lang] || defaults.en;
    return {
      badge: ui.texts && ui.texts.badge ? ui.texts.badge : fallback.badge,
      title: ui.texts && ui.texts.title ? ui.texts.title : fallback.title,
      text: ui.texts && ui.texts.text ? ui.texts.text : fallback.text,
      settingsTitle: ui.texts && ui.texts.settingsTitle ? ui.texts.settingsTitle : fallback.settingsTitle,
      modalText: ui.texts && ui.texts.modalText ? ui.texts.modalText : fallback.modalText,
      necessaryTitle: ui.texts && ui.texts.necessaryTitle ? ui.texts.necessaryTitle : fallback.necessaryTitle,
      necessaryText: ui.texts && ui.texts.necessaryText ? ui.texts.necessaryText : fallback.necessaryText,
      analyticsTitle: ui.texts && ui.texts.analyticsTitle ? ui.texts.analyticsTitle : fallback.analyticsTitle,
      analyticsText: ui.texts && ui.texts.analyticsText ? ui.texts.analyticsText : fallback.analyticsText,
      marketingTitle: ui.texts && ui.texts.marketingTitle ? ui.texts.marketingTitle : fallback.marketingTitle,
      marketingText: ui.texts && ui.texts.marketingText ? ui.texts.marketingText : fallback.marketingText,
      acceptAll: ui.texts && ui.texts.acceptAll ? ui.texts.acceptAll : fallback.acceptAll,
      rejectAll: ui.texts && ui.texts.rejectAll ? ui.texts.rejectAll : fallback.rejectAll,
      settings: ui.texts && ui.texts.settings ? ui.texts.settings : fallback.settings,
      save: ui.texts && ui.texts.save ? ui.texts.save : fallback.save,
      manage: ui.texts && ui.texts.manage ? ui.texts.manage : fallback.manage,
      privacy: ui.texts && ui.texts.privacy ? ui.texts.privacy : fallback.privacy,
      close: ui.texts && ui.texts.close ? ui.texts.close : fallback.close
    };
  }

  function safeParse(raw, fallback) {
    try {
      return raw ? JSON.parse(raw) : fallback;
    } catch (error) {
      return fallback;
    }
  }

  function getDefaultConsent() {
    return {
      analytics: false,
      marketing: false,
      savedAt: ""
    };
  }

  function normalizeConsent(input) {
    var base = getDefaultConsent();
    if (!input || typeof input !== "object") return base;

    return {
      analytics: !!input.analytics,
      marketing: !!input.marketing,
      savedAt: input.savedAt || ""
    };
  }

  function getSavedConsent() {
    var raw = "";
    try {
      raw = localStorage.getItem(consentKey);
    } catch (error) {
      raw = "";
    }
    return normalizeConsent(safeParse(raw, getDefaultConsent()));
  }

  function hasSavedConsent() {
    try {
      return !!localStorage.getItem(consentKey);
    } catch (error) {
      return false;
    }
  }

  function saveConsent(consent) {
    var payload = normalizeConsent(consent);
    payload.savedAt = new Date().toISOString();

    try {
      localStorage.setItem(consentKey, JSON.stringify(payload));
    } catch (error) {
      console.error("Consent save failed", error);
    }

    return payload;
  }

  function hasAnalyticsConsent() {
    return !!getSavedConsent().analytics;
  }

  function hasMarketingConsent() {
    return !!getSavedConsent().marketing;
  }

  function updateGoogleConsent(consent) {
    if (typeof window.gtag !== "function") return;

    window.gtag("consent", "update", {
      analytics_storage: consent.analytics ? "granted" : "denied",
      ad_storage: consent.marketing ? "granted" : "denied",
      ad_user_data: consent.marketing ? "granted" : "denied",
      ad_personalization: consent.marketing ? "granted" : "denied"
    });
  }

  function initGoogleTags(consent) {
    if (typeof window.gtag !== "function") return;
    if(!state.googleInitialized && ((consent.analytics && tracking.ga4MeasurementId)||(consent.marketing && tracking.googleAdsTagId))){window.gtag("js",new Date());state.googleInitialized=true;}

    if (tracking.ga4MeasurementId && consent.analytics && !state.ga4Configured) {
      loadScriptOnce("liikkuva-google-tag", "https://www.googletagmanager.com/gtag/js?id="+encodeURIComponent(tracking.ga4MeasurementId));
      var campaignFields={utm_source:"campaign_source",utm_medium:"campaign_medium",utm_campaign:"campaign_name",utm_id:"campaign_id",utm_term:"campaign_term",utm_content:"campaign_content"};
      var campaignParams=new URLSearchParams(window.location.search||"");
      Object.keys(campaignFields).forEach(function(key){var value=campaignParams.get(key)||"";if (/^[A-Za-z0-9_. -]{1,100}$/.test(value)) window.gtag("set",campaignFields[key],value);});
      window.gtag("config", tracking.ga4MeasurementId, {
        anonymize_ip: true,
        send_page_view: true,
        site_brand: "liikkuva",site_domain: "liikkuvakorjaamo.fi",
        page_location: window.liikkuvaSafeLocation ? window.liikkuvaSafeLocation(consent.marketing) : window.location.origin + window.location.pathname,
        page_referrer: (function(){try{var u=new URL(document.referrer);return u.origin+u.pathname;}catch(_){return "";}})()
      });
      state.ga4Configured = true;
    }

    if (tracking.googleAdsTagId && consent.marketing && !state.adsConfigured) {
      loadScriptOnce("liikkuva-google-tag", "https://www.googletagmanager.com/gtag/js?id="+encodeURIComponent(tracking.googleAdsTagId));
      window.gtag("config", tracking.googleAdsTagId, {page_location:window.liikkuvaSafeLocation ? window.liikkuvaSafeLocation(true) : location.origin+location.pathname});
      state.adsConfigured = true;
    }
  }

  function loadScriptOnce(id, src, onload) {
    var existing = document.getElementById(id);
    if (existing) {
      if (typeof onload === "function") onload();
      return;
    }

    var script = document.createElement("script");
    script.id = id;
    script.async = true;
    script.src = src;
    if (typeof onload === "function") {
      script.onload = onload;
    }
    document.head.appendChild(script);
  }

  function initMetaPixel() {
    if (!tracking.metaPixelId || !hasMarketingConsent()) return;
    if (state.metaInitialized) return;

    if (!window.fbq) {
      var fbq = function () {
        fbq.callMethod
          ? fbq.callMethod.apply(fbq, arguments)
          : fbq.queue.push(arguments);
      };
      fbq.queue = [];
      fbq.loaded = true;
      fbq.version = "2.0";
      window.fbq = fbq;
      window._fbq = fbq;

      loadScriptOnce(
        "liikkuva-meta-pixel",
        "https://connect.facebook.net/en_US/fbevents.js"
      );
    }

    try {
      window.fbq("consent", "grant");
      window.fbq("init", String(tracking.metaPixelId));
      if (!state.metaPageViewSent) {
        window.fbq("track", "PageView");
        state.metaPageViewSent = true;
      }
      state.metaInitialized = true;
    } catch (error) {
      console.error("Meta Pixel init failed", error);
    }
  }

  function revokeMetaConsent() {
    if (typeof window.fbq !== "function") return;
    try {
      window.fbq("consent", "revoke");
    } catch (error) {}
  }

  function initTikTokPixel() {
    if (!tracking.tiktokPixelId || !hasMarketingConsent()) return;
    if (state.tiktokInitialized) return;

    if (!window.ttq) {
      var ttq = (window.ttq = window.ttq || []);
      ttq.methods = [
        "page",
        "track",
        "identify",
        "instances",
        "debug",
        "on",
        "off",
        "once",
        "ready",
        "alias",
        "group",
        "enableCookie",
        "disableCookie"
      ];

      ttq.setAndDefer = function (obj, method) {
        obj[method] = function () {
          obj.push([method].concat(Array.prototype.slice.call(arguments, 0)));
        };
      };

      for (var i = 0; i < ttq.methods.length; i += 1) {
        ttq.setAndDefer(ttq, ttq.methods[i]);
      }

      ttq.load = function (pixelId, options) {
        var url =
          "https://analytics.tiktok.com/i18n/pixel/events.js?sdkid=" +
          encodeURIComponent(pixelId) +
          "&lib=ttq";
        ttq._i = ttq._i || {};
        ttq._i[pixelId] = [];
        ttq._o = ttq._o || {};
        ttq._o[pixelId] = options || {};
        loadScriptOnce("liikkuva-tiktok-pixel", url);
      };
    }

    try {
      window.ttq.load(String(tracking.tiktokPixelId));
      if (!state.tiktokPageViewSent && typeof window.ttq.page === "function") {
        window.ttq.page();
        state.tiktokPageViewSent = true;
      }
      state.tiktokInitialized = true;
    } catch (error) {
      console.error("TikTok Pixel init failed", error);
    }
  }

  function initLinkedInInsight() {
    if (!tracking.linkedinPartnerId || !hasMarketingConsent()) return;
    if (state.linkedinInitialized) return;

    try {
      window._linkedin_data_partner_ids = window._linkedin_data_partner_ids || [];
      if (window._linkedin_data_partner_ids.indexOf(String(tracking.linkedinPartnerId)) === -1) {
        window._linkedin_data_partner_ids.push(String(tracking.linkedinPartnerId));
      }

      if (!window.lintrk) {
        window.lintrk = function (a, b) {
          window.lintrk.q.push([a, b]);
        };
        window.lintrk.q = [];
      }

      loadScriptOnce(
        "liikkuva-linkedin-insight",
        "https://snap.licdn.com/li.lms-analytics/insight.min.js"
      );

      state.linkedinInitialized = true;
    } catch (error) {
      console.error("LinkedIn Insight init failed", error);
    }
  }

  function initMarketingTags() {
    if (!hasMarketingConsent()) return;
    initMetaPixel();
    initTikTokPixel();
    initLinkedInInsight();
  }

  function setPendingLead(payload) {
    try {
      sessionStorage.setItem(
        pendingLeadKey,
        JSON.stringify({
          at: Date.now(),
          payload: payload || {}
        })
      );
      sessionStorage.removeItem(sentLeadKey);
    } catch (error) {}
  }

  function getPendingLead() {
    return safeParse(sessionStorage.getItem(pendingLeadKey), null);
  }

  function clearPendingLead() {
    try {
      sessionStorage.removeItem(pendingLeadKey);
    } catch (error) {}
  }

  function markLeadSent(payload) {
    try {
      sessionStorage.setItem(
        sentLeadKey,
        JSON.stringify({
          at: Date.now(),
          payload: payload || {}
        })
      );
    } catch (error) {}
  }

  function hasLeadAlreadyBeenSent() {
    try {
      return !!sessionStorage.getItem(sentLeadKey);
    } catch (error) {
      return false;
    }
  }

  function toLinkedInConversionId(value) {
    if (value === undefined || value === null || value === "") return null;
    var num = Number(value);
    return isNaN(num) ? String(value) : num;
  }

  function sendLeadConversion(payload) {
    if (window.liikkuvaSavedLead) return false;
    if (!hasMarketingConsent()) return false;
    if (hasLeadAlreadyBeenSent()) return false;

    initMarketingTags();

    var eventPayload = payload || {};
    var pageLanguage = eventPayload.page_language || (ui.lang || "en");

    try {
      if (typeof window.fbq === "function") {
        window.fbq("track", "Lead", {
          content_name: "Lead form",
          page_language: pageLanguage
        });
      }
    } catch (error) {
      console.error("Meta lead track failed", error);
    }

    try {
      if (window.ttq && typeof window.ttq.track === "function") {
        window.ttq.track("SubmitForm", {
          content_name: "Lead form",
          page_language: pageLanguage
        });
      }
    } catch (error) {
      console.error("TikTok submit track failed", error);
    }

    try {
      if (typeof window.lintrk === "function" && tracking.linkedinLeadConversionID) {
        window.lintrk("track", {
          conversion_id: toLinkedInConversionId(tracking.linkedinLeadConversionID)
        });
      }
    } catch (error) {
      console.error("LinkedIn conversion track failed", error);
    }

    markLeadSent(eventPayload);
    return true;
  }

  function isThankYouPage() {
    return /^\/thank-you-(fi|en)$/.test(window.location.pathname || "");
  }

  function applyConsent(nextConsent) {
    var consent = saveConsent(nextConsent);

    updateGoogleConsent(consent);
    initGoogleTags(consent);

    if (consent.marketing) {
      initMarketingTags();
    } else {
      revokeMetaConsent();
    }

    hideBanner();
    closeSettingsModal();
    updateManageButtonVisibility();

    try {
      window.dispatchEvent(
        new CustomEvent("liikkuva:consent-updated", {
          detail: consent
        })
      );
    } catch (error) {}

    return consent;
  }

  function trackCustomEvent(name, params) {
    var payload = params || {};

    if (name === "lead_submit_success") {
      setPendingLead(payload);
      return true;
    }

    if (name === "lead_thankyou_view") {
      if (getPendingLead()) {
        sendLeadConversion(payload);
        clearPendingLead();
      }
      return true;
    }

    if (!hasMarketingConsent()) {
      return false;
    }

    initMarketingTags();

    try {
      if (name === "phone_click" && typeof window.fbq === "function") {
        window.fbq("track", "Contact");
      }

      if (name === "phone_click" && window.ttq && typeof window.ttq.track === "function") {
        window.ttq.track("ClickButton", {
          button_name: "phone"
        });
      }

      if (name === "mobile_sticky_form_click" && window.ttq && typeof window.ttq.track === "function") {
        window.ttq.track("ClickButton", {
          button_name: "mobile_sticky_form"
        });
      }
    } catch (error) {
      console.error("Custom marketing event failed", error);
    }

    return true;
  }

  function injectStyles() {
  if (state.stylesInjected) return;
  state.stylesInjected = true;

  var style = document.createElement("style");
  style.id = "liikkuva-consent-styles";
  style.textContent = `
    html.liikkuva-consent-lock,
    body.liikkuva-consent-lock {
      overflow: hidden !important;
    }

    .liikkuva-consent-root {
      position: fixed;
      inset: 0;
      pointer-events: none;
      z-index: 2147483000;
      font-family: Arial, Helvetica, sans-serif;
    }

    .liikkuva-consent-banner {
      position: fixed;
      left: 20px;
      bottom: 20px;
      width: min(460px, calc(100vw - 40px));
      background: #ffffff;
      border: 1px solid #dfe2d8;
      border-radius: 24px;
      box-shadow: 0 24px 70px rgba(2, 8, 23, 0.22);
      padding: 20px;
      pointer-events: auto;
      color: #202c29;
    }

    .liikkuva-consent-badge {
      display: inline-flex;
      align-items: center;
      padding: 7px 12px;
      border-radius: 999px;
      background: #fff0e7;
      color: #a74124;
      font-size: 12px;
      font-weight: 800;
      text-transform: uppercase;
      letter-spacing: .06em;
      margin-bottom: 10px;
    }

    .liikkuva-consent-title {
      margin: 0 0 8px;
      font-size: 22px;
      line-height: 1.15;
      font-weight: 800;
    }

    .liikkuva-consent-text {
      margin: 0 0 16px;
      color: #475569;
      font-size: 14px;
      line-height: 1.6;
    }

    .liikkuva-consent-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
    }

    .liikkuva-consent-btn {
      appearance: none;
      border: 0;
      border-radius: 16px;
      min-height: 48px;
      padding: 0 18px;
      font-size: 14px;
      font-weight: 800;
      cursor: pointer;
      transition: transform .18s ease, box-shadow .18s ease, opacity .18s ease;
    }

    .liikkuva-consent-btn:hover {
      transform: translateY(-1px);
    }

    .liikkuva-consent-btn-primary {
      background: #cf512b;
      color: #ffffff;
      box-shadow: 0 12px 28px rgba(234, 88, 12, 0.22);
    }

    .liikkuva-consent-btn-secondary {
      background: #eeeee6;
      color: #202c29;
    }

    .liikkuva-consent-btn-ghost {
      background: #ffffff;
      border: 1px solid #dfe2d8;
      color: #202c29;
    }

.liikkuva-consent-manage {
  position: fixed;
  right: 18px;
  bottom: 18px;
  left: auto;
  z-index: 2147483003;
  appearance: none;
  border: 1px solid #dfe2d8;
  background: rgba(255, 255, 255, 0.96);
  color: #202c29;
  box-shadow: 0 12px 32px rgba(2, 8, 23, 0.16);
  border-radius: 999px;
  width: 48px;
  height: 48px;
  min-height: 48px;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  pointer-events: auto;
  transition: transform .18s ease, box-shadow .18s ease, background .18s ease;
}

.liikkuva-consent-manage:hover {
  transform: translateY(-1px);
  box-shadow: 0 16px 38px rgba(2, 8, 23, 0.22);
  background: #ffffff;
}

.liikkuva-consent-manage svg {
  width: 22px;
  height: 22px;
  display: block;
}

.liikkuva-consent-manage .liikkuva-consent-manage-text {
  display: none;
}

    .liikkuva-consent-modal-backdrop {
      position: fixed;
      inset: 0;
      z-index: 2147483004;
      background: rgba(7, 17, 31, 0.72);
      backdrop-filter: blur(8px);
      -webkit-backdrop-filter: blur(8px);
      display: none;
      align-items: center;
      justify-content: center;
      padding: 24px;
      pointer-events: auto;
    }

    .liikkuva-consent-modal-backdrop.is-open {
      display: flex;
    }

    .liikkuva-consent-modal {
      position: relative;
      z-index: 2147483005;
      width: min(100%, 640px);
      max-height: calc(100vh - 48px);
      overflow-y: auto;
      background: #ffffff;
      border: 1px solid #dfe2d8;
      border-radius: 28px;
      box-shadow: 0 32px 90px rgba(2, 8, 23, 0.35);
      padding: 24px;
      color: #202c29;
    }

    .liikkuva-consent-modal-head {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 12px;
      margin-bottom: 12px;
    }

    .liikkuva-consent-modal-title {
      margin: 0;
      font-size: 34px;
      line-height: 1.08;
      font-weight: 800;
      letter-spacing: -0.02em;
    }

    .liikkuva-consent-close {
      appearance: none;
      border: 0;
      background: #eef2f7;
      color: #202c29;
      border-radius: 999px;
      width: 44px;
      height: 44px;
      font-size: 28px;
      line-height: 1;
      cursor: pointer;
      flex: 0 0 auto;
    }

    .liikkuva-consent-modal-text {
      margin: 0 0 18px;
      color: #64748b;
      font-size: 14px;
      line-height: 1.65;
    }

    .liikkuva-consent-option {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 14px;
      padding: 16px;
      border: 1px solid #dfe2d8;
      border-radius: 20px;
      background: #f8fbff;
    }

    .liikkuva-consent-option + .liikkuva-consent-option {
      margin-top: 12px;
    }

    .liikkuva-consent-option-copy strong {
      display: block;
      font-size: 16px;
      margin-bottom: 6px;
    }

    .liikkuva-consent-option-copy span {
      display: block;
      color: #64748b;
      font-size: 14px;
      line-height: 1.55;
    }

    .liikkuva-consent-switch {
      position: relative;
      display: inline-block;
      width: 56px;
      height: 32px;
      flex: 0 0 auto;
    }

    .liikkuva-consent-switch input {
      opacity: 0;
      width: 0;
      height: 0;
    }

    .liikkuva-consent-slider {
      position: absolute;
      inset: 0;
      background: #cbd5e1;
      border-radius: 999px;
      transition: .2s;
    }

    .liikkuva-consent-slider:before {
      content: "";
      position: absolute;
      left: 4px;
      top: 4px;
      width: 24px;
      height: 24px;
      border-radius: 50%;
      background: #ffffff;
      transition: .2s;
      box-shadow: 0 2px 8px rgba(2, 8, 23, 0.18);
    }

    .liikkuva-consent-switch input:checked + .liikkuva-consent-slider {
      background: linear-gradient(135deg, #0ea5e9, #f59e0b);
    }

    .liikkuva-consent-switch input:checked + .liikkuva-consent-slider:before {
      transform: translateX(24px);
    }

    .liikkuva-consent-switch input:disabled + .liikkuva-consent-slider {
      opacity: .75;
      cursor: not-allowed;
    }

    .liikkuva-consent-modal-actions {
      position: sticky;
      bottom: -24px;
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 20px;
      padding-top: 14px;
      padding-bottom: 4px;
      background: linear-gradient(180deg, rgba(255,255,255,0), #ffffff 18px, #ffffff 100%);
    }

    .liikkuva-consent-footer {
      margin-top: 14px;
    }

    .liikkuva-consent-link {
      font-size: 13px;
      font-weight: 700;
      color: #a74124;
      text-decoration: none;
    }

        @media (max-width: 720px) {
      .liikkuva-consent-banner {
        left: 10px;
        right: 10px;
        bottom: 72px;
        width: auto;
        max-width: none;
        padding: 14px;
        border-radius: 18px;
        box-shadow: 0 14px 36px rgba(2, 8, 23, 0.16);
      }

      .liikkuva-consent-badge {
        margin-bottom: 8px;
        padding: 6px 10px;
        font-size: 10px;
      }

      .liikkuva-consent-title {
        margin: 0 0 6px;
        font-size: 16px;
        line-height: 1.2;
      }

      .liikkuva-consent-text {
        margin: 0 0 10px;
        font-size: 12px;
        line-height: 1.45;
      }

      .liikkuva-consent-actions {
        gap: 8px;
      }

      .liikkuva-consent-btn {
        min-height: 40px;
        padding: 0 12px;
        font-size: 12px;
      }

      .liikkuva-consent-manage {
        left: auto;
        right: 10px;
        bottom: 128px;
        width: 40px;
        height: 40px;
        min-width: 40px;
        min-height: 40px;
        padding: 0;
        border-radius: 999px;
      }

      .liikkuva-consent-manage-text {
        display: none;
      }

      .liikkuva-consent-manage svg {
        width: 18px;
        height: 18px;
      }

      .liikkuva-consent-modal-backdrop {
        padding: 12px;
      }

      .liikkuva-consent-modal {
        width: 100%;
        max-height: calc(100vh - 24px);
        padding: 18px;
        border-radius: 24px;
      }

      .liikkuva-consent-modal-title {
        font-size: 24px;
      }

      .liikkuva-consent-modal-actions {
        bottom: -18px;
      }
    }
  `;

  document.head.appendChild(style);
}

  function createBanner() {
    if (state.bannerEl) return state.bannerEl;

    var texts = getTexts();
    var root = document.createElement("div");
    root.className = "liikkuva-consent-root";

    var banner = document.createElement("div");
    banner.className = "liikkuva-consent-banner";
    banner.innerHTML =
      '<div class="liikkuva-consent-badge">' + escapeHtml(texts.badge) + "</div>" +
      '<h3 class="liikkuva-consent-title">' + escapeHtml(texts.title) + "</h3>" +
      '<p class="liikkuva-consent-text">' + escapeHtml(texts.text) + "</p>" +
      '<div class="liikkuva-consent-actions">' +
      '<button type="button" class="liikkuva-consent-btn liikkuva-consent-btn-secondary" data-consent-action="reject">' + escapeHtml(texts.rejectAll) + "</button>" +
      '<button type="button" class="liikkuva-consent-btn liikkuva-consent-btn-ghost" data-consent-action="settings">' + escapeHtml(texts.settings) + "</button>" +
      '<button type="button" class="liikkuva-consent-btn liikkuva-consent-btn-primary" data-consent-action="accept">' + escapeHtml(texts.acceptAll) + "</button>" +
      "</div>";

    root.appendChild(banner);
    document.body.appendChild(root);

    banner.querySelector('[data-consent-action="accept"]').addEventListener("click", function () {
      applyConsent({ analytics: true, marketing: true });
    });

    banner.querySelector('[data-consent-action="reject"]').addEventListener("click", function () {
      applyConsent({ analytics: false, marketing: false });
    });

    banner.querySelector('[data-consent-action="settings"]').addEventListener("click", function () {
      openSettingsModal();
    });

    state.bannerEl = root;
    return root;
  }

function createManageButton() {
  if (state.manageButtonEl) return state.manageButtonEl;

  var texts = getTexts();
  var button = document.createElement("button");
  button.type = "button";
  button.className = "liikkuva-consent-manage";
  button.setAttribute("aria-label", texts.manage);
  button.setAttribute("title", texts.manage);

  button.innerHTML = `
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path d="M14.6 3.2c.2 1-.1 2-.8 2.7-.8.8-2 1.1-3 .7a3.3 3.3 0 0 0-4.2 4.1c.4 1-.1 2.2-.9 3a3.4 3.4 0 0 0 2.4 5.8h6.5a6.4 6.4 0 0 0 0-12.8Z" fill="#F59E0B"/>
      <circle cx="10" cy="10" r="1.1" fill="#0F172A"/>
      <circle cx="13.8" cy="12.1" r="1" fill="#0F172A"/>
      <circle cx="11.3" cy="15" r="1" fill="#0F172A"/>
    </svg>
    <span class="liikkuva-consent-manage-text">` + escapeHtml(texts.manage) + `</span>
  `;

  button.addEventListener("click", function () {
    openSettingsModal();
  });

  document.body.appendChild(button);
  state.manageButtonEl = button;
  return button;
}
  function createSettingsModal() {
    if (state.modalEl) return state.modalEl;

    var texts = getTexts();
    var backdrop = document.createElement("div");
    backdrop.className = "liikkuva-consent-modal-backdrop";

    var privacyUrl = ui.privacyUrl || "/privacy";

    backdrop.innerHTML =
      '<div class="liikkuva-consent-modal" role="dialog" aria-modal="true" aria-labelledby="liikkuvaConsentModalTitle">' +
      '<div class="liikkuva-consent-modal-head">' +
      '<div>' +
      '<div class="liikkuva-consent-badge">' + escapeHtml(texts.badge) + "</div>" +
      '<h3 id="liikkuvaConsentModalTitle" class="liikkuva-consent-modal-title">' + escapeHtml(texts.settingsTitle) + "</h3>" +
      "</div>" +
      '<button type="button" class="liikkuva-consent-close" aria-label="' + escapeHtml(texts.close) + '">×</button>' +
      "</div>" +
      '<p class="liikkuva-consent-modal-text">' + escapeHtml(texts.modalText) + "</p>" +

      '<div class="liikkuva-consent-option">' +
      '<div class="liikkuva-consent-option-copy">' +
      "<strong>" + escapeHtml(texts.necessaryTitle) + "</strong>" +
      "<span>" + escapeHtml(texts.necessaryText) + "</span>" +
      "</div>" +
      '<label class="liikkuva-consent-switch">' +
      '<input type="checkbox" checked disabled>' +
      '<span class="liikkuva-consent-slider"></span>' +
      "</label>" +
      "</div>" +

      '<div class="liikkuva-consent-option">' +
      '<div class="liikkuva-consent-option-copy">' +
      "<strong>" + escapeHtml(texts.analyticsTitle) + "</strong>" +
      "<span>" + escapeHtml(texts.analyticsText) + "</span>" +
      "</div>" +
      '<label class="liikkuva-consent-switch">' +
      '<input type="checkbox" id="liikkuvaConsentAnalytics">' +
      '<span class="liikkuva-consent-slider"></span>' +
      "</label>" +
      "</div>" +

      '<div class="liikkuva-consent-option">' +
      '<div class="liikkuva-consent-option-copy">' +
      "<strong>" + escapeHtml(texts.marketingTitle) + "</strong>" +
      "<span>" + escapeHtml(texts.marketingText) + "</span>" +
      "</div>" +
      '<label class="liikkuva-consent-switch">' +
      '<input type="checkbox" id="liikkuvaConsentMarketing">' +
      '<span class="liikkuva-consent-slider"></span>' +
      "</label>" +
      "</div>" +

      '<div class="liikkuva-consent-modal-actions">' +
      '<button type="button" class="liikkuva-consent-btn liikkuva-consent-btn-secondary" data-consent-modal-action="reject">' + escapeHtml(texts.rejectAll) + "</button>" +
      '<button type="button" class="liikkuva-consent-btn liikkuva-consent-btn-primary" data-consent-modal-action="save">' + escapeHtml(texts.save) + "</button>" +
      "</div>" +

      '<div class="liikkuva-consent-footer">' +
      '<a class="liikkuva-consent-link" href="' + escapeAttribute(privacyUrl) + '">' + escapeHtml(texts.privacy) + "</a>" +
      "</div>" +
      "</div>";

    document.body.appendChild(backdrop);

    var closeButton = backdrop.querySelector(".liikkuva-consent-close");
    var rejectButton = backdrop.querySelector('[data-consent-modal-action="reject"]');
    var saveButton = backdrop.querySelector('[data-consent-modal-action="save"]');

    closeButton.addEventListener("click", closeSettingsModal);
    backdrop.addEventListener("click", function (event) {
      if (event.target === backdrop) closeSettingsModal();
    });

    rejectButton.addEventListener("click", function () {
      applyConsent({ analytics: false, marketing: false });
    });

    saveButton.addEventListener("click", function () {
      var analyticsEl = document.getElementById("liikkuvaConsentAnalytics");
      var marketingEl = document.getElementById("liikkuvaConsentMarketing");

      applyConsent({
        analytics: !!(analyticsEl && analyticsEl.checked),
        marketing: !!(marketingEl && marketingEl.checked)
      });
    });

    state.modalEl = backdrop;
    return backdrop;
  }

  function syncModalWithSavedConsent() {
    var modal = createSettingsModal();
    var consent = getSavedConsent();

    var analyticsEl = modal.querySelector("#liikkuvaConsentAnalytics");
    var marketingEl = modal.querySelector("#liikkuvaConsentMarketing");

    if (analyticsEl) analyticsEl.checked = !!consent.analytics;
    if (marketingEl) marketingEl.checked = !!consent.marketing;
  }

function openSettingsModal() {
  syncModalWithSavedConsent();

  if (state.modalEl) {
    state.modalEl.classList.add("is-open");
  }

  if (state.manageButtonEl) {
    state.manageButtonEl.style.display = "none";
  }

  document.documentElement.classList.add("liikkuva-consent-lock");
  document.body.classList.add("liikkuva-consent-lock");
}

function closeSettingsModal() {
  if (state.modalEl) {
    state.modalEl.classList.remove("is-open");
  }

  document.documentElement.classList.remove("liikkuva-consent-lock");
  document.body.classList.remove("liikkuva-consent-lock");

  updateManageButtonVisibility();
}

  function showBanner() {
    createBanner();
    if (state.bannerEl) {
      state.bannerEl.style.display = "block";
    }
    updateManageButtonVisibility();
  }

  function hideBanner() {
    if (state.bannerEl) {
      state.bannerEl.style.display = "none";
    }
  }

  function updateManageButtonVisibility() {
    createManageButton();
    if (!state.manageButtonEl) return;

    if (hasSavedConsent()) {
      state.manageButtonEl.style.display = "inline-flex";
    } else {
      state.manageButtonEl.style.display = "none";
    }
  }

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function escapeAttribute(value) {
    return escapeHtml(value);
  }

  function bootTracking() {
    var consent = getSavedConsent();
    updateGoogleConsent(consent);
    initGoogleTags(consent);

    if (consent.marketing) {
      initMarketingTags();
    } else {
      revokeMetaConsent();
    }
  }

  function bootLeadRecovery() {
    if (!isThankYouPage()) return;
    if (!getPendingLead()) return;
    if (hasLeadAlreadyBeenSent()) return;

    sendLeadConversion({
      page_language: ui.lang || "en",
      page_path: window.location.pathname
    });
    clearPendingLead();
  }

  function bootUI() {
    injectStyles();
    createSettingsModal();
    createManageButton();

    if (!hasSavedConsent()) {
      showBanner();
    } else {
      hideBanner();
      updateManageButtonVisibility();
    }
  }

  window.liikkuvaOpenConsentSettings = openSettingsModal;
  window.liikkuvaApplyConsentSettings = applyConsent;
  window.liikkuvaConsentState = getSavedConsent;
  window.liikkuvaInitMetaPixel = initMetaPixel;
  window.liikkuvaTrackCustomEvent = trackCustomEvent;

  document.addEventListener("DOMContentLoaded", function () {
    bootTracking();
    bootUI();
    bootLeadRecovery();
  });
})();