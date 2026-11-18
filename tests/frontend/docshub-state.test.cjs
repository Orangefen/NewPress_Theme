const test = require('node:test');
const assert = require('node:assert/strict');
const state = import('../../sources/docshub/assets/js/modules/state.mjs');

test('DocsHub selects search identity instead of every link to Explore', async () => {
  const { navigationMatch: match } = await state;
  const current = 'https://docs.example/explore/?page=2&q=API&sort=title&view=list';
  assert.ok(match('/explore/?q=API', current).score > match('/explore/', current).score);
  assert.equal(match('/explore/?q=API', current).current, 'page');
  assert.equal(match('/explore/', current).current, 'location');
  assert.equal(match('/explore/?q=入门', current), null);
  assert.equal(match('/', current), null);
});

test('DocsHub handles URL normalization, query order, origins and route boundaries', async () => {
  const { navigationMatch: match } = await state;
  assert.equal(match('/explore/?category=入门&q=api', 'https://docs.example/explore?q=API&category=%E5%85%A5%E9%97%A8').current, 'page');
  assert.equal(match('https://elsewhere.example/explore/', 'https://docs.example/explore/'), null);
  assert.equal(match('javascript:alert(1)', 'https://docs.example/'), null);
  assert.equal(match('/posts/start', 'https://docs.example/posts/starter'), null);
  assert.equal(match('/series/', 'https://docs.example/series/developer/').current, 'location');
  assert.equal(match('/posts/start', 'https://docs.example/posts/start/').current, 'page');
  assert.equal(match('/categories/%e5%85%a5%e9%97%a8/', 'https://docs.example/categories/%E5%85%A5%E9%97%A8/').current, 'page');
});

test('DocsHub anchor navigation outranks its page and does not select another section', async () => {
  const { navigationMatch: match } = await state;
  const current = 'https://docs.example/#tutorials';
  assert.ok(match('/#tutorials', current).score > match('/', current).score);
  assert.equal(match('/#other', current), null);
});

test('DocsHub filters retain search, sort and view while resetting pagination', async () => {
  const { discoveryURL } = await state;
  const current = 'https://docs.example/explore/?q=C%2B%2B&category=old&sort=title&view=list&page=4#results';
  const result = new URL(discoveryURL(current, { category: '入门' }), current);
  assert.equal(result.pathname, '/explore/');
  assert.deepEqual(Object.fromEntries(result.searchParams), { q: 'C++', category: '入门', sort: 'title', view: 'list' });
  assert.equal(result.hash, '');
  assert.equal(new URL(discoveryURL(current, { q: '' }), current).searchParams.has('q'), false);
  assert.equal(new URL(discoveryURL('https://docs.example/authors/2/?q=hello', { view: 'list' }), current).pathname, '/authors/2/');
});

test('DocsHub directory updates at section boundaries and short final sections', async () => {
  const { activeHeadingIndex } = await state;
  assert.equal(activeHeadingIndex([], 112), -1);
  assert.equal(activeHeadingIndex([200, 400, 800], 112), 0);
  assert.equal(activeHeadingIndex([-600, 110, 500], 112), 1);
  assert.equal(activeHeadingIndex([-600, 110, 500], 112, true), 2);
});

test('DocsHub headings keep stable IDs without collisions with content or shell', async () => {
  const { headingIDs } = await state;
  const ids = headingIDs(['intro', 'intro', '', 'section-3', 'main'], ['main', 'section-2']);
  assert.equal(ids[0], 'intro');
  assert.equal(ids[3], 'section-3');
  assert.equal(new Set([...ids, 'main', 'section-2']).size, 7);
  assert.deepEqual(headingIDs(['', '']), ['section-1', 'section-2']);
});

test('DocsHub URL view wins over the configured default with invalid values falling back', async () => {
  const { resolveListView } = await state;
  assert.equal(resolveListView(null, 'list'), 'list');
  assert.equal(resolveListView('', 'list'), 'list');
  assert.equal(resolveListView('unknown', 'list'), 'list');
  assert.equal(resolveListView('grid', 'list'), 'grid');
  assert.equal(resolveListView('list', 'grid'), 'list');
  assert.equal(resolveListView(null, 'unknown'), 'grid');
});

test('DocsHub TOC depth includes only the requested heading levels', async () => {
  const { tocHeadingSelector } = await state;
  assert.equal(tocHeadingSelector('h2'), 'h2');
  assert.equal(tocHeadingSelector('h3'), 'h2,h3');
  assert.equal(tocHeadingSelector('h4'), 'h2,h3,h4');
  assert.equal(tocHeadingSelector('h1,script'), 'h2,h3');
});
