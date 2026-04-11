// Tauri API — use window.__TAURI__ (Tauri v2 plain HTML, no bundler)
const _tauri = window.__TAURI__;
const invoke = _tauri
  ? _tauri.core.invoke
  : async (cmd, args) => { console.warn('Tauri not available:', cmd, args); return '[]'; };
const listen = _tauri
  ? _tauri.event.listen
  : async () => {};

// ── State ─────────────────────────────────────────────────────────────────────
let activeRecipe  = null;
let editingRecipe = null;
let pendingDeleteId     = null;  // recipe
let pendingDeleteEngine = null;  // engine
let engineConfigs = {};
let allRecipes    = [];   // full list cache for pin re-render
let pinnedIds     = JSON.parse(localStorage.getItem('pa_pinned') || '[]'); // ordered array
const paramValues = { tone: 'casual', length: 'short', complexity: 'normal' };

const PARAM_CONFIG = {
  tone:       { label: 'Style',      emoji: '🔮', options: ['professional', 'casual', 'concise'] },
  length:     { label: 'Length',     emoji: '📏', options: ['short', 'medium', 'long'] },
  complexity: { label: 'Complexity', emoji: '🧩', options: ['simple', 'normal', 'technical'] },
};

// ── DOM refs ──────────────────────────────────────────────────────────────────
const outputEl        = document.getElementById('output');
const runBtn          = document.getElementById('run-btn');
const copyBtn         = document.getElementById('copy-btn');
const copyWrapEl      = document.getElementById('copy-wrap');
const copyFmtToggle   = document.getElementById('copy-fmt-toggle');
const copyFmtMenu     = document.getElementById('copy-fmt-menu');
const closeBtn        = document.getElementById('close-btn');
const backBtn         = document.getElementById('back-btn');
const addRecipeBtn    = document.getElementById('add-recipe-btn');
const formCancelBtn   = document.getElementById('form-cancel-btn');
const formSaveBtn     = document.getElementById('form-save-btn');
const formTitle       = document.getElementById('form-title');
const formError       = document.getElementById('form-error');
const recipeListEl    = document.getElementById('recipe-list');
const inputArea       = document.getElementById('input-area');
const instructionText = document.getElementById('instruction-text');
const paramsSection   = document.getElementById('params-section');
const viewInput       = document.getElementById('view-input');
const viewOutput      = document.getElementById('view-output');
const viewForm        = document.getElementById('view-form');
const viewSettings    = document.getElementById('view-settings');
const settingsBtn     = document.querySelector('.hbtn[title="Settings"]');
const settingsCloseBtn = document.getElementById('settings-close-btn');
const settingsMsg     = document.getElementById('settings-msg');

// ── Markdown renderer ─────────────────────────────────────────────────────────
function escapeHtml(s) {
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
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

function renderMarkdown(raw) {
  // Extract fenced code blocks to protect them from inline processing
  const fences = [];
  let md = raw.replace(/^```([\w.-]*)\r?\n([\s\S]*?)^```/gm, (_, lang, code) => {
    fences.push({ lang, code: code.replace(/\n$/, '') });
    return `\x01F${fences.length - 1}\x01`;
  });
  // Handle unclosed fence (mid-stream): render as open code block
  md = md.replace(/^```([\w.-]*)\r?\n([\s\S]*)$/m, (_, lang, code) => {
    fences.push({ lang, code });
    return `\x01F${fences.length - 1}\x01`;
  });

  const lines  = md.split('\n');
  const out    = [];
  let i        = 0;

  const flushFence = (token) => {
    const idx = parseInt(token.match(/\d+/)[0]);
    const { lang, code } = fences[idx];
    const label = lang ? `<span class="md-code-lang">${escapeHtml(lang)}</span>` : '';
    return `<pre class="md-pre">${label}<code class="md-code-block">${escapeHtml(code)}</code></pre>`;
  };

  while (i < lines.length) {
    const line = lines[i];

    // Code fence placeholder
    if (/^\x01F\d+\x01$/.test(line.trim())) {
      out.push(flushFence(line.trim()));
      i++; continue;
    }
    // Heading
    const hm = line.match(/^(#{1,6}) (.+)/);
    if (hm) {
      const lvl = Math.min(hm[1].length, 6);
      out.push(`<h${lvl} class="md-h${lvl}">${inlineMd(hm[2].trim())}</h${lvl}>`);
      i++; continue;
    }
    // Horizontal rule
    if (/^[-*_]{3,}$/.test(line.trim())) {
      out.push('<hr class="md-hr">');
      i++; continue;
    }
    // Blockquote
    if (line.startsWith('> ')) {
      const qlines = [];
      while (i < lines.length && lines[i].startsWith('> ')) { qlines.push(lines[i].slice(2)); i++; }
      out.push(`<blockquote class="md-blockquote">${renderMarkdown(qlines.join('\n'))}</blockquote>`);
      continue;
    }
    // Unordered list
    if (/^[-*+] /.test(line)) {
      const items = [];
      while (i < lines.length && /^[-*+] /.test(lines[i])) {
        items.push(`<li>${inlineMd(lines[i].replace(/^[-*+] /, ''))}</li>`);
        i++;
      }
      out.push(`<ul class="md-ul">${items.join('')}</ul>`);
      continue;
    }
    // Ordered list
    if (/^\d+[.)]\s/.test(line)) {
      const items = [];
      while (i < lines.length && /^\d+[.)]\s/.test(lines[i])) {
        items.push(`<li>${inlineMd(lines[i].replace(/^\d+[.)]\s/, ''))}</li>`);
        i++;
      }
      out.push(`<ol class="md-ol">${items.join('')}</ol>`);
      continue;
    }
    // Blank line
    if (line.trim() === '') { i++; continue; }

    // Paragraph — gather lines until block boundary
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

// ── Helpers ───────────────────────────────────────────────────────────────────
function sparkleIcon() {
  return `<svg viewBox="0 0 20 20" fill="currentColor">
    <path d="M10 1.5 L11.4 8.6 L18.5 10 L11.4 11.4 L10 18.5 L8.6 11.4 L1.5 10 L8.6 8.6 Z"/>
    <path d="M4.5 4.5 L5.1 6.4 L7 5 L5.1 5.6 Z" opacity="0.45"/>
    <path d="M15.5 4.5 L15.4 6.4 L13.8 5 L15.4 5.6 Z" opacity="0.45"/>
    <path d="M4.5 15.5 L5.1 13.6 L7 15 L5.1 14.4 Z" opacity="0.45"/>
    <path d="M15.5 15.5 L15.4 13.6 L13.8 15 L15.4 14.4 Z" opacity="0.45"/>
  </svg>`;
}

function editIcon() {
  return `<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
    <path d="M14.5 2.5a2.121 2.121 0 013 3L6 17l-4 1 1-4 11.5-11.5z"/>
  </svg>`;
}

function trashIcon() {
  return `<svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
    <polyline points="3 5 5 5 17 5"/>
    <path d="M8 5V3h4v2M5 5l1 11h8l1-11"/>
  </svg>`;
}

function pinIcon(active) {
  return `<svg viewBox="0 0 20 20" fill="${active ? 'currentColor' : 'none'}" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
    <circle cx="10" cy="7" r="3"/>
    <path d="M10 10v7M7 17h6"/>
    <line x1="7" y1="4.5" x2="13" y2="4.5"/>
  </svg>`;
}

function capitalize(s) { return s.charAt(0).toUpperCase() + s.slice(1); }

function slugify(s) {
  return s.toLowerCase().trim().replace(/\s+/g, '_').replace(/[^a-z0-9_]/g, '');
}

// ── View switching ────────────────────────────────────────────────────────────
const viewHistory  = document.getElementById('view-history');

function showView(name) {
  viewInput.style.display  = name === 'input'    ? 'flex' : 'none';
  viewOutput.classList.toggle('visible',   name === 'output');
  viewForm.classList.toggle('visible',     name === 'form');
  viewSettings.classList.toggle('visible', name === 'settings');
  viewHistory.classList.toggle('visible',  name === 'history');
}

// ── Recipes ───────────────────────────────────────────────────────────────────
async function loadRecipes() {
  try {
    const json = await invoke('list_recipes');
    const recipes = JSON.parse(json);
    allRecipes = recipes;
    renderRecipeList(recipes);
    // Restore last-used recipe, fall back to first
    const lastId = localStorage.getItem('pa_last');
    const toSelect = recipes.find(r => r.id === lastId) || recipes[0];
    if (toSelect) selectRecipe(toSelect);
  } catch (err) {
    console.error('Failed to load recipes:', err);
  }
}

function sortedByPin(recipes) {
  const pinned = pinnedIds.map(id => recipes.find(r => r.id === id)).filter(Boolean);
  const rest   = recipes.filter(r => !pinnedIds.includes(r.id));
  return [...pinned, ...rest];
}

function renderRecipeList(recipes) {
  recipeListEl.innerHTML = '';
  for (const r of sortedByPin(recipes)) {
    const isPinned = pinnedIds.includes(r.id);
    const el = document.createElement('div');
    el.className = 'recipe-item' + (isPinned ? ' pinned' : '');
    el.dataset.id = r.id;
    el.innerHTML = `
      <span class="recipe-icon">${sparkleIcon()}</span>
      <span class="recipe-name">${r.name}</span>
      <div class="recipe-actions">
        <button class="recipe-pin-btn ${isPinned ? 'active' : ''}" title="${isPinned ? 'Unpin' : 'Pin'}" data-id="${r.id}">${pinIcon(isPinned)}</button>
        <button class="recipe-edit-btn" title="Edit" data-id="${r.id}">${editIcon()}</button>
        <button class="recipe-del-btn" title="Delete" data-id="${r.id}">${trashIcon()}</button>
      </div>
    `;
    el.addEventListener('click', (e) => {
      if (!e.target.closest('.recipe-actions')) selectRecipe(r);
    });
    el.querySelector('.recipe-pin-btn').addEventListener('click', (e) => {
      e.stopPropagation();
      togglePin(r.id);
    });
    el.querySelector('.recipe-edit-btn').addEventListener('click', (e) => {
      e.stopPropagation();
      openEditForm(r);
    });
    el.querySelector('.recipe-del-btn').addEventListener('click', (e) => {
      e.stopPropagation();
      showDeleteConfirm(r);
    });
    recipeListEl.appendChild(el);
  }
}

function togglePin(id) {
  const idx = pinnedIds.indexOf(id);
  if (idx === -1) pinnedIds.push(id);
  else pinnedIds.splice(idx, 1);
  localStorage.setItem('pa_pinned', JSON.stringify(pinnedIds));
  renderRecipeList(allRecipes);
  // Restore active highlight after re-render
  if (activeRecipe) {
    document.querySelectorAll('.recipe-item').forEach(el => {
      el.classList.toggle('active', el.dataset.id === activeRecipe.id);
    });
  }
}

function showDeleteConfirm(recipe) {
  pendingDeleteId = recipe.id;
  const modal = document.getElementById('delete-modal');
  document.getElementById('delete-modal-name').textContent = recipe.name || recipe.id;
  modal.classList.add('visible');
}

function hideDeleteConfirm() {
  pendingDeleteId     = null;
  pendingDeleteEngine = null;
  document.getElementById('delete-modal').classList.remove('visible');
}

async function deleteRecipe(id) {
  console.log('[delete] deleting recipe:', id);
  try {
    const result = await invoke('delete_recipe', { id });
    console.log('[delete] result:', result);
    if (activeRecipe && activeRecipe.id === id) {
      activeRecipe = null;
      instructionText.textContent = 'Select a recipe to see instructions…';
      instructionText.classList.add('ph');
      paramsSection.innerHTML = '';
      updateRunState();
    }
    await loadRecipes();
  } catch (err) {
    console.error('[delete] error:', err);
  }
}

document.getElementById('delete-modal-cancel').addEventListener('click', hideDeleteConfirm);
document.getElementById('delete-modal-confirm').addEventListener('click', async () => {
  if (pendingDeleteId) {
    const id = pendingDeleteId;
    hideDeleteConfirm();
    await deleteRecipe(id);
  } else if (pendingDeleteEngine) {
    const eng = pendingDeleteEngine;
    hideDeleteConfirm();
    await doDeleteEngine(eng);
  }
});
document.getElementById('delete-modal').addEventListener('click', (e) => {
  if (e.target === e.currentTarget) hideDeleteConfirm();
});

function populateEngineDropdown(selectedValue) {
  const select = document.getElementById('f-engine');
  select.innerHTML = '';
  const engines = Object.keys(engineConfigs);
  if (engines.length === 0) {
    const opt = document.createElement('option');
    opt.value = '';
    opt.textContent = 'No engines configured';
    select.appendChild(opt);
  } else {
    for (const eng of engines) {
      const opt = document.createElement('option');
      opt.value = eng;
      opt.textContent = capitalize(eng);
      select.appendChild(opt);
    }
    select.value = (selectedValue && engineConfigs[selectedValue]) ? selectedValue : engines[0];
  }
}

function selectRecipe(recipe) {
  activeRecipe = recipe;
  localStorage.setItem('pa_last', recipe.id);
  document.querySelectorAll('.recipe-item').forEach(el => {
    el.classList.toggle('active', el.dataset.id === recipe.id);
  });
  // Show template content in instructions
  const tpl = recipe.template || recipe.description || '';
  instructionText.textContent = tpl ? `"${tpl}"` : 'No instructions';
  instructionText.classList.remove('ph');
  renderParams(recipe.params || []);
  updateRunState();
}

// ── Params ────────────────────────────────────────────────────────────────────
function renderParams(params) {
  paramsSection.innerHTML = '';
  for (const pKey of params) {
    const cfg = PARAM_CONFIG[pKey];
    if (!cfg) continue;
    const row = document.createElement('div');
    row.className = 'param-row';
    row.dataset.param = pKey;
    row.innerHTML = `
      <div class="param-row-header">
        <span class="param-row-label">${cfg.label}</span>
        <div class="param-row-right">
          <span class="param-emoji">${cfg.emoji}</span>
          <span class="param-value">${capitalize(paramValues[pKey] || 'Select')}</span>
          <span class="param-caret">⌃</span>
        </div>
      </div>
      <div class="param-options">
        ${cfg.options.map(o =>
          `<button class="param-opt${paramValues[pKey] === o ? ' active' : ''}" data-param="${pKey}" data-value="${o}">${capitalize(o)}</button>`
        ).join('')}
      </div>
    `;
    row.querySelector('.param-row-header').addEventListener('click', () => row.classList.toggle('open'));
    row.querySelectorAll('.param-opt').forEach(opt => {
      opt.addEventListener('click', () => {
        const p = opt.dataset.param;
        const v = opt.dataset.value;
        paramValues[p] = v;
        row.querySelectorAll('.param-opt').forEach(o => o.classList.toggle('active', o.dataset.value === v));
        row.querySelector('.param-value').textContent = capitalize(v);
      });
    });
    paramsSection.appendChild(row);
  }
}

function updateRunState() {
  runBtn.disabled = !activeRecipe || !inputArea.value.trim();
}

// ── Tauri events ──────────────────────────────────────────────────────────────
await listen('selection', ({ payload }) => {
  currentSelection = payload || '';
  if (currentSelection) {
    inputArea.value = currentSelection;
    updateRunState();
  }
});

await listen('chunk', ({ payload }) => {
  const bodyEl = document.getElementById('output-body');
  if (!bodyEl) return;
  // Remove "thinking" indicator on first real chunk
  const thinking = outputEl.querySelector('.output-thinking');
  if (thinking) thinking.remove();
  aiResponseText += payload;
  bodyEl.innerHTML = renderMarkdown(aiResponseText);
  outputEl.scrollTop = outputEl.scrollHeight;
});

await listen('error', ({ payload }) => {
  const bodyEl = document.getElementById('output-body');
  if (bodyEl) {
    const errEl = document.createElement('div');
    errEl.className = 'output-error';
    errEl.textContent = payload;
    bodyEl.appendChild(errEl);
  }
});

await listen('done', () => {
  setRunLoading(false);
  const thinking = outputEl.querySelector('.output-thinking');
  if (thinking) thinking.remove();
  const trimmed = aiResponseText.trim();
  if (trimmed) {
    copyWrapEl.style.display = '';
    // Word count meta
    const wordCount = trimmed.split(/\s+/).filter(Boolean).length;
    const metaEl = document.getElementById('output-meta');
    if (metaEl) metaEl.textContent = `${wordCount} words`;
    // Done badge
    const bodyEl = document.getElementById('output-body');
    if (bodyEl && !bodyEl.querySelector('.output-done-badge')) {
      const badge = document.createElement('div');
      badge.className = 'output-done-badge';
      badge.textContent = 'Done';
      bodyEl.appendChild(badge);
    }
  }
});

// ── Run / Output ──────────────────────────────────────────────────────────────
let currentSelection = '';
let aiResponseText = '';   // pure AI text — used for copy

inputArea.addEventListener('input', updateRunState);

function setRunLoading(loading) {
  const sendIcon = runBtn.querySelector('.run-icon-send');
  const loadIcon = runBtn.querySelector('.run-icon-loading');
  runBtn.disabled = loading;
  runBtn.classList.toggle('loading', loading);
  sendIcon.style.display = loading ? 'none' : '';
  loadIcon.style.display = loading ? 'flex' : 'none';
}

runBtn.addEventListener('click', async () => {
  const text = inputArea.value.trim();
  if (!activeRecipe || !text) return;

  // Guard: engine must have an API key configured
  const engine = activeRecipe.engine || '';
  const engineCfg = engineConfigs[engine] || {};
  if (!engineCfg.api_key) {
    showView('output');
    outputEl.innerHTML = '';
    copyWrapEl.style.display = 'none';
    outputEl.innerHTML = `<div class="output-error no-key-error">
      No API key for <strong>${engine || 'this engine'}</strong>.
      Go to <strong>Settings ⚙</strong> and add the key first.
    </div>`;
    return;
  }

  aiResponseText = '';
  showView('output');
  outputEl.innerHTML = '';
  copyWrapEl.style.display = 'none';
  copyFmtMenu.classList.add('hidden');
  setRunLoading(true);
  outputEl.innerHTML = `<div class="output-thinking"><span class="output-engine-dot"></span>Thinking with <strong>${engine}</strong>…</div><div class="output-body" id="output-body"></div>`;
  const bodyEl = document.getElementById('output-body');

  try {
    const recipeParams = activeRecipe.params || [];
    await invoke('run_recipe', {
      recipeId:   activeRecipe.id,
      selection:  text,
      engine:     engine,
      tone:       recipeParams.includes('tone')       ? paramValues.tone       : '',
      length:     recipeParams.includes('length')     ? paramValues.length     : '',
      complexity: recipeParams.includes('complexity') ? paramValues.complexity : '',
    });
  } catch (err) {
    const errEl = document.createElement('div');
    errEl.className = 'output-error';
    errEl.textContent = String(err);
    outputEl.innerHTML = '';
    outputEl.appendChild(errEl);
    setRunLoading(false);
  }
});

backBtn.addEventListener('click', () => {
  showView('input');
  updateRunState();
});

async function writeToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    const ta = document.createElement('textarea');
    ta.value = text;
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
  }
}

async function copyAs(fmt) {
  const text = aiResponseText.trim();
  if (!text) return;
  let content = text;
  if (fmt === 'html') content = renderMarkdown(text);
  if (fmt === 'with-prompt') content = `Input:\n${inputArea.value.trim()}\n\n---\n\n${text}`;
  await writeToClipboard(content);
  copyBtn.innerHTML = '✓ Copied';
  setTimeout(() => { copyBtn.innerHTML = '<svg width="12" height="12" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="8" y="8" width="10" height="10" rx="2"/><path d="M4 12H3a2 2 0 01-2-2V3a2 2 0 012-2h7a2 2 0 012 2v1"/></svg> Copy'; }, 1800);
  copyFmtMenu.classList.add('hidden');
}

copyBtn.addEventListener('click', () => copyAs('text'));

copyFmtToggle.addEventListener('click', (e) => {
  e.stopPropagation();
  copyFmtMenu.classList.toggle('hidden');
});

document.querySelectorAll('.copy-fmt-opt').forEach(opt => {
  opt.addEventListener('click', () => copyAs(opt.dataset.fmt));
});

document.addEventListener('click', () => copyFmtMenu.classList.add('hidden'));

// ── Recipe Form (Create + Edit) ───────────────────────────────────────────────
function openCreateForm() {
  editingRecipe = null;
  formTitle.textContent = 'New Recipe';
  document.getElementById('f-name').value = '';
  document.getElementById('f-desc').value = '';
  document.getElementById('f-template').value = '';
  populateEngineDropdown('');
  document.getElementById('fc-tone').checked = false;
  document.getElementById('fc-length').checked = false;
  document.getElementById('fc-complexity').checked = false;
  formError.textContent = '';
  formSaveBtn.disabled = false;
  formSaveBtn.textContent = 'Save Recipe';
  showView('form');
}

function openEditForm(recipe) {
  editingRecipe = recipe;
  formTitle.textContent = 'Edit Recipe';
  document.getElementById('f-name').value = recipe.name || '';
  document.getElementById('f-desc').value = recipe.description || '';
  document.getElementById('f-template').value = recipe.template || '';
  populateEngineDropdown(recipe.engine || '');
  const params = recipe.params || [];
  document.getElementById('fc-tone').checked = params.includes('tone');
  document.getElementById('fc-length').checked = params.includes('length');
  document.getElementById('fc-complexity').checked = params.includes('complexity');
  formError.textContent = '';
  formSaveBtn.disabled = false;
  formSaveBtn.textContent = 'Save Changes';
  showView('form');
}

addRecipeBtn.addEventListener('click', openCreateForm);
formCancelBtn.addEventListener('click', () => showView('input'));

formSaveBtn.addEventListener('click', async () => {
  const name     = document.getElementById('f-name').value.trim();
  const template = document.getElementById('f-template').value.trim();

  if (!name)     { formError.textContent = 'Name is required'; return; }
  if (!template) { formError.textContent = 'Template is required'; return; }

  formError.textContent = '';
  formSaveBtn.disabled = true;
  formSaveBtn.textContent = 'Saving…';

  const description   = document.getElementById('f-desc').value.trim();
  const engine        = document.getElementById('f-engine').value;
  const checkedParams = ['tone', 'length', 'complexity']
    .filter(k => document.getElementById(`fc-${k}`).checked)
    .join(',');

  try {
    if (editingRecipe) {
      await invoke('update_recipe', {
        id: editingRecipe.id, name, description, template, engine,
        params: checkedParams, icon: '',
      });
    } else {
      const id = slugify(name) || `recipe_${Date.now()}`;
      await invoke('save_recipe', { id, name, description, template, engine, params: checkedParams, icon: '' });
    }
    await loadRecipes();
    showView('input');
  } catch (err) {
    formError.textContent = String(err);
  } finally {
    formSaveBtn.disabled = false;
    formSaveBtn.textContent = editingRecipe ? 'Save Changes' : 'Save Recipe';
  }
});

// ── Window drag ───────────────────────────────────────────────────────────────
// data-tauri-drag-region is set on #header but can be unreliable with
// transparent + decoration-less windows on macOS. Manually call startDragging().
function makeDraggable(el) {
  el.addEventListener('mousedown', (e) => {
    if (e.target.closest('button, input, textarea, select, a')) return;
    if (e.button !== 0) return; // left click only
    try {
      _tauri?.window?.getCurrentWindow?.()?.startDragging?.();
    } catch (_) {}
  });
}
makeDraggable(document.getElementById('header'));
makeDraggable(document.querySelector('.sidebar-header'));

// ── Close / Keyboard ──────────────────────────────────────────────────────────
closeBtn.addEventListener('click', () => {
  invoke('hide_window').catch(() => window.close());
});

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') invoke('hide_window').catch(() => window.close());
  if (e.key === 'Enter' && !e.shiftKey && !runBtn.disabled && viewInput.style.display !== 'none') {
    e.preventDefault();
    runBtn.click();
  }
});

// ── Settings ──────────────────────────────────────────────────────────────────
const engineListEl    = document.getElementById('engine-list');
const addEngineBtn    = document.getElementById('add-engine-btn');
const engineForm      = document.getElementById('engine-form');
const engineFormTitle = document.getElementById('engine-form-title');
const engineFormCancel = document.getElementById('engine-form-cancel');
const engineFormSave  = document.getElementById('engine-form-save');
const engineFormErr   = document.getElementById('engine-form-err');
const efId    = document.getElementById('ef-id');
const efKey   = document.getElementById('ef-key');
const efModel = document.getElementById('ef-model');

let editingEngine = null; // null = new, string = engine id being edited

async function loadEngineConfigs() {
  try {
    const json = await invoke('get_engine_configs');
    engineConfigs = JSON.parse(json) || {};
  } catch {
    engineConfigs = {};
  }
  renderEngineList();
}

function renderEngineList() {
  engineListEl.innerHTML = '';
  const engines = Object.keys(engineConfigs);
  if (engines.length === 0) {
    engineListEl.innerHTML = '<p class="settings-empty">No engines configured yet.</p>';
    return;
  }
  for (const eng of engines) {
    const cfg = engineConfigs[eng] || {};
    const key = cfg.api_key || '';
    const model = cfg.model || '';
    const hasKey = key !== '';
    const card = document.createElement('div');
    card.className = 'engine-card';
    card.innerHTML = `
      <div class="engine-header">
        <span class="engine-dot"></span>
        <span class="engine-name">${eng}</span>
        <span class="engine-status ${hasKey ? 'configured' : 'empty'}">${hasKey ? 'Configured' : 'Not set'}</span>
        <div class="engine-card-actions">
          <button class="engine-edit-btn" data-engine="${eng}" title="Edit">Edit</button>
          <button class="engine-del-btn" data-engine="${eng}" title="Delete">✕</button>
        </div>
      </div>
      <div class="engine-info">
        <span class="engine-info-row"><span class="engine-info-label">Key:</span> ${key.startsWith('env:') ? `<code>${key}</code>` : (hasKey ? '••••••••' : '—')}</span>
        <span class="engine-info-row"><span class="engine-info-label">Model:</span> ${model || '—'}</span>
      </div>
    `;
    card.querySelector('.engine-edit-btn').addEventListener('click', () => openEngineForm(eng));
    card.querySelector('.engine-del-btn').addEventListener('click', () => deleteEngine(eng));
    engineListEl.appendChild(card);
  }
}

function openEngineForm(engId = null) {
  editingEngine = engId;
  engineFormTitle.textContent = engId ? `Edit — ${engId}` : 'New Engine';
  efId.value    = engId || '';
  efId.disabled = !!engId; // can't rename existing engine
  const cfg     = engId ? (engineConfigs[engId] || {}) : {};
  const key     = cfg.api_key || '';
  efKey.value   = key.startsWith('env:') ? '' : key;
  efKey.placeholder = key.startsWith('env:') ? key : (engId === 'gemini' ? 'AIza...' : engId === 'openai' ? 'sk-...' : engId === 'claude' ? 'sk-ant-...' : 'api-key');
  efModel.value = cfg.model || '';
  engineFormErr.textContent = '';
  engineForm.classList.remove('hidden');
  efId.focus();
}

function closeEngineForm() {
  engineForm.classList.add('hidden');
  editingEngine = null;
}

function deleteEngine(eng) {
  pendingDeleteEngine = eng;
  document.getElementById('delete-modal-name').textContent = `engine "${eng}"`;
  document.getElementById('delete-modal').classList.add('visible');
}

async function doDeleteEngine(eng) {
  settingsMsg.textContent = '';
  try {
    await invoke('delete_engine_config', { engine: eng });
    await loadEngineConfigs();
    settingsMsg.textContent = `✓ Engine "${eng}" deleted`;
    settingsMsg.className = 'settings-msg';
    setTimeout(() => { settingsMsg.textContent = ''; }, 2500);
  } catch (err) {
    settingsMsg.textContent = String(err);
    settingsMsg.className = 'settings-msg error';
  }
}

addEngineBtn.addEventListener('click', () => openEngineForm(null));
engineFormCancel.addEventListener('click', closeEngineForm);

engineFormSave.addEventListener('click', async () => {
  const id    = efId.value.trim();
  const key   = efKey.value.trim();
  const model = efModel.value.trim();

  if (!id) { engineFormErr.textContent = 'Engine ID is required'; return; }

  engineFormSave.disabled = true;
  engineFormSave.textContent = 'Saving…';
  engineFormErr.textContent = '';
  settingsMsg.textContent = '';

  try {
    await invoke('save_engine_config', { engine: id, apiKey: key, model });
    await loadEngineConfigs();
    closeEngineForm();
    settingsMsg.textContent = `✓ Engine "${id}" saved`;
    settingsMsg.className = 'settings-msg';
    setTimeout(() => { settingsMsg.textContent = ''; }, 2500);
  } catch (err) {
    engineFormErr.textContent = String(err);
  } finally {
    engineFormSave.disabled = false;
    engineFormSave.textContent = 'Save';
  }
});

settingsBtn.addEventListener('click', async () => {
  await loadEngineConfigs();
  closeEngineForm();
  settingsMsg.textContent = '';
  settingsMsg.className = 'settings-msg';
  showView('settings');
});

settingsCloseBtn.addEventListener('click', () => showView('input'));

// ── History ───────────────────────────────────────────────────────────────────
const historyBtn      = document.getElementById('history-btn');
const historyCloseBtn = document.getElementById('history-close-btn');
const historyClearBtn = document.getElementById('history-clear-btn');
const historyListEl   = document.getElementById('history-list');

function formatTimeAgo(dateStr) {
  const d = new Date(dateStr);
  if (isNaN(d)) return '';
  const diff = Date.now() - d.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

async function loadHistory() {
  historyListEl.innerHTML = '<p class="history-empty">Loading…</p>';
  try {
    const json = await invoke('list_history');
    const records = JSON.parse(json) || [];
    if (records.length === 0) {
      historyListEl.innerHTML = '<p class="history-empty">No history yet.</p>';
      return;
    }
    historyListEl.innerHTML = '';
    for (const r of records) {
      const item = document.createElement('div');
      item.className = 'history-item';
      const responseText  = (r.response     || '').trim();
      const finalPrompt   = (r.final_prompt || '').trim();
      const preview = responseText.slice(0, 100).replace(/\n+/g, ' ') || '(no response)';
      item.innerHTML = `
        <div class="history-item-header">
          <span class="history-recipe">${r.prompt_id || '—'}</span>
          <span class="history-engine">${r.engine || ''}</span>
          <span class="history-time">${formatTimeAgo(r.created_at)}</span>
        </div>
        <div class="history-preview">${escapeHtml(preview)}</div>
        <div class="history-detail">
          ${finalPrompt ? `<div class="history-input-label">Prompt sent</div><div class="history-input-text">${escapeHtml(finalPrompt)}</div>` : ''}
          <div class="history-response-header">
            <span class="history-response-label">Response</span>
            ${responseText ? `<button class="history-copy-btn" title="Copy response">Copy</button>` : ''}
          </div>
          <div class="history-response-text">${responseText ? renderMarkdown(responseText) : '<em>(empty)</em>'}</div>
        </div>
      `;
      item.addEventListener('click', (e) => {
        if (!e.target.closest('.history-copy-btn')) item.classList.toggle('expanded');
      });
      if (responseText) {
        item.querySelector('.history-copy-btn').addEventListener('click', async (e) => {
          e.stopPropagation();
          await writeToClipboard(responseText);
          const btn = e.currentTarget;
          btn.textContent = '✓ Copied';
          setTimeout(() => { btn.textContent = 'Copy'; }, 1800);
        });
      }
      historyListEl.appendChild(item);
    }
  } catch (err) {
    historyListEl.innerHTML = `<p class="history-empty">${err}</p>`;
  }
}

historyBtn.addEventListener('click', async () => {
  await loadHistory();
  showView('history');
});

historyCloseBtn.addEventListener('click', () => showView('input'));

let clearConfirmPending = false;
let clearConfirmTimer = null;

historyClearBtn.addEventListener('click', async () => {
  if (!clearConfirmPending) {
    // First click: ask for confirmation inline
    clearConfirmPending = true;
    historyClearBtn.textContent = 'Sure?';
    clearConfirmTimer = setTimeout(() => {
      clearConfirmPending = false;
      historyClearBtn.textContent = 'Clear All';
    }, 3000);
    return;
  }
  // Second click within 3s: execute
  clearTimeout(clearConfirmTimer);
  clearConfirmPending = false;
  historyClearBtn.textContent = 'Clear All';
  try {
    await invoke('clear_history');
    await loadHistory();
  } catch (err) {
    historyListEl.innerHTML = `<p class="history-empty">Error: ${err}</p>`;
  }
});

// ── Init ──────────────────────────────────────────────────────────────────────
showView('input');
loadRecipes();
loadEngineConfigs();
