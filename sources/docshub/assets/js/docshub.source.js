(() => {
  'use strict';
  const $ = selector => document.querySelector(selector);
  const $$ = selector => [...document.querySelectorAll(selector)];
  const root = document.documentElement;
  let toastTimer;
  const announce = message => { const toast = $('[data-toast]'); if (!toast) return; toast.textContent = message; toast.hidden = false; clearTimeout(toastTimer); toastTimer = setTimeout(() => { toast.hidden = true; }, 2600); };
  const syncMode = mode => {
    root.dataset.mode = mode === 'dark' ? 'dark' : 'light';
    $$('[data-color-mode]').forEach(button => button.setAttribute('aria-pressed', String(button.dataset.colorMode === root.dataset.mode)));
    $('meta[name="theme-color"]')?.setAttribute('content', root.dataset.mode === 'dark' ? '#111523' : '#f5f6fc');
  };
  syncMode(root.dataset.mode);
  $$('[data-color-mode]').forEach(button => button.addEventListener('click', () => { syncMode(button.dataset.colorMode); try { localStorage.setItem('docshub:mode', root.dataset.mode); } catch {} }));
  window.addEventListener('storage', event => { if (event.key === 'docshub:mode') syncMode(event.newValue); });
  const sidebar = $('.sidebar'), toggle = $('[data-menu-toggle]'), scrim = $('[data-menu-close]');
  const mobile = window.matchMedia('(max-width: 760px)');
  const setMenu = (open, focus = true) => {
    sidebar?.classList.toggle('is-open', open);
    document.body.classList.toggle('menu-open', open);
    toggle?.setAttribute('aria-expanded', String(open));
    toggle?.setAttribute('aria-label', open ? '关闭导航' : '打开导航');
    if (scrim) scrim.hidden = !open;
    if (sidebar) sidebar.inert = mobile.matches && !open;
    if (focus) { if (open) sidebar?.querySelector('a')?.focus(); else toggle?.focus(); }
  };
  toggle?.addEventListener('click', () => setMenu(toggle.getAttribute('aria-expanded') !== 'true'));
  scrim?.addEventListener('click', () => setMenu(false));
  sidebar?.addEventListener('click', event => { if (mobile.matches && event.target.closest('a')) setMenu(false, false); });
  mobile.addEventListener('change', () => setMenu(false, false));
  setMenu(false, false);
  document.addEventListener('keydown', event => {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') { event.preventDefault(); $('.search-form input')?.focus(); }
    if (event.key === 'Escape') { if (document.body.classList.contains('menu-open')) setMenu(false); $$('.topbar-dropdown[open]').forEach(menu => menu.open = false); }
    if (event.key === 'Tab' && document.body.classList.contains('menu-open')) {
      const nodes = [...sidebar.querySelectorAll('a,button,summary')].filter(node => node.getClientRects().length && !node.disabled);
      const first = nodes[0], last = nodes[nodes.length - 1];
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
    }
  });
  document.addEventListener('click', event => { $$('.topbar-dropdown[open]').forEach(menu => { if (!menu.contains(event.target)) menu.open = false; }); });
  $$('[data-relative-time]').forEach(time => {
    const date = new Date(time.dateTime), seconds = (Date.now() - date.getTime()) / 1000;
    if (!Number.isFinite(seconds) || seconds < 0) return;
    if (seconds < 60) time.textContent = '刚刚';
    else if (seconds < 3600) time.textContent = Math.floor(seconds / 60) + ' 分钟前';
    else if (seconds < 86400) time.textContent = Math.floor(seconds / 3600) + ' 小时前';
    else if (seconds < 604800) time.textContent = Math.floor(seconds / 86400) + ' 天前';
    time.title = date.toLocaleString(document.documentElement.lang || 'zh-CN');
  });
  const copy = async text => {
    if (navigator.clipboard?.writeText) { await navigator.clipboard.writeText(text); return; }
    const previous = document.activeElement;
    const field = document.createElement('textarea'); field.value = text; field.className = 'sr-only'; field.setAttribute('aria-hidden', 'true'); document.body.append(field); field.select();
    let ok = false; try { ok = document.execCommand('copy'); } finally { field.remove(); previous?.focus(); }
    if (!ok) throw new Error('clipboard unavailable');
  };
  $('[data-copy-link]')?.addEventListener('click', async () => { try { await copy(location.href); announce('链接已复制'); } catch { announce('复制失败，请从地址栏复制链接'); } });
  const article = $('[data-article-body]');
  if (!article) return;
  const reading = $('[data-reading-time]');
  if (reading) { const text = article.textContent || ''; const cjk = (text.match(/[\u3400-\u9fff]/g) || []).length; const words = text.replace(/[\u3400-\u9fff]/g, '').trim().split(/\s+/).filter(Boolean).length; reading.textContent = '约 ' + Math.max(1, Math.ceil(cjk / 350 + words / 220)) + ' 分钟阅读'; }
  article.querySelectorAll('pre').forEach(pre => {
    const code = pre.querySelector('code'), text = (code || pre).textContent;
    const button = document.createElement('button'); button.className = 'copy-code'; button.type = 'button'; button.textContent = '复制代码';
    button.addEventListener('click', async () => { try { await copy(text); button.textContent = '已复制'; announce('代码已复制'); setTimeout(() => { button.textContent = '复制代码'; }, 2200); } catch { announce('复制失败，请选中代码手动复制'); } }); pre.append(button);
  });
  const toc = $('[data-toc]'), list = $('[data-toc-list]');
  const headings = [...article.querySelectorAll('h2,h3')];
  const used = new Set($$('[id]').map(node => node.id));
  const links = new Map();
  headings.forEach((heading, index) => {
    if (!heading.id) { let id = 'section-' + (index + 1); while (used.has(id)) id += '-doc'; heading.id = id; used.add(id); }
    const li = document.createElement('li'), link = document.createElement('a');
    if (heading.tagName === 'H3') li.className = 'toc-sub';
    link.href = '#' + encodeURIComponent(heading.id); link.textContent = heading.textContent; li.append(link); list?.append(li); links.set(heading, link);
    $('[data-mobile-toc-list]')?.append(li.cloneNode(true));
  });
  if (toc && headings.length) toc.hidden = false;
  const mobileTOC = $('[data-mobile-toc]');
  if (mobileTOC && headings.length) { mobileTOC.hidden = false; mobileTOC.addEventListener('click', event => { if (event.target.closest('a')) mobileTOC.open = false; }); }
  const activate = heading => { for (const [target, link] of links) { if (target === heading) link.setAttribute('aria-current', 'true'); else link.removeAttribute('aria-current'); } };
  activate(headings[0]);
  let frame;
  const track = () => { if (frame) return; frame = requestAnimationFrame(() => { frame = undefined; let active = headings[0]; for (const heading of headings) { if (heading.getBoundingClientRect().top <= 130) active = heading; else break; } activate(active); }); };
  if (headings.length) { window.addEventListener('scroll', track, { passive: true }); track(); }
})();
