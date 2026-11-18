const test = require('node:test');
const assert = require('node:assert/strict');
const appearance = import('../../sources/docshub/assets/js/modules/appearance-state.mjs');

test('DocsHub appearance honors administrator locks, preview and invalid storage', async () => {
  const { colorPreference } = await appearance;
  assert.equal(colorPreference('light', 'dark'), 'dark');
  assert.equal(colorPreference('dark', 'light', false), 'dark');
  assert.equal(colorPreference('system', 'dark', true, true), 'system');
  assert.equal(colorPreference('system', 'old-invalid-value'), 'system');
  assert.equal(colorPreference('unknown', null), 'light');
  assert.equal(colorPreference('dark', null), 'dark');
});

test('DocsHub automatic preference follows system while explicit choices remain stable', async () => {
  const { resolvedColorMode } = await appearance;
  assert.equal(resolvedColorMode('system', true), 'dark');
  assert.equal(resolvedColorMode('system', false), 'light');
  assert.equal(resolvedColorMode('light', true), 'light');
  assert.equal(resolvedColorMode('dark', false), 'dark');
});

test('DocsHub runtime synchronizes system changes, cross-tab removal and saved choices', async () => {
  const { initAppearance } = await import('../../sources/docshub/assets/js/modules/appearance.mjs');
  const root = { dataset: { defaultMode: 'system', allowColorSwitch: 'true', appearancePreview: 'false' } };
  const controls = { hidden: true };
  const listeners = {};
  const system = { matches: false, addEventListener(name, listener) { this[name] = listener; } };
  const saved = new Map();
  const meta = { setAttribute(name, value) { this[name] = value; } };
  const buttons = ['light', 'dark', 'system'].map(mode => ({
    dataset: { colorMode: mode },
    setAttribute(name, value) { this[name] = value; },
    addEventListener(name, listener) { this[name] = listener; }
  }));
  const originals = Object.fromEntries(['document', 'window', 'localStorage'].map(name => [name, Object.getOwnPropertyDescriptor(globalThis, name)]));
  Object.assign(globalThis, {
    document: {
      documentElement: root,
      querySelector(selector) { return selector === '[data-appearance-controls]' ? controls : meta; },
      querySelectorAll() { return buttons; }
    },
    window: { matchMedia() { return system; }, addEventListener(name, listener) { listeners[name] = listener; } },
    localStorage: { getItem(key) { return saved.get(key) ?? null; }, setItem(key, value) { saved.set(key, value); } }
  });
  try {
    initAppearance();
    assert.equal(root.dataset.mode, 'light');
    assert.equal(controls.hidden, false);
    assert.equal(buttons[2]['aria-pressed'], 'true');
    system.matches = true;
    system.change();
    assert.equal(root.dataset.mode, 'dark');
    assert.equal(buttons[2]['aria-pressed'], 'true');
    buttons[0].click();
    assert.equal(root.dataset.mode, 'light');
    assert.equal(saved.get('docshub:mode'), 'light');
    system.change();
    assert.equal(root.dataset.mode, 'light');
    listeners.storage({ key: 'docshub:mode', newValue: 'dark' });
    assert.equal(root.dataset.mode, 'dark');
    assert.equal(buttons[1]['aria-pressed'], 'true');
    listeners.storage({ key: 'docshub:mode', newValue: null });
    assert.equal(root.dataset.colorPreference, 'system');
    assert.equal(buttons[2]['aria-pressed'], 'true');
    assert.equal(meta.content, '#111523');
  } finally {
    for (const [name, descriptor] of Object.entries(originals)) {
      if (descriptor) Object.defineProperty(globalThis, name, descriptor);
      else delete globalThis[name];
    }
  }
});
