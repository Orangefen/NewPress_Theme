const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const { buildSync } = require('esbuild');

function load(name) {
  const code = buildSync({ entryPoints: [`sources/nexora/assets/js/modules/${name}.js`], bundle: true, write: false, format: 'cjs' }).outputFiles[0].text;
  const module = { exports: {} };
  vm.runInNewContext(code, { module, exports: module.exports, URL });
  return module.exports;
}
const state = load('navigation-state');
const carousel = load('carousel');
const search = load('search');
const navigation = load('navigation');
const url = path => new URL(path, 'https://nexora.test');
const select = (paths, path, section = '') => state.currentNavigation(paths.map(href => typeof href === 'string' ? { href } : href), url(path), section);

test('Nexora navigation: homepage anchors have one selected destination and return to home at the top', () => {
  const links = ['/', '/#about', '/#business', '/explore/'];
  assert.equal(select(links, '/').index, 0);
  assert.equal(select(links, '/#about', 'business').index, 2);
  assert.equal(select(links, '/#about', 'business').kind, 'location');
  assert.equal(select(links, '/#about').index, 0);
  assert.equal(select(['/#about'], '/pages/about/'), null);
});

test('Nexora navigation: exact destinations beat ancestor links and require matching query values', () => {
  const links = ['/', '/explore/', '/explore/?category=research', '/explore/?category=design'];
  assert.equal(select(links, '/explore/?category=design&page=2').index, 3);
  assert.equal(select(links, '/explore/?category=unknown').index, 1);
  assert.equal(select(['/pages/', '/pages/company/', '/pages/company/team/'], '/pages/company/team/history/').index, 2);
  assert.equal(select(['/pages/company/'], '/pages/company-extra/'), null);
});

test('Nexora navigation: external URLs, malformed fragments and non-page schemes cannot steal current state', () => {
  const links = ['https://other.test/', 'javascript:alert(1)', '/#%ZZ', 'https://nexora.test/#%E5%85%B3%E4%BA%8E', '/'];
  assert.equal(select(links, '/', '关于').index, 3);
  assert.equal(select(links, '/').index, 4);
  assert.equal(select(['/missing/', { href: '/explore/', sectionFallback: true }], '/posts/new-product/').index, 1);
  assert.equal(select(['/explore/'], '/pages/contact/'), null);
});

test('Nexora navigation: section tracking follows page order, including short final sections', () => {
  const sections = [{ id: 'business', top: 320 }, { id: 'about', top: -300 }, { id: 'innovation', top: 50 }];
  assert.equal(state.visibleSection(sections, 160, 700, false), 'innovation');
  assert.equal(state.visibleSection(sections, 160, 700, true), 'business');
  assert.equal(state.visibleSection([{ id: 'about', top: 400 }], 160, 700, false), '');
});

class Node {
  constructor(doc, props = {}) {
    this.doc = doc; this.parentElement = null; this.children = []; this.attrs = new Map(); this.dataset = {}; this.listeners = {}; this.inert = false; this.hidden = false; this.isConnected = true; this.open = false; this.rect = { top: 0, bottom: 100, left: 0, right: 400, height: 88 }; this.map = new Map();
    const classes = new Set();
    this.classList = { add: name => classes.add(name), remove: name => classes.delete(name), contains: name => classes.has(name), toggle(name, active) { if (active === undefined) active = !classes.has(name); active ? classes.add(name) : classes.delete(name); } };
    Object.assign(this, props);
  }
  add(...nodes) { nodes.forEach(node => { node.parentElement = this; this.children.push(node); }); return this; }
  contains(node) { return node === this || this.children.some(child => child.contains(node)); }
  setAttribute(name, value) { this.attrs.set(name, String(value)); }
  getAttribute(name) { return this.attrs.get(name) ?? null; }
  hasAttribute(name) { return this.attrs.has(name); }
  removeAttribute(name) { this.attrs.delete(name); }
  addEventListener(name, fn) { (this.listeners[name] ||= []).push(fn); }
  emit(name, props = {}) {
    const event = { target: this, button: 0, defaultPrevented: false, preventDefault() { this.defaultPrevented = true; }, ...props };
    this.listeners[name]?.forEach(fn => fn(event)); return event;
  }
  focus(options) { this.doc.activeElement = this; this.focusOptions = options; }
  select() { this.selected = true; }
  getBoundingClientRect() { return this.rect; }
  getClientRects() {
    for (let node = this; node; node = node.parentElement) {
      if (node.hidden || (node.mobileMenu && !node.classList.contains('is-open'))) return [];
      if (node.parentElement?.group && !node.parentElement.open && !node.summary) return [];
    }
    return [this.rect];
  }
  closest(selector) {
    for (let node = this; node; node = node.parentElement) {
      if (selector === '.nav-group' && node.group || selector === '.nav-group[open]' && node.group && node.open || selector === '[inert]' && node.inert || selector === 'a[href]' && node.hasAttribute('href')) return node;
    }
    return null;
  }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  querySelectorAll(selector) {
    if (this.map.has(selector)) return this.map.get(selector);
    const all = this.children.flatMap(child => [child, ...child.querySelectorAll('*')]);
    if (selector === '*') return all;
    if (selector === ':scope > summary') return this.children.filter(child => child.summary);
    if (selector === '.nav-group') return all.filter(child => child.group);
    if (selector === 'a[href]') return all.filter(child => child.hasAttribute('href'));
    if (selector === 'a[aria-current]') return all.filter(child => child.hasAttribute('href') && child.hasAttribute('aria-current'));
    if (selector.startsWith('a[href], button')) return all.filter(child => child.focusable);
    return [];
  }
}

function documentFixture() {
  const doc = new Node(null); doc.doc = doc; doc.body = new Node(doc); doc.documentElement = new Node(doc, { scrollHeight: 2000 }); doc.ids = new Map(); doc.getElementById = id => doc.ids.get(id);
  const win = new Node(doc, { location: { href: 'https://nexora.test/' }, scrollY: 0, innerHeight: 700 }); win.requestAnimationFrame = fn => fn();
  return { doc, win, node: props => new Node(doc, props), bind(selector, ...nodes) { doc.map.set(selector, nodes); } };
}

test('Nexora navigation: scroll changes leaf and ancestor selections together', () => {
  const f = documentFixture(), header = f.node(), menu = f.node(), group = f.node({ group: true }), home = f.node(), aboutLink = f.node(), about = f.node();
  home.setAttribute('href', '/'); aboutLink.setAttribute('href', '/#about'); group.add(aboutLink); menu.add(home, group); header.add(menu);
  f.bind('[data-header]', header); f.bind('#primary-nav', menu); f.bind('[data-hero]', f.node()); f.doc.ids.set('about', about); about.rect.top = 500;
  state.initNavigationState(f.doc, f.win);
  assert.equal(home.getAttribute('aria-current'), 'page'); assert.equal(aboutLink.getAttribute('aria-current'), null);
  f.win.scrollY = 450; about.rect.top = 50; f.win.emit('scroll');
  assert.equal(home.getAttribute('aria-current'), null); assert.equal(aboutLink.getAttribute('aria-current'), 'location'); assert.equal(group.classList.contains('is-active'), true); assert.equal(header.classList.contains('is-scrolled'), true);
  f.win.scrollY = 0; about.rect.top = 500; f.win.emit('scroll');
  assert.equal(home.getAttribute('aria-current'), 'page'); assert.equal(group.classList.contains('is-active'), false);
});

function menuFixture(mobile = true) {
  const f = documentFixture(), header = f.node(), menu = f.node({ mobileMenu: mobile }), toggle = f.node({ focusable: true }), brand = f.node({ focusable: true }), main = f.node(), footer = f.node({ inert: true });
  const root = f.node({ group: true }), summary = f.node({ focusable: true, summary: true }), dropdown = f.node(), child = f.node({ group: true }), childSummary = f.node({ focusable: true, summary: true }), childBody = f.node(), link = f.node({ focusable: true });
  link.setAttribute('href', '/pages/company/'); link.setAttribute('aria-current', 'page'); childBody.add(link); child.add(childSummary, childBody); dropdown.add(child); root.add(summary, dropdown); menu.add(root); header.add(brand, menu, toggle);
  const media = f.node({ matches: !mobile }); f.win.matchMedia = () => media;
  f.bind('[data-header]', header); f.bind('#primary-nav', menu); f.bind('[data-menu-toggle]', toggle); f.bind('main', main); f.bind('.site-footer', footer);
  const api = navigation.initNavigation(f.doc, f.win);
  return { ...f, header, menu, toggle, brand, main, footer, root, summary, child, childSummary, link, media, api };
}

test('Nexora navigation: a static header does not advance the selected section early', () => {
  const f = documentFixture(), header = f.node({ dataset: { sticky: 'false' } }), menu = f.node(), home = f.node(), link = f.node(), section = f.node();
  home.setAttribute('href', '/'); link.setAttribute('href', '/#about'); menu.add(home, link); header.add(menu);
  f.bind('[data-header]', header); f.bind('#primary-nav', menu); f.doc.ids.set('about', section); section.rect.top = 150;
  state.initNavigationState(f.doc, f.win);
  assert.equal(home.getAttribute('aria-current'), 'page');
  section.rect.top = 100; f.win.emit('scroll'); assert.equal(link.getAttribute('aria-current'), 'location');
});

function autoplayFixture(options = {}) {
  const f = documentFixture(), timers = new Map(); let id = 0;
  const hero = f.node({ dataset: { autoplay: 'true', interval: '7', motion: options.disabled ? 'false' : 'true' } });
  const slides = [f.node(), f.node({hidden:true}), f.node({hidden:true})], playback = f.node(), status = f.node({textContent:''});
  const controls = f.node(), buttons = slides.map((_,i)=>f.node({dataset:{slideTo:String(i)}}));
  hero.map.set('[data-slide]',slides); hero.map.set('[data-slide-to]',buttons); hero.map.set('[data-slide-playback]',[playback]);
  hero.map.set('[data-slide-status]',[status]); hero.map.set('[data-hero-controls]',[controls]); f.bind('[data-hero]',hero);
  f.win.setTimeout = (fn, delay) => { timers.set(++id, {fn,delay}); return id; }; f.win.clearTimeout = id => timers.delete(id);
  const media = f.node({matches:!!options.reduced}); f.win.matchMedia = () => media;
  let intersection; f.win.IntersectionObserver = class { constructor(fn){intersection=fn;} observe(){} };
  carousel.initCarousel(f.doc,f.win);
  return {...f,hero,slides,playback,status,controls,buttons,media,timers, visible(value){intersection([{isIntersecting:value}]);}, next(){const [key,timer]=timers.entries().next().value;timers.delete(key);timer.fn();}};
}

test('Nexora carousel: timed slides are silent and pause for hover, background tabs and offscreen content', () => {
  const f = autoplayFixture(); assert.equal(f.timers.size,1); assert.equal([...f.timers.values()][0].delay,7000);
  f.next(); assert.equal(f.slides[1].hidden,false); assert.equal(f.status.textContent,'');
  f.hero.emit('mouseenter'); assert.equal(f.timers.size,0); f.hero.emit('mouseleave'); assert.equal(f.timers.size,1);
  f.doc.hidden=true; f.doc.emit('visibilitychange'); assert.equal(f.timers.size,0); f.doc.hidden=false; f.doc.emit('visibilitychange'); assert.equal(f.timers.size,1);
  f.visible(false); assert.equal(f.timers.size,0); f.visible(true); assert.equal(f.timers.size,1);
  f.playback.emit('click'); assert.equal(f.timers.size,0); assert.equal(f.playback.textContent,'播放轮播');
});

test('Nexora carousel: keyboard focus and manual navigation stop autoplay until explicitly restarted', () => {
  const f=autoplayFixture(); f.hero.emit('focusin'); assert.equal(f.timers.size,0); f.hero.emit('focusout'); assert.equal(f.timers.size,0);
  f.playback.emit('click'); assert.equal(f.timers.size,1); f.buttons[2].emit('click'); assert.equal(f.timers.size,0); assert.equal(f.slides[2].hidden,false); assert.equal(f.status.textContent,'第 3 张，共 3 张');
});

test('Nexora carousel: pointer pause survives focusin and subsequent clicks can resume and pause again', () => {
  const f = autoplayFixture();
  assert.equal(f.playback.textContent, '暂停轮播');
  f.hero.emit('mouseenter');
  f.playback.emit('pointerdown', { button: 0, isPrimary: true });
  f.hero.emit('focusin', { target: f.playback });
  f.playback.emit('click', { detail: 1 });
  f.hero.emit('mouseleave');
  assert.equal(f.playback.textContent, '播放轮播');
  assert.equal(f.timers.size, 0, 'the first pointer click must remain paused after leaving');

  f.hero.emit('mouseenter');
  f.playback.emit('pointerdown', { button: 0, isPrimary: true });
  f.playback.emit('click', { detail: 1 });
  assert.equal(f.playback.textContent, '暂停轮播');
  assert.equal(f.timers.size, 0, 'hover still suspends the resumed timer');
  f.hero.emit('mouseleave');
  assert.equal(f.timers.size, 1);

  f.playback.emit('pointerdown', { button: 0, isPrimary: true });
  f.playback.emit('click', { detail: 1 });
  assert.equal(f.playback.textContent, '播放轮播');
  assert.equal(f.timers.size, 0);
});

test('Nexora carousel: cancelled pointers and keyboard activation discard stale pointer intent', () => {
  const f = autoplayFixture();
  f.playback.emit('pointerdown', { button: 0, isPrimary: true });
  f.hero.emit('focusin', { target: f.playback });
  f.playback.emit('pointercancel');
  f.playback.emit('click', { detail: 1 });
  assert.equal(f.timers.size, 1, 'a cancelled gesture must not override a later activation');

  f.playback.emit('pointerdown', { button: 0, isPrimary: true });
  f.hero.emit('focusin', { target: f.playback });
  f.playback.emit('click', { detail: 0 });
  assert.equal(f.timers.size, 1, 'keyboard activation explicitly resumes after focus pauses');
  f.playback.emit('click', { detail: 0 });
  assert.equal(f.timers.size, 0, 'keyboard activation can also pause playback');
});

test('Nexora carousel: system and theme motion preferences prevent playback, including live changes', () => {
  for (const options of [{reduced:true},{disabled:true}]) { const f=autoplayFixture(options); assert.equal(f.timers.size,0); assert.equal(f.playback.disabled,true); f.playback.emit('click'); assert.equal(f.timers.size,0); }
  const f=autoplayFixture(); f.media.matches=true; f.media.emit('change'); assert.equal(f.timers.size,0); f.media.matches=false; f.media.emit('change'); assert.equal(f.timers.size,0); f.playback.emit('click'); assert.equal(f.timers.size,1);
});

test('Nexora mobile menu: opens the selected nested branch, traps focus, restores existing inert state', () => {
  const f = menuFixture(); f.toggle.emit('click');
  assert.equal(f.toggle.getAttribute('aria-expanded'), 'true'); assert.equal(f.doc.activeElement, f.link); assert.equal(f.root.open, true); assert.equal(f.child.open, true); assert.equal(f.main.inert, true);
  f.root.emit('toggle'); assert.equal(f.child.open, true, 'opening an ancestor must not close its selected descendant');
  f.toggle.focus(); const tab = f.doc.emit('keydown', { key: 'Tab' }); assert.equal(tab.defaultPrevented, true); assert.equal(f.doc.activeElement, f.brand);
  f.api.close(); assert.equal(f.main.inert, false); assert.equal(f.footer.inert, true); assert.equal(f.doc.body.classList.contains('menu-open'), false);
});

test('Nexora menu: Escape closes the deepest disclosure first and then returns focus to the trigger', () => {
  const f = menuFixture(); f.toggle.emit('click');
  f.doc.emit('keydown', { key: 'Escape' }); assert.equal(f.child.open, false); assert.equal(f.root.open, true); assert.equal(f.doc.activeElement, f.childSummary);
  f.doc.emit('keydown', { key: 'Escape' }); assert.equal(f.root.open, false); assert.equal(f.doc.activeElement, f.summary);
  f.doc.emit('keydown', { key: 'Escape' }); assert.equal(f.toggle.getAttribute('aria-expanded'), 'false'); assert.equal(f.doc.activeElement, f.toggle);
  assert.equal(f.toggle.focusOptions?.preventScroll, true, 'restoring focus after a sticky menu closes must preserve the reading position');
});

test('Nexora menu: crossing to desktop releases the page and modified links keep native behavior', () => {
  const f = menuFixture(); f.toggle.emit('click'); f.menu.emit('click', { target: f.link, ctrlKey: true }); assert.equal(f.toggle.getAttribute('aria-expanded'), 'true');
  f.media.matches = true; f.media.emit('change'); assert.equal(f.main.inert, false); assert.equal(f.toggle.getAttribute('aria-expanded'), 'false');
});

test('Nexora carousel: previous wraps, dots and live status agree, keyboard Home/End keep focus visible', () => {
  const f = documentFixture(), hero = f.node(), controls = f.node({ hidden: true }), count = f.node(), status = f.node();
  const slides = Array.from({ length: 3 }, (_, i) => f.node({ hidden: i > 0 }));
  const buttons = slides.map((_, i) => { const node = f.node({ dataset: { slideTo: String(i) } }); node.setAttribute('data-slide-to', String(i)); return node; });
  const previous = f.node({ dataset: { slideStep: '-1' } });
  hero.map.set('[data-slide]', slides); hero.map.set('[data-slide-to]', buttons); hero.map.set('[data-hero-controls]', [controls]); hero.map.set('[data-current-slide]', [count]); hero.map.set('[data-slide-status]', [status]); hero.map.set('[data-slide-step]', [previous]); f.bind('[data-hero]', hero);
  carousel.initCarousel(f.doc); previous.emit('click');
  assert.equal(controls.hidden, false); assert.equal(slides[2].hidden, false); assert.equal(count.textContent, '03'); assert.equal(buttons[2].getAttribute('aria-pressed'), 'true'); assert.equal(status.textContent, '第 3 张，共 3 张');
  controls.emit('keydown', { key: 'Home', target: buttons[2] }); assert.equal(count.textContent, '01'); assert.equal(f.doc.activeElement, buttons[0]);
  controls.emit('keydown', { key: 'End', target: buttons[0] }); assert.equal(count.textContent, '03'); assert.equal(f.doc.activeElement, buttons[2]);
  assert.equal(carousel.wrappedSlide(-7, 3), 2); assert.equal(carousel.wrappedSlide(NaN, 3), 0);
});

test('Nexora search: query is prefilled, only an outside press and release closes, focus is restored', () => {
  const f = documentFixture(), dialog = f.node(), trigger = f.node(), input = f.node({ value: '' }), form = f.node(), close = f.node();
  f.win.location.href = 'https://nexora.test/explore/?q=technology'; let closedMenus = 0;
  dialog.showModal = () => { dialog.open = true; }; dialog.close = () => { dialog.open = false; dialog.emit('close'); };
  dialog.map.set('input[type="search"]', [input]); dialog.map.set('form', [form]); dialog.map.set('[data-search-close]', [close]); f.bind('#site-search', dialog); f.bind('[data-search-open]', trigger);
  search.initSearch(f.doc, f.win, { close() { closedMenus++; } });
  assert.equal(trigger.emit('click', { ctrlKey: true }).defaultPrevented, false); assert.equal(dialog.open, false);
  trigger.emit('click'); assert.equal(dialog.open, true); assert.equal(input.value, 'technology'); assert.equal(closedMenus, 1);
  input.value = '   '; input.reportValidity = () => { input.reported = true; };
  assert.equal(form.emit('submit').defaultPrevented, true); assert.equal(input.value, ''); assert.equal(input.reported, true);
  dialog.emit('pointerdown', { clientX: 30, clientY: 30 }); dialog.emit('click', { clientX: 30, clientY: 150 }); assert.equal(dialog.open, true, 'dragging from the panel onto the backdrop must not dismiss');
  dialog.emit('pointerdown', { clientX: 30, clientY: 150 }); dialog.emit('click', { clientX: 30, clientY: 150 }); assert.equal(dialog.open, false); assert.equal(f.doc.activeElement, trigger); assert.equal(f.doc.body.classList.contains('search-open'), false);
});
