(function (root, factory) {
  'use strict';
  const api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  if (!root.document) return;
  // In case an older template includes the script twice, bind only once.
  if (root.__liikkuvaThemeInitialized) return;
  root.__liikkuvaThemeInitialized = true;
  const document = root.document;
  const html = document.documentElement;
  const key = 'liikkuva_theme_v1';
  let preference = 'auto';
  try { preference = api.preference(root.localStorage.getItem(key)); } catch (_) {}
  const media = root.matchMedia ? root.matchMedia('(prefers-color-scheme: dark)') : null;
  function apply() {
    const theme = api.resolve({preference, policy: html.dataset.themePolicy || 'system',
      start: html.dataset.nightStart, end: html.dataset.nightEnd,
      systemDark: !!(media && media.matches), now: new Date()});
    html.dataset.theme = theme;
    document.querySelectorAll('[data-theme-picker]').forEach(select => { select.value = preference; });
    document.querySelectorAll('[data-theme-toggle]').forEach(button => {
      const dark = theme === 'dark';
      const fi = html.lang === 'fi';
      button.setAttribute('aria-pressed', String(dark));
      button.setAttribute('aria-label', fi ? (dark ? 'Vaihda vaaleaan tilaan' : 'Vaihda tummaan tilaan')
        : (dark ? 'Switch to light mode' : 'Switch to dark mode'));
      button.title = button.getAttribute('aria-label');
    });
  }
  apply();
  document.addEventListener('DOMContentLoaded', () => {
    apply();
    document.querySelectorAll('[data-theme-toggle]').forEach(button => {
      button.addEventListener('click', () => {
        preference = html.dataset.theme === 'dark' ? 'light' : 'dark';
        try { root.localStorage.setItem(key, preference); } catch (_) {}
        apply();
      });
    });
    document.querySelectorAll('[data-theme-picker]').forEach(select => {
      select.value = preference;
      select.addEventListener('change', () => {
        preference = api.preference(select.value);
        try { root.localStorage.setItem(key, preference); } catch (_) {}
        apply();
      });
    });
  });
  if (media && media.addEventListener) media.addEventListener('change', apply);
  else if (media && media.addListener) media.addListener(apply);
  root.addEventListener('storage', event => { if (event.key === key) { preference = api.preference(event.newValue); apply(); } });
  root.addEventListener('focus', apply);
  document.addEventListener('visibilitychange', () => { if (!document.hidden) apply(); });
  root.setInterval(() => { if (!document.hidden) apply(); }, 60000);
})(typeof window === 'undefined' ? globalThis : window, function () {
  'use strict';
  function preference(value) { return ['light', 'dark'].includes(value) ? value : 'auto'; }
  function minute(value, fallback) {
    return /^([01]\d|2[0-3]):[0-5]\d$/.test(value || '')
      ? Number(value.slice(0, 2)) * 60 + Number(value.slice(3)) : fallback;
  }
  function inSchedule(current, start, end) {
    if (start === end) return false;
    return start < end ? current >= start && current < end : current >= start || current < end;
  }
  function helsinkiMinute(date) {
    const parts = new Intl.DateTimeFormat('en-GB', {timeZone: 'Europe/Helsinki',
      hour: '2-digit', minute: '2-digit', hourCycle: 'h23'}).formatToParts(date);
    const part = name => Number(parts.find(p => p.type === name).value);
    return (part('hour') % 24) * 60 + part('minute');
  }
  function resolve(options) {
    const selected = preference(options.preference);
    if (selected !== 'auto') return selected;
    if (options.policy === 'dark') return 'dark';
    if (options.policy === 'system') return options.systemDark ? 'dark' : 'light';
    if (options.policy === 'night') {
      try {
        return inSchedule(helsinkiMinute(options.now), minute(options.start, 1260), minute(options.end, 420)) ? 'dark' : 'light';
      } catch (_) { return 'light'; }
    }
    return 'light';
  }
  return {preference, minute, inSchedule, helsinkiMinute, resolve};
});
