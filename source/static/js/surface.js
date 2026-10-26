document.addEventListener('DOMContentLoaded', () => {
  const menu = document.getElementById('menuToggle');
  const closeMenu = () => { document.body.classList.remove('nav-open'); menu?.setAttribute('aria-expanded', 'false'); };
  menu?.addEventListener('click', () => { const open = document.body.classList.toggle('nav-open'); menu.setAttribute('aria-expanded', String(open)); });
  document.querySelectorAll('#mainNav a').forEach(a => a.addEventListener('click', closeMenu));
  document.addEventListener('keydown', e => { if (e.key === 'Escape') closeMenu(); });
  document.querySelectorAll('[data-year]').forEach(el => { el.textContent = String(new Date().getFullYear()); });
  document.querySelectorAll('[data-service]').forEach(a => a.addEventListener('click', () => {
    const message = document.getElementById('message');
    if (message && !message.value.trim()) { message.value = a.dataset.service + ': '; message.dispatchEvent(new Event('input', {bubbles: true})); }
  }));
  const faq = document.querySelectorAll('.faq-list details');
  faq.forEach(el => el.addEventListener('toggle', () => { if (el.open) faq.forEach(other => { if (other !== el) other.open = false; }); }));
});
