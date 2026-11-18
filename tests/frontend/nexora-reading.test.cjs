const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const { buildSync } = require('esbuild');

const moduleObject = { exports: {} };
vm.runInNewContext(buildSync({ entryPoints: ['sources/nexora/assets/js/modules/reading.js'], bundle: true, write: false, format: 'cjs' }).outputFiles[0].text, { module: moduleObject, exports: moduleObject.exports });
const { initReading, initCommentReplies } = moduleObject.exports;

class Element {
  constructor(props = {}) { this.dataset = {}; this.attributes = new Map(); this.children = []; this.listeners = {}; this.hidden = true; this.id = ''; this.textContent = ''; this.top = 0; Object.assign(this, props); }
  querySelector(selector) { return this.selectors?.[selector] || null; }
  querySelectorAll(selector) { return this.selectors?.[selector] || []; }
  addEventListener(name, callback) { (this.listeners[name] ||= []).push(callback); }
  fire(name) { return Promise.all((this.listeners[name] || []).map(callback => callback({ preventDefault() {} }))); }
  append(child) { this.children.push(child); }
  getBoundingClientRect() { return { top: this.top, height: 80 }; }
  setAttribute(key, value) { this.attributes.set(key, value); }
  hasAttribute(key) { return this.attributes.has(key); }
  removeAttribute(key) { this.attributes.delete(key); }
  focus() { this.focused = true; }
  scrollIntoView(options) { this.scrolled = true; this.scrollOptions = options; }
}

test('Nexora reading preserves existing anchors, avoids duplicate IDs and tolerates malformed links', async () => {
  const existing = new Element({ id: 'main' });
  const heading = new Element({ id: 'main', tagName: 'H2', textContent: '正文标题', top: -10 });
  const subheading = new Element({ id: 'existing-section', tagName: 'H3', textContent: '内容细节', top: 600 });
  const article = new Element({ textContent: '正文内容', selectors: { 'h2, h3': [heading, subheading] } });
  const toc = new Element();
  const list = new Element();
  const copy = new Element();
  const status = new Element();
  const doc = new Element({ selectors: { '[data-article-body]': article, '[data-reading-toc]': toc, '[data-reading-toc-list]': list, '[data-copy-link]': copy, '[data-copy-status]': status } });
  const win = new Element({ location: { hash: '#%ZZ', href: 'https://nexora.test/posts/article' }, navigator: { clipboard: { writeText: async () => { throw new Error('denied'); } } }, requestAnimationFrame: callback => callback() });
  doc.defaultView = win;
  doc.createElement = tagName => new Element({ tagName: tagName.toUpperCase() });
  doc.getElementById = id => [existing, heading, subheading].find(element => element.id === id);
  initReading(doc);
  initReading(doc);
  assert.notEqual(heading.id, 'main', 'document IDs must not be duplicated');
  assert.equal(subheading.id, 'existing-section', 'existing deep links must remain valid');
  assert.equal(toc.hidden, false);
  assert.equal(toc.open, true, 'the existing default keeps the TOC expanded');
  assert.equal(list.children.length, 2, 'repeated initialization cannot duplicate the TOC');
  assert.equal(list.children[0].children[0].attributes.get('aria-current'), 'location');
  await list.children[1].children[0].fire('click');
  assert.equal(subheading.focused, true, 'TOC navigation moves keyboard focus into the article');
  subheading.top = 30;
  await win.fire('scroll');
  assert.equal(list.children[1].children[0].attributes.get('aria-current'), 'location');
  assert.equal(list.children[0].children[0].attributes.has('aria-current'), false);
  await copy.fire('click');
  assert.equal(copy.disabled, false);
  assert.match(status.textContent, /复制失败/);
});

test('Nexora reading honors a collapsed TOC without disabling reading time or sharing', async () => {
  const headings = [new Element({ id: 'section-a', tagName: 'H2', textContent: '章节一', top: -20 }), new Element({ id: 'section-b', tagName: 'H3', textContent: '章节二', top: 100 })];
  const article = new Element({ textContent: '阅读内容'.repeat(200), selectors: { 'h2, h3': headings } });
  const toc = new Element({ dataset: { tocOpen: 'false' } });
  const list = new Element();
  const duration = new Element();
  const copy = new Element();
  const status = new Element();
  const canonical = new Element({ href: 'https://nexora.test/posts/canonical' });
  let copied = '';
  const win = new Element({ location: { hash: '#section-b', href: 'https://nexora.test/posts/article#section-b' }, navigator: { clipboard: { writeText: async value => { copied = value; } } }, requestAnimationFrame: callback => callback() });
  const header = new Element({ dataset: { sticky: 'false' } });
  const doc = new Element({ defaultView: win, body: { classList: { contains: name => name === 'nexora-no-motion' } }, selectors: { '[data-article-body]': article, '[data-reading-toc]': toc, '[data-reading-toc-list]': list, '[data-reading-time]': duration, '[data-copy-link]': copy, '[data-copy-status]': status, '[data-header]': header, 'link[rel="canonical"]': canonical } });
  doc.createElement = tagName => new Element({ tagName: tagName.toUpperCase() });
  doc.getElementById = id => headings.find(heading => heading.id === id);
  initReading(doc);
  assert.equal(toc.hidden, false);
  assert.equal(toc.open, false);
  assert.equal(list.children.length, 2);
  assert.equal(list.children[0].children[0].attributes.get('aria-current'), 'location', 'a static header must not offset reading position');
  assert.equal(headings[1].scrollOptions.behavior, 'instant', 'disabled motion also applies to deep-link scrolling');
  assert.equal(duration.hidden, false);
  assert.equal(duration.textContent, '约 2 分钟阅读');
  await copy.fire('click');
  assert.equal(copied, canonical.href);
  assert.equal(status.textContent, '链接已复制');
  assert.equal(copy.disabled, false);
});

test('Nexora reading keeps sharing usable when the TOC and duration are disabled', async () => {
  const article = new Element({ textContent: '正文', selectors: { 'h2, h3': [] } });
  const copy = new Element();
  let copied = '';
  const win = new Element({ location: { hash: '', href: 'https://nexora.test/posts/article#section' }, navigator: { clipboard: { writeText: async value => { copied = value; } } } });
  const doc = new Element({ defaultView: win, selectors: { '[data-article-body]': article, '[data-copy-link]': copy } });
  initReading(doc);
  assert.equal(copy.hidden, false);
  await copy.fire('click');
  assert.equal(copied, 'https://nexora.test/posts/article');
  assert.equal(copy.disabled, false);
});

test('Nexora replies retain a rejected reply and cancel without clearing the written comment', async () => {
  const parent = new Element({ value: '7' });
  const content = new Element({ value: '已经写好的回复内容' });
  const context = new Element();
  const label = new Element();
  const cancel = new Element();
  const button = new Element({ dataset: { replyId: '7', replyAuthor: '读者甲' } });
  const invalid = new Element({ dataset: { replyId: 'not-an-id', replyAuthor: '无效' } });
  const form = new Element({ selectors: { '[name="parent_id"]': parent, '[name="content"]': content, '[data-reply-context]': context, '[data-reply-label]': label, '[data-reply-cancel]': cancel } });
  const doc = new Element({ selectors: { '[data-comment-form]': form, '[data-comment-reply]': [button, invalid] } });
  initCommentReplies(doc);
  initCommentReplies(doc);
  assert.equal(context.hidden, false);
  assert.equal(label.textContent, '回复 读者甲');
  await cancel.fire('click');
  assert.equal(parent.value, '0');
  assert.equal(context.hidden, true);
  assert.equal(content.value, '已经写好的回复内容');
  assert.equal(button.focused, true);
  await invalid.fire('click');
  assert.equal(parent.value, '0');
  await button.fire('click');
  assert.equal(parent.value, '7');
  assert.equal(content.focused, true);
  assert.equal(button.listeners.click.length, 1);
});
