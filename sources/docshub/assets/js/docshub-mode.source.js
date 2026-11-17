/* Apply preference before first paint; storage can be unavailable in private contexts. */
(() => {
  let mode = 'light';
  try { if (localStorage.getItem('docshub:mode') === 'dark') mode = 'dark'; } catch {}
  document.documentElement.dataset.mode = mode;
})();
