/* Nexora enhances native links/forms; it performs no business API calls. */
(() => {
  'use strict';
  document.documentElement.classList.add('js');
  const header = document.querySelector('[data-header]');
  const menu = document.querySelector('#primary-nav');
  const menuToggle = document.querySelector('[data-menu-toggle]');
  const desktop = window.matchMedia('(min-width: 761px)');
  function setMenu(open, restoreFocus = false) {
    if (!menu || !menuToggle) return;
    menu.classList.toggle('is-open', open);
    document.body.classList.toggle('menu-open', open);
    menuToggle.setAttribute('aria-expanded', String(open));
    menuToggle.setAttribute('aria-label', open ? '关闭主导航' : '展开主导航');
    menuToggle.dataset.open = String(open);
    if (restoreFocus) menuToggle.focus();
  }
  menuToggle?.addEventListener('click', () => setMenu(menuToggle.getAttribute('aria-expanded') !== 'true'));
  menu?.addEventListener('click', event => { if (event.target.closest('a')) setMenu(false); });
  desktop.addEventListener('change', event => { if (event.matches) setMenu(false); });
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape' && menuToggle?.getAttribute('aria-expanded') === 'true') setMenu(false, true);
    if (event.key !== 'Tab' || menuToggle?.getAttribute('aria-expanded') !== 'true' || !header) return;
    const focusable = [...header.querySelectorAll('a, button, summary')].filter(node => node.getClientRects().length && !node.closest('details:not([open]) .nav-dropdown'));
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
  });

  const search = document.querySelector('#site-search');
  let searchReturn = null;
  document.querySelectorAll('[data-search-open]').forEach(link => link.addEventListener('click', event => {
    if (!search || typeof search.showModal !== 'function') return;
    event.preventDefault();
    setMenu(false);
    searchReturn = link;
    if (!search.open) search.showModal();
    document.body.classList.add('search-open');
    search.querySelector('input')?.focus();
  }));
  search?.querySelector('[data-search-close]')?.addEventListener('click', () => search.close());
  search?.addEventListener('click', event => {
    if (event.target !== search) return;
    const rect = search.getBoundingClientRect();
    if (event.clientY > rect.bottom || event.clientX < rect.left || event.clientX > rect.right) search.close();
  });
  search?.addEventListener('close', () => {
    document.body.classList.remove('search-open');
    searchReturn?.focus();
  });

  const hero = document.querySelector('[data-hero]');
  if (!hero) return;
  const slides = [...hero.querySelectorAll('[data-slide]')];
  const tabs = [...hero.querySelectorAll('[data-slide-to]')];
  const controls = hero.querySelector('[data-hero-controls]');
  const currentLabel = hero.querySelector('[data-current-slide]');
  const status = hero.querySelector('[data-slide-status]');
  let current = 0;
  function showSlide(next) {
    next = (next + slides.length) % slides.length;
    if (next === current) return;
    slides.forEach((slide, index) => {
      slide.hidden = index !== next;
      slide.classList.toggle('is-entering', index === next);
    });
    tabs.forEach((tab, index) => tab.setAttribute('aria-pressed', String(index === next)));
    current = next;
    if (currentLabel) currentLabel.textContent = String(current + 1).padStart(2, '0');
    if (status) status.textContent = `第 ${current + 1} 张，共 ${slides.length} 张`;
  }
  if (controls && slides.length > 1) controls.hidden = false;
  tabs.forEach(tab => tab.addEventListener('click', () => showSlide(Number(tab.dataset.slideTo))));
  hero.querySelectorAll('[data-slide-step]').forEach(button => button.addEventListener('click', () => showSlide(current + Number(button.dataset.slideStep))));
  controls?.addEventListener('keydown', event => {
    if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') {
      event.preventDefault();
      showSlide(current + (event.key === 'ArrowRight' ? 1 : -1));
    }
  });
})();
