/**
 * Unit tests for history item rendering helpers.
 * Run with:  node ui/history.test.js
 *
 * No external dependencies — uses a minimal DOM stub and Node's built-in assert.
 */

'use strict';

const assert = require('assert');

// ─── Minimal DOM stub ────────────────────────────────────────────────────────
class FakeEl {
  constructor(tag) {
    this.tagName     = tag.toUpperCase();
    this.className   = '';
    this.textContent = '';
    this.title       = '';
    this._innerHTML  = '';
    this.children    = [];
    this._listeners  = {};
    this.classList   = {
      _set: new Set(),
      add:    (...c) => c.forEach(x => this.classList._set.add(x)),
      remove: (...c) => c.forEach(x => this.classList._set.delete(x)),
      toggle: (c, force) => {
        if (force === undefined) force = !this.classList._set.has(c);
        force ? this.classList._set.add(c) : this.classList._set.delete(c);
        return force;
      },
      contains: (c) => this.classList._set.has(c),
    };
  }
  get innerHTML() { return this._innerHTML; }
  set innerHTML(v) { this._innerHTML = v; }
  appendChild(child) {
    child._parent = this;   // track parent for closest() traversal
    this.children.push(child);
    return child;
  }
  addEventListener(evt, fn) {
    if (!this._listeners[evt]) this._listeners[evt] = [];
    this._listeners[evt].push(fn);
  }
  closest(sel) {
    const cls = sel.replace(/^\./, '');
    let cur = this;
    while (cur) {
      if (cur.classList && cur.classList.contains && cur.classList.contains(cls)) return cur;
      cur = cur._parent;
    }
    return null;
  }
  // Simulate a click — target is self or a child element
  _click(targetEl) {
    targetEl = targetEl || this;
    const e = {
      target: targetEl,
      stopPropagation: () => {},
    };
    let cur = targetEl;
    while (cur) {
      (cur._listeners['click'] || []).forEach(fn => fn(e));
      cur = cur._parent;
    }
  }
}

const document = {
  createElement: (tag) => new FakeEl(tag),
};

// ─── Pure helpers copied from main.js ────────────────────────────────────────

function escapeHtml(s) {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function formatTimeAgo(dateStr) {
  const d = new Date(dateStr);
  if (isNaN(d)) return '';
  const diff = Date.now() - d.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1)  return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24)  return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function inlineMd(text) {
  text = escapeHtml(text);
  text = text.replace(/`([^`\n]+)`/g, '<code>$1</code>');
  text = text.replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>');
  text = text.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  text = text.replace(/__(.+?)__/g, '<strong>$1</strong>');
  text = text.replace(/\*([^*\n]+?)\*/g, '<em>$1</em>');
  text = text.replace(/_([^_\n]+?)_/g, '<em>$1</em>');
  text = text.replace(/~~(.+?)~~/g, '<del>$1</del>');
  return text;
}

function renderMarkdown(raw, _depth = 0) {
  if (_depth > 4) return escapeHtml(raw);

  const fences = [];
  const fenceLines = raw.split('\n');
  const processedLines = [];
  let inFence = false, fenceLang = '', fenceContent = [];
  for (const fl of fenceLines) {
    if (!inFence) {
      const fm = fl.match(/^```([\w.-]*)$/);
      if (fm) { inFence = true; fenceLang = fm[1]; fenceContent = []; }
      else { processedLines.push(fl); }
    } else {
      if (fl === '```') {
        fences.push({ lang: fenceLang, code: fenceContent.join('\n') });
        processedLines.push(`\x01F${fences.length - 1}\x01`);
        inFence = false;
      } else { fenceContent.push(fl); }
    }
  }
  if (inFence) {
    fences.push({ lang: fenceLang, code: fenceContent.join('\n') });
    processedLines.push(`\x01F${fences.length - 1}\x01`);
  }
  let md = processedLines.join('\n');
  const lines = md.split('\n');
  const out   = [];
  let i       = 0;

  const flushFence = (token) => {
    const idx = parseInt(token.match(/\d+/)[0]);
    const { lang, code } = fences[idx];
    const label = lang ? `<span class="md-code-lang">${escapeHtml(lang)}</span>` : '';
    return `<pre class="md-pre">${label}<code class="md-code-block">${escapeHtml(code)}</code></pre>`;
  };

  while (i < lines.length) {
    const line = lines[i];
    if (/^\x01F\d+\x01$/.test(line.trim())) { out.push(flushFence(line.trim())); i++; continue; }
    const hm = line.match(/^(#{1,6}) (.+)/);
    if (hm) { const lvl = Math.min(hm[1].length, 6); out.push(`<h${lvl} class="md-h${lvl}">${inlineMd(hm[2].trim())}</h${lvl}>`); i++; continue; }
    if (/^[-*_]{3,}$/.test(line.trim())) { out.push('<hr class="md-hr">'); i++; continue; }
    if (line.startsWith('> ')) {
      const qlines = [];
      while (i < lines.length && lines[i].startsWith('> ')) { qlines.push(lines[i].slice(2)); i++; }
      out.push(`<blockquote class="md-blockquote">${renderMarkdown(qlines.join('\n'), _depth + 1)}</blockquote>`);
      continue;
    }
    if (/^[-*+] /.test(line)) {
      const items = [];
      while (i < lines.length && /^[-*+] /.test(lines[i])) { items.push(`<li>${inlineMd(lines[i].replace(/^[-*+] /, ''))}</li>`); i++; }
      out.push(`<ul class="md-ul">${items.join('')}</ul>`);
      continue;
    }
    if (/^\d+[.)]\s/.test(line)) {
      const items = [];
      while (i < lines.length && /^\d+[.)]\s/.test(lines[i])) { items.push(`<li>${inlineMd(lines[i].replace(/^\d+[.)]\s/, ''))}</li>`); i++; }
      out.push(`<ol class="md-ol">${items.join('')}</ol>`);
      continue;
    }
    if (line.trim() === '') { i++; continue; }
    const plines = [];
    while (i < lines.length) {
      const l = lines[i];
      if (l.trim() === '') break;
      if (/^\x01F\d+\x01$/.test(l.trim())) break;
      if (/^#{1,6} /.test(l)) break;
      if (/^[-*_]{3,}$/.test(l.trim())) break;
      if (/^[-*+] /.test(l)) break;
      if (/^\d+[.)]\s/.test(l)) break;
      if (l.startsWith('> ')) break;
      plines.push(inlineMd(l));
      i++;
    }
    if (plines.length) out.push(`<p class="md-p">${plines.join('<br>')}</p>`);
  }
  return out.join('');
}

async function writeToClipboard() {} // stub

function buildHistoryTurnEl(r, label) {
  const responseText = (r.response     || '').trim();
  const finalPrompt  = (r.final_prompt || '').trim();
  const preview      = responseText.slice(0, 100).replace(/\n+/g, ' ') || '(no response)';

  const safeRecipeId = escapeHtml(r.prompt_id || '\u2014');
  const safeEngine   = escapeHtml(r.engine    || '');
  const safeTime     = escapeHtml(formatTimeAgo(r.created_at));
  const safeLabel    = label ? escapeHtml(label) : '';

  const el = document.createElement('div');
  el.className = 'history-item';

  const header = document.createElement('div');
  header.className = 'history-item-header';
  header.innerHTML = `
    <span class="history-recipe">${safeRecipeId}</span>
    ${safeLabel ? `<span class="history-turn-badge">${safeLabel}</span>` : ''}
    <span class="history-engine">${safeEngine}</span>
    <span class="history-time">${safeTime}</span>
  `;

  const previewEl = document.createElement('div');
  previewEl.className = 'history-preview';
  previewEl.textContent = preview;

  const detail = document.createElement('div');
  detail.className = 'history-detail';

  if (finalPrompt) {
    const inputLabel = document.createElement('div');
    inputLabel.className = 'history-input-label';
    inputLabel.textContent = 'Prompt sent';
    const inputText = document.createElement('div');
    inputText.className = 'history-input-text';
    inputText.textContent = finalPrompt;
    detail.appendChild(inputLabel);
    detail.appendChild(inputText);
  }

  const responseHeader = document.createElement('div');
  responseHeader.className = 'history-response-header';
  const responseLabel = document.createElement('span');
  responseLabel.className = 'history-response-label';
  responseLabel.textContent = 'Response';
  responseHeader.appendChild(responseLabel);

  let copyBtn = null;
  if (responseText) {
    copyBtn = document.createElement('button');
    copyBtn.className = 'history-copy-btn';
    copyBtn.title = 'Copy response';
    copyBtn.textContent = 'Copy';
    copyBtn.addEventListener('click', async (e) => {
      e.stopPropagation();
      await writeToClipboard(responseText);
      copyBtn.textContent = '\u2713 Copied';
      setTimeout(() => { copyBtn.textContent = 'Copy'; }, 1800);
    });
    responseHeader.appendChild(copyBtn);
  }

  const responseText_el = document.createElement('div');
  responseText_el.className = 'history-response-text';
  responseText_el.innerHTML = responseText ? renderMarkdown(responseText) : '<em>(empty)</em>';

  detail.appendChild(responseHeader);
  detail.appendChild(responseText_el);
  el.appendChild(header);
  el.appendChild(previewEl);
  el.appendChild(detail);

  // Set _parent on children added via innerHTML/manual append so stub closest() works on all nodes
  [header, previewEl, detail].forEach(c => c._parent = el);
  detail.children.forEach(c => c._parent = detail);

  el.addEventListener('click', (e) => {
    // If click originated inside the detail panel → do nothing
    const targetNode = e.target;
    if (targetNode && typeof targetNode.closest === 'function' && targetNode.closest('.history-detail')) return;
    el.classList.toggle('expanded');
  });

  return el;
}

// ─── Test runner ─────────────────────────────────────────────────────────────
let passed = 0, failed = 0;
function test(name, fn) {
  try {
    fn();
    console.log(`  \u2705  ${name}`);
    passed++;
  } catch (e) {
    console.error(`  \u274c  ${name}`);
    console.error(`       ${e.message}`);
    failed++;
  }
}

// ═══════════════════════════════════════════════════════════════════════════
// SUITE 1: escapeHtml
// ═══════════════════════════════════════════════════════════════════════════
console.log('\n\u2500\u2500 escapeHtml \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500');
test('escapes & < > "', () => {
  assert.strictEqual(escapeHtml('a & b < c > d "e"'), 'a &amp; b &lt; c &gt; d &quot;e&quot;');
});
test('leaves safe strings unchanged', () => {
  assert.strictEqual(escapeHtml('hello world'), 'hello world');
});
test('handles empty string', () => {
  assert.strictEqual(escapeHtml(''), '');
});
test('XSS script tag is escaped', () => {
  const result = escapeHtml('<script>alert(1)</script>');
  assert.ok(!result.includes('<script>'), 'should not contain raw <script>');
  assert.ok(result.includes('&lt;script&gt;'));
});

// ═══════════════════════════════════════════════════════════════════════════
// SUITE 2: formatTimeAgo
// ═══════════════════════════════════════════════════════════════════════════
console.log('\n\u2500\u2500 formatTimeAgo \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500');
test('returns "just now" for very recent date', () => {
  assert.strictEqual(formatTimeAgo(new Date().toISOString()), 'just now');
});
test('returns minutes for <1h ago', () => {
  const d = new Date(Date.now() - 30 * 60 * 1000).toISOString();
  assert.strictEqual(formatTimeAgo(d), '30m ago');
});
test('returns hours for <24h ago', () => {
  const d = new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString();
  assert.strictEqual(formatTimeAgo(d), '3h ago');
});
test('returns days for >24h ago', () => {
  const d = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString();
  assert.strictEqual(formatTimeAgo(d), '2d ago');
});
test('returns empty string for invalid date', () => {
  assert.strictEqual(formatTimeAgo('not-a-date'), '');
});

// ═══════════════════════════════════════════════════════════════════════════
// SUITE 3: renderMarkdown
// ═══════════════════════════════════════════════════════════════════════════
console.log('\n\u2500\u2500 renderMarkdown \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500');
test('renders a paragraph', () => {
  const html = renderMarkdown('Hello world');
  assert.ok(html.includes('<p'), 'should wrap in <p>');
  assert.ok(html.includes('Hello world'));
});
test('renders bold **text**', () => {
  assert.ok(renderMarkdown('**bold**').includes('<strong>bold</strong>'));
});
test('renders italic *text*', () => {
  assert.ok(renderMarkdown('*italic*').includes('<em>italic</em>'));
});
test('renders heading # H1', () => {
  const html = renderMarkdown('# Title');
  assert.ok(html.includes('<h1'));
  assert.ok(html.includes('Title'));
});
test('renders unordered list', () => {
  const html = renderMarkdown('- item1\n- item2');
  assert.ok(html.includes('<ul'));
  assert.ok(html.includes('<li>'));
  assert.ok(html.includes('item1'));
});
test('renders ordered list', () => {
  const html = renderMarkdown('1. first\n2. second');
  assert.ok(html.includes('<ol'));
  assert.ok(html.includes('first'));
});
test('renders fenced code block', () => {
  const html = renderMarkdown('```js\nconsole.log(1)\n```');
  assert.ok(html.includes('<pre'), 'should have <pre>');
  assert.ok(html.includes('console.log(1)'));
});
test('escapes HTML inside code block (XSS safe)', () => {
  const html = renderMarkdown('```\n<script>alert(1)</script>\n```');
  assert.ok(!html.includes('<script>'), 'script tag must be escaped in code block');
  assert.ok(html.includes('&lt;script&gt;'));
});
test('nested blockquote depth guard does not throw', () => {
  const html = renderMarkdown('> > > > > > deep text');
  assert.ok(typeof html === 'string');
});
test('handles unclosed code fence gracefully', () => {
  const html = renderMarkdown('```js\nconst x = 1;');
  assert.ok(html.includes('const x = 1'));
});
test('handles empty string', () => {
  assert.strictEqual(renderMarkdown(''), '');
});
test('depth > 4 returns escaped raw string (no infinite recursion)', () => {
  const result = renderMarkdown('<b>deep</b>', 5);
  assert.ok(result.includes('&lt;b&gt;'), 'should escape at depth > 4');
  assert.ok(!result.includes('<b>'));
});

// ═══════════════════════════════════════════════════════════════════════════
// SUITE 4: buildHistoryTurnEl
// ═══════════════════════════════════════════════════════════════════════════
console.log('\n\u2500\u2500 buildHistoryTurnEl \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500');
const NOW = new Date().toISOString();
function makeRecord(overrides = {}) {
  return {
    prompt_id:    'translate-vn',
    engine:       'claude',
    response:     'This is the AI response.',
    final_prompt: 'Translate: hello',
    session_id:   'sess-1',
    turn_index:   0,
    created_at:   NOW,
    ...overrides,
  };
}

test('returns element with className history-item', () => {
  const el = buildHistoryTurnEl(makeRecord(), null);
  assert.strictEqual(el.className, 'history-item');
});
test('has 3 children: header, preview, detail', () => {
  const el = buildHistoryTurnEl(makeRecord(), null);
  assert.strictEqual(el.children.length, 3);
  assert.strictEqual(el.children[0].className, 'history-item-header');
  assert.strictEqual(el.children[1].className, 'history-preview');
  assert.strictEqual(el.children[2].className, 'history-detail');
});
test('preview text is truncated to 100 chars', () => {
  const el = buildHistoryTurnEl(makeRecord({ response: 'a'.repeat(200) }), null);
  assert.ok(el.children[1].textContent.length <= 100);
});
test('preview shows "(no response)" when response is empty', () => {
  const el = buildHistoryTurnEl(makeRecord({ response: '' }), null);
  assert.strictEqual(el.children[1].textContent, '(no response)');
});
test('XSS in prompt_id is escaped', () => {
  const el = buildHistoryTurnEl(makeRecord({ prompt_id: '<img src=x onerror=alert(1)>' }), null);
  const h = el.children[0].innerHTML;
  assert.ok(!h.includes('<img'), 'raw <img> must not appear');
  assert.ok(h.includes('&lt;img'));
});
test('XSS in engine is escaped', () => {
  const el = buildHistoryTurnEl(makeRecord({ engine: '<script>evil()</script>' }), null);
  const h = el.children[0].innerHTML;
  assert.ok(!h.includes('<script>'));
  assert.ok(h.includes('&lt;script&gt;'));
});
test('label badge appears when label is provided', () => {
  const el = buildHistoryTurnEl(makeRecord(), 'Initial');
  assert.ok(el.children[0].innerHTML.includes('history-turn-badge'));
  assert.ok(el.children[0].innerHTML.includes('Initial'));
});
test('no label badge when label is null', () => {
  const el = buildHistoryTurnEl(makeRecord(), null);
  assert.ok(!el.children[0].innerHTML.includes('history-turn-badge'));
});
test('copy button exists when response is non-empty', () => {
  const el = buildHistoryTurnEl(makeRecord({ response: 'Some response' }), null);
  const detail = el.children[2];
  const respHeader = detail.children.find(c => c.className === 'history-response-header');
  const copyBtnEl  = respHeader && respHeader.children.find(c => c.className === 'history-copy-btn');
  assert.ok(copyBtnEl, 'copy button should exist when response is non-empty');
});
test('copy button does NOT exist when response is empty', () => {
  const el = buildHistoryTurnEl(makeRecord({ response: '' }), null);
  const detail = el.children[2];
  const respHeader = detail.children.find(c => c.className === 'history-response-header');
  const copyBtnEl  = respHeader && respHeader.children.find(c => c.className === 'history-copy-btn');
  assert.strictEqual(copyBtnEl, undefined, 'copy button must NOT exist when response is empty');
});
test('clicking element toggles expanded class', () => {
  const el = buildHistoryTurnEl(makeRecord(), null);
  assert.ok(!el.classList.contains('expanded'), 'should start collapsed');
  el._click(el.children[0]); // click header
  assert.ok(el.classList.contains('expanded'), 'should expand on first click');
  el._click(el.children[0]); // click header again
  assert.ok(!el.classList.contains('expanded'), 'should collapse on second click');
});
test('clicking inside detail when expanded does NOT collapse', () => {
  const el = buildHistoryTurnEl(makeRecord(), null);
  el._click(el.children[0]); // open via header click
  assert.ok(el.classList.contains('expanded'), 'should be expanded after header click');
  const detailEl = el.children[2];
  const childOfDetail = detailEl.children[0];
  childOfDetail.closest = (sel) => sel === '.history-detail' ? detailEl : null;
  el._click(childOfDetail); // click inside detail element
  assert.ok(el.classList.contains('expanded'), 'should stay expanded when clicking inside detail');
});
test('clicking header when expanded collapses item', () => {
  const el = buildHistoryTurnEl(makeRecord(), null);
  el._click(el.children[0]); // open
  assert.ok(el.classList.contains('expanded'));
  el._click(el.children[0]); // click header to close
  assert.ok(!el.classList.contains('expanded'), 'should collapse when clicking header');
});
test('finalPrompt section present when non-empty', () => {
  const el = buildHistoryTurnEl(makeRecord({ final_prompt: 'Do a thing' }), null);
  const detail = el.children[2];
  const inputLabel = detail.children.find(c => c.className === 'history-input-label');
  assert.ok(inputLabel, 'history-input-label should exist');
  assert.strictEqual(inputLabel.textContent, 'Prompt sent');
  const inputText = detail.children.find(c => c.className === 'history-input-text');
  assert.strictEqual(inputText.textContent, 'Do a thing');
});
test('XSS in finalPrompt stored safely via textContent', () => {
  const el = buildHistoryTurnEl(makeRecord({ final_prompt: '<script>evil()</script>' }), null);
  const detail = el.children[2];
  const inputText = detail.children.find(c => c.className === 'history-input-text');
  // textContent stores raw string — not parsed as HTML in real browser
  assert.strictEqual(inputText.textContent, '<script>evil()</script>');
});
test('finalPrompt section absent when empty', () => {
  const el = buildHistoryTurnEl(makeRecord({ final_prompt: '' }), null);
  const detail = el.children[2];
  const inputLabel = detail.children.find(c => c.className === 'history-input-label');
  assert.strictEqual(inputLabel, undefined);
});
test('response text rendered as markdown', () => {
  const el = buildHistoryTurnEl(makeRecord({ response: '**bold**' }), null);
  const detail = el.children[2];
  const respTextEl = detail.children.find(c => c.className === 'history-response-text');
  assert.ok(respTextEl.innerHTML.includes('<strong>bold</strong>'));
});
test('copy button text changes to Copied on click', async () => {
  const el = buildHistoryTurnEl(makeRecord({ response: 'hello' }), null);
  const detail = el.children[2];
  const respHeader = detail.children.find(c => c.className === 'history-response-header');
  const copyBtnEl  = respHeader.children.find(c => c.className === 'history-copy-btn');
  assert.strictEqual(copyBtnEl.textContent, 'Copy');
  await (copyBtnEl._listeners['click'][0])({ stopPropagation: () => {} });
  assert.strictEqual(copyBtnEl.textContent, '\u2713 Copied');
});

// ─── Summary ─────────────────────────────────────────────────────────────────
console.log(`\n${'─'.repeat(60)}`);
console.log(`Results: ${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
