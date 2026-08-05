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
let allRecipes    = [];   // full list cache for re-render
const paramValues = { tone: 'casual', length: 'short', complexity: 'normal' };

const PARAM_CONFIG = {
  tone:       { label: 'Style',      emoji: '', options: ['professional', 'casual', 'concise'] },
  length:     { label: 'Length',     emoji: '', options: ['short', 'medium', 'long'] },
  complexity: { label: 'Complexity', emoji: '', options: ['simple', 'normal', 'technical'] },
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
const breadcrumbPage  = document.getElementById('breadcrumb-page');
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

function renderMarkdown(raw, _depth = 0) {
  // Guard against infinite recursion (e.g. deeply nested blockquotes)
  if (_depth > 4) return escapeHtml(raw);

  // Extract fenced code blocks to protect them from inline processing
  // Uses a non-backtracking approach: split on ``` lines manually
  const fences = [];
  const fenceLines = raw.split('\n');
  const processedLines = [];
  let inFence = false;
  let fenceLang = '';
  let fenceContent = [];
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
  // Handle unclosed fence at end of stream
  if (inFence) {
    fences.push({ lang: fenceLang, code: fenceContent.join('\n') });
    processedLines.push(`\x01F${fences.length - 1}\x01`);
  }
  let md = processedLines.join('\n');

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
    // Blockquote — use depth+1 to prevent infinite recursion
    if (line.startsWith('> ')) {
      const qlines = [];
      while (i < lines.length && lines[i].startsWith('> ')) { qlines.push(lines[i].slice(2)); i++; }
      out.push(`<blockquote class="md-blockquote">${renderMarkdown(qlines.join('\n'), _depth + 1)}</blockquote>`);
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


// ── Recipe drag-and-drop (mouse-event based — more reliable in Tauri WebView) ──
let dragState = null; // { srcId, srcEl }

function onDragMove(e) {
  if (!dragState) return;
  recipeListEl.querySelectorAll('.recipe-item').forEach(i => i.classList.remove('drag-over'));
  const els = document.elementsFromPoint(e.clientX, e.clientY);
  const target = els.find(el => el.classList.contains('recipe-item') && el.dataset.id && el.dataset.id !== dragState.srcId);
  if (target) {
    target.classList.add('drag-over');
    dragState.currentTargetId = target.dataset.id;
  } else {
    dragState.currentTargetId = null;
  }
}

async function onDragEnd() {
  document.removeEventListener('mousemove', onDragMove);
  document.removeEventListener('mouseup', onDragEnd);
  if (!dragState) return;

  dragState.srcEl.classList.remove('dragging');
  recipeListEl.querySelectorAll('.recipe-item').forEach(i => i.classList.remove('drag-over'));

  const { srcId, currentTargetId } = dragState;
  dragState = null;

  if (!currentTargetId || currentTargetId === srcId) return;

  const items = [...recipeListEl.querySelectorAll('.recipe-item')];
  const ids = items.map(i => i.dataset.id);
  const srcIdx = ids.indexOf(srcId);
  const dstIdx = ids.indexOf(currentTargetId);
  if (srcIdx === -1 || dstIdx === -1) return;
  ids.splice(srcIdx, 1);
  ids.splice(dstIdx, 0, srcId);

  const idToRecipe = Object.fromEntries(allRecipes.map(x => [x.id, x]));
  allRecipes = ids.map(id => idToRecipe[id]).filter(Boolean);
  renderRecipeList(allRecipes);
  if (activeRecipe) {
    document.querySelectorAll('.recipe-item').forEach(el => {
      el.classList.toggle('active', el.dataset.id === activeRecipe.id);
    });
  }

  try {
    await invoke('reorder_recipes', { ids });
  } catch (err) {
    console.error('Failed to persist recipe order:', err);
  }
}

function renderRecipeList(recipes) {
  recipeListEl.innerHTML = '';
  for (const r of recipes) {
    const el = document.createElement('div');
    el.className = 'recipe-item';
    el.dataset.id = r.id;
    el.innerHTML = `
      <span class="recipe-drag-handle" title="Drag to reorder">⠿</span>
      <span class="recipe-icon">${sparkleIcon()}</span>
      <span class="recipe-name">${r.name}</span>
      <div class="recipe-actions">
        <button class="recipe-edit-btn" title="Edit" data-id="${r.id}">${editIcon()}</button>
        <button class="recipe-del-btn" title="Delete" data-id="${r.id}">${trashIcon()}</button>
      </div>
    `;
    el.addEventListener('click', (e) => {
      if (!e.target.closest('.recipe-actions, .recipe-drag-handle')) selectRecipe(r);
    });
    el.querySelector('.recipe-edit-btn').addEventListener('click', (e) => {
      e.stopPropagation();
      openEditForm(r);
    });
    el.querySelector('.recipe-del-btn').addEventListener('click', (e) => {
      e.stopPropagation();
      showDeleteConfirm(r);
    });

    // Drag handle — mousedown starts the drag
    el.querySelector('.recipe-drag-handle').addEventListener('mousedown', (e) => {
      if (e.button !== 0) return;
      e.preventDefault();
      e.stopPropagation();
      dragState = { srcId: r.id, srcEl: el, currentTargetId: null };
      el.classList.add('dragging');
      document.addEventListener('mousemove', onDragMove);
      document.addEventListener('mouseup', onDragEnd);
    });

    recipeListEl.appendChild(el);
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
      const cfg = engineConfigs[eng] || {};
      opt.textContent = cfg.name || capitalize(eng);
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
  // Update breadcrumb
  if (breadcrumbPage) breadcrumbPage.textContent = recipe.name || '';
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

// ── Conversation state (declared early — used in Tauri event listeners below) ──
let conversationHistory = [];
let conversationEngine  = '';
let conversationSessionId = '';
let conversationTurnIndex = 0;   // 0 = initial run, 1,2,3... = follow-ups
let conversationPromptId  = '';

const followupBar   = document.getElementById('followup-bar');
const followupInput = document.getElementById('followup-input');
const followupBtn   = document.getElementById('followup-btn');

function setFollowupLoading(loading) {
  if (followupInput) followupInput.disabled = loading;
  if (followupBtn)   followupBtn.disabled   = loading;
}

// ── Tauri events ──────────────────────────────────────────────────────────────
await listen('selection', ({ payload }) => {
  currentSelection = payload || '';
  // Always reset to input view so clipboard context is visible and ready
  showView('input');
  if (currentSelection) {
    inputArea.value = currentSelection;
  }
  updateRunState();
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
  setFollowupLoading(false);
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
    // Save AI response to conversation history and show follow-up input
    if (conversationEngine) {
      // For turn 0 (initial recipe), seed history with the user's input text
      if (conversationHistory.length === 0) {
        conversationHistory.push({ role: 'user', content: currentSelection || inputArea.value.trim() });
      }
      conversationHistory.push({ role: 'assistant', content: trimmed });
      followupBar.classList.remove('hidden');
      followupInput.focus();
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

// Append a streaming turn into the output element.
// Returns a reference to the new body div for chunk appending.
function appendTurnStart(engine) {
  const bodyEl = document.getElementById('output-body');
  if (bodyEl) {
    const hr = document.createElement('hr');
    hr.className = 'chat-divider';
    bodyEl.appendChild(hr);
  }
  const thinking = document.createElement('div');
  thinking.className = 'output-thinking';
  thinking.innerHTML = `<span class="output-engine-dot"></span>Thinking with <strong>${engine}</strong>…`;
  outputEl.appendChild(thinking);
  const newBody = document.createElement('div');
  newBody.className = 'output-body';
  newBody.id = 'output-body';
  outputEl.appendChild(newBody);
  return { thinking, newBody };
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

  // Reset conversation for a new recipe run
  conversationHistory   = [];
  conversationEngine    = engine;
  conversationSessionId = crypto.randomUUID();
  conversationTurnIndex = 0;
  conversationPromptId  = activeRecipe.id;
  aiResponseText = '';
  showView('output');
  outputEl.innerHTML = '';
  copyWrapEl.style.display = 'none';
  copyFmtMenu.classList.add('hidden');
  followupBar.classList.add('hidden');
  setRunLoading(true);
  outputEl.innerHTML = `<div class="output-thinking"><span class="output-engine-dot"></span>Thinking with <strong>${engine}</strong>…</div><div class="output-body" id="output-body"></div>`;

  try {
    const recipeParams = activeRecipe.params || [];
    await invoke('run_recipe', {
      recipeId:   activeRecipe.id,
      selection:  text,
      engine:     engine,
      tone:       recipeParams.includes('tone')       ? paramValues.tone       : '',
      length:     recipeParams.includes('length')     ? paramValues.length     : '',
      complexity: recipeParams.includes('complexity') ? paramValues.complexity : '',
      sessionId:  conversationSessionId,
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

// Send follow-up message
async function sendFollowup() {
  const text = followupInput.value.trim();
  if (!text || conversationEngine === '') return;

  // Append user turn to history and display it
  conversationHistory.push({ role: 'user', content: text });
  followupInput.value = '';

  const bodyEl = document.getElementById('output-body');
  if (bodyEl) {
    const hr = document.createElement('hr');
    hr.className = 'chat-divider';
    bodyEl.appendChild(hr);
    const bubble = document.createElement('div');
    bubble.className = 'chat-user-bubble';
    bubble.textContent = text;
    bodyEl.appendChild(bubble);
  }

  followupBar.classList.add('hidden');
  setFollowupLoading(true);
  aiResponseText = '';

  // Append thinking indicator + new body
  const thinking = document.createElement('div');
  thinking.className = 'output-thinking';
  thinking.innerHTML = `<span class="output-engine-dot"></span>Thinking with <strong>${conversationEngine}</strong>…`;
  outputEl.appendChild(thinking);
  const newBody = document.createElement('div');
  newBody.className = 'output-body';
  newBody.id = 'output-body';
  outputEl.appendChild(newBody);
  outputEl.parentElement.scrollTop = outputEl.parentElement.scrollHeight;

  conversationTurnIndex += 1;

  try {
    await invoke('send_chat', {
      messages:   JSON.stringify(conversationHistory),
      engine:     conversationEngine,
      sessionId:  conversationSessionId,
      turnIndex:  conversationTurnIndex,
      promptId:   conversationPromptId,
    });
  } catch (err) {
    thinking.remove();
    const errEl = document.createElement('div');
    errEl.className = 'output-error';
    errEl.textContent = String(err);
    newBody.appendChild(errEl);
    setFollowupLoading(false);
    followupBar.classList.remove('hidden');
  }
}

followupBtn.addEventListener('click', sendFollowup);
followupInput.addEventListener('keydown', e => {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    sendFollowup();
  }
});
// Auto-resize textarea
followupInput.addEventListener('input', () => {
  followupInput.style.height = 'auto';
  followupInput.style.height = Math.min(followupInput.scrollHeight, 80) + 'px';
});

backBtn.addEventListener('click', () => {
  conversationHistory   = [];
  conversationEngine    = '';
  conversationSessionId = '';
  conversationTurnIndex = 0;
  conversationPromptId  = '';
  followupBar.classList.add('hidden');
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
const efId       = document.getElementById('ef-id');       // text input: unique slug
const efProvider = document.getElementById('ef-provider'); // select: gemini/openai/claude
const efName  = document.getElementById('ef-name');
const efKey   = document.getElementById('ef-key');
const efModel = document.getElementById('ef-model');

let editingEngine = null; // null = new, string = engine id being edited

const ENGINE_MODELS = {
  gemini: [
    'gemini-2.5-pro',
    'gemini-2.5-flash',
    'gemini-2.5-flash-lite',
    'gemini-2.0-flash',
    'gemini-1.5-flash',
    'gemini-1.5-pro',
  ],
  claude: [
    'claude-opus-4-5',
    'claude-sonnet-4-5',
    'claude-haiku-4-5',
  ],
  openai: [
    'gpt-4o',
    'gpt-4o-mini',
    'gpt-4-turbo',
    'gpt-3.5-turbo',
  ],
};

function updateModelDatalist(engineId) {
  const dl = document.getElementById('engine-model-list');
  dl.innerHTML = '';
  const models = ENGINE_MODELS[engineId] || [];
  for (const m of models) {
    const opt = document.createElement('option');
    opt.value = m;
    dl.appendChild(opt);
  }
}

async function loadEngineConfigs() {
  try {
    const raw = await invoke('get_engine_configs');
    // Backend returns Array [{id, provider, name, api_key, model}, ...]
    const arr = JSON.parse(raw);
    engineConfigs = {};
    if (Array.isArray(arr)) {
      for (const item of arr) {
        engineConfigs[item.id] = {
          provider: item.provider,
          name: item.name,
          api_key: item.api_key,
          model: item.model,
        };
      }
    }
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
    const displayName = cfg.name || eng;
    const hasKey = key !== '';
    const card = document.createElement('div');
    card.className = 'engine-card';
    card.innerHTML = `
      <div class="engine-header">
        <span class="engine-dot"></span>
        <span class="engine-name">${displayName}</span>
        <span class="engine-id-badge">${eng}</span>
        ${cfg.provider ? `<span class="engine-provider-badge">${cfg.provider}</span>` : ''}
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

const ENGINE_KEY_PLACEHOLDERS = { gemini: 'AIza...', claude: 'sk-ant-...', openai: 'sk-...' };

function applyEngineFormDefaults(provider) {
  updateModelDatalist(provider);
  efKey.placeholder = ENGINE_KEY_PLACEHOLDERS[provider] || 'api-key';
}

function openEngineForm(engId = null) {
  editingEngine = engId;
  engineFormTitle.textContent = engId ? `Edit — ${engId}` : 'New Engine';

  // ID field: editable only when creating new
  efId.value    = engId || '';
  efId.disabled = !!engId;

  // Load existing config if editing
  const cfg = engId ? (engineConfigs[engId] || {}) : {};
  const key = cfg.api_key || '';
  const provider = cfg.provider || 'gemini';

  efProvider.value = provider;
  efProvider.disabled = !!engId; // lock provider when editing (changing would break the entry)
  efName.value  = cfg.name  || '';
  efKey.value   = key.startsWith('env:') ? '' : key;
  efModel.value = cfg.model || '';

  // Update model datalist + key placeholder for selected provider
  applyEngineFormDefaults(provider);
  if (key.startsWith('env:')) efKey.placeholder = key;

  engineFormErr.textContent = '';
  engineForm.classList.remove('hidden');
  efId.disabled ? efKey.focus() : efId.focus();
}

function closeEngineForm() {
  engineForm.classList.add('hidden');
  editingEngine = null;
  efId.value = '';
  efProvider.value = 'gemini';
  efName.value = '';
  efKey.value = '';
  efModel.value = '';
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

// Update model suggestions and key placeholder when provider changes
efProvider.addEventListener('change', () => applyEngineFormDefaults(efProvider.value));

engineFormSave.addEventListener('click', async () => {
  const id       = editingEngine || efId.value.trim().toLowerCase().replace(/\s+/g, '-');
  const provider = editingEngine
    ? (engineConfigs[editingEngine]?.provider || efProvider.value)
    : efProvider.value;
  const name  = efName.value.trim();
  const key   = efKey.value.trim();
  const model = efModel.value.trim();

  if (!id) {
    engineFormErr.textContent = 'ID is required (e.g. gemini-flash).';
    return;
  }

  // Check duplicate ID when creating new
  if (!editingEngine && engineConfigs[id]) {
    engineFormErr.textContent = `ID "${id}" already exists. Choose a different ID.`;
    return;
  }

  engineFormSave.disabled = true;
  engineFormSave.textContent = 'Saving…';
  engineFormErr.textContent = '';
  settingsMsg.textContent = '';

  try {
    // Always pass all fields — backend handles empty strings as "clear field"
    await invoke('save_engine_config', { engine: id, provider, apiKey: key, model, name });
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

// ── Hotkey recorder ───────────────────────────────────────────────────────────
const hotkeyDisplayText  = document.getElementById('hotkey-display-text');
const hotkeyRecordBtn    = document.getElementById('hotkey-record-btn');
const hotkeyRecordingHint = document.getElementById('hotkey-recording-hint');
const hotkeyMsg          = document.getElementById('hotkey-msg');

let isRecording = false;

async function loadHotkey() {
  try {
    const hk = await invoke('get_hotkey_cmd');
    hotkeyDisplayText.textContent = hk || 'Alt+Space';
  } catch {
    hotkeyDisplayText.textContent = 'Alt+Space';
  }
}

function startRecording() {
  isRecording = true;
  hotkeyRecordBtn.textContent = 'Cancel';
  hotkeyRecordBtn.classList.add('recording');
  hotkeyRecordingHint.classList.remove('hidden');
  hotkeyDisplayText.textContent = '…';
}

function stopRecording() {
  isRecording = false;
  hotkeyRecordBtn.textContent = 'Change';
  hotkeyRecordBtn.classList.remove('recording');
  hotkeyRecordingHint.classList.add('hidden');
}

hotkeyRecordBtn.addEventListener('click', () => {
  if (isRecording) {
    stopRecording();
    loadHotkey(); // restore previous
  } else {
    startRecording();
  }
});

document.addEventListener('keydown', async (e) => {
  if (!isRecording) return;
  e.preventDefault();

  if (e.key === 'Escape') {
    stopRecording();
    loadHotkey();
    return;
  }

  // Need at least one modifier
  const mods = [];
  if (e.ctrlKey)  mods.push('Ctrl');
  if (e.altKey)   mods.push('Alt');
  if (e.shiftKey) mods.push('Shift');
  if (e.metaKey)  mods.push('Super');

  const modKeys = new Set(['Control','Alt','Shift','Meta']);
  if (modKeys.has(e.key)) return; // still waiting for the main key

  if (mods.length === 0) return; // require at least one modifier

  const key = e.code.startsWith('Key')   ? e.code.slice(3)      // KeyA → A
            : e.code.startsWith('Digit') ? e.code.slice(5)      // Digit1 → 1
            : e.code === 'Space'         ? 'Space'
            : e.key;

  const hotkey = [...mods, key].join('+');
  hotkeyDisplayText.textContent = hotkey;
  stopRecording();

  hotkeyMsg.textContent = '';
  try {
    await invoke('set_hotkey', { hotkey });
    hotkeyMsg.textContent = `✓ Hotkey saved: ${hotkey}`;
    hotkeyMsg.className = 'settings-msg';
    setTimeout(() => { hotkeyMsg.textContent = ''; }, 2500);
  } catch (err) {
    hotkeyMsg.textContent = String(err);
    hotkeyMsg.className = 'settings-msg error';
    loadHotkey();
  }
});

// ── Appearance / Theme ────────────────────────────────────────────────────────
function applyTheme(name) {
  document.documentElement.dataset.theme = name;
  document.querySelectorAll('.theme-swatch').forEach(s => {
    s.classList.toggle('active', s.dataset.theme === name);
  });
  localStorage.setItem('theme', name);
}

function applyDark(dark) {
  document.documentElement.dataset.dark = dark ? 'true' : 'false';
  const btn = document.getElementById('dark-toggle');
  if (btn) btn.setAttribute('aria-pressed', dark ? 'true' : 'false');
  localStorage.setItem('dark', dark ? 'true' : 'false');
}

document.querySelectorAll('.theme-swatch').forEach(btn => {
  btn.addEventListener('click', () => applyTheme(btn.dataset.theme));
});

document.getElementById('dark-toggle').addEventListener('click', () => {
  applyDark(document.documentElement.dataset.dark !== 'true');
});

// Sync toggle state with what the inline script already applied at load time
applyDark(localStorage.getItem('dark') === 'true');
applyTheme(localStorage.getItem('theme') || 'claude');

// ── Settings tabs ─────────────────────────────────────────────────────────────
function switchSettingsTab(tabName) {
  document.querySelectorAll('.settings-tab').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.tab === tabName);
  });
  document.querySelectorAll('.settings-tab-panel').forEach(panel => {
    panel.classList.toggle('active', panel.id === `settings-tab-${tabName}`);
  });
  if (tabName === 'hotkeys') loadHotkey();
}

document.querySelectorAll('.settings-tab').forEach(btn => {
  btn.addEventListener('click', () => switchSettingsTab(btn.dataset.tab));
});

settingsBtn.addEventListener('click', async () => {
  await loadEngineConfigs();
  closeEngineForm();
  settingsMsg.textContent = '';
  settingsMsg.className = 'settings-msg';
  switchSettingsTab('engines');
  showView('settings');
});

settingsCloseBtn.addEventListener('click', () => showView('input'));

// ── Backup/Restore ────────────────────────────────────────────────────────────
let selectedImportFile = null;

async function exportBackup() {
  const exportBtn = document.getElementById('backup-export-btn');
  const msgEl = document.getElementById('backup-msg');
  msgEl.textContent = '';
  msgEl.className = 'settings-msg';

  try {
    exportBtn.disabled = true;
    exportBtn.textContent = 'Exporting...';
    
    const csvContent = await invoke('export_data');
    
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.setAttribute("href", url);
    
    const dateStr = new Date().toISOString().slice(0, 10);
    link.setAttribute("download", `promptly_backup_${dateStr}.csv`);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    msgEl.textContent = '✓ Export successful!';
    msgEl.className = 'settings-msg';
  } catch (err) {
    console.error(err);
    msgEl.textContent = `❌ Export failed: ${err}`;
    msgEl.className = 'settings-msg error';
  } finally {
    exportBtn.disabled = false;
    exportBtn.textContent = 'Export to CSV';
  }
}

function setupBackupTab() {
  const importFileEl = document.getElementById('backup-import-file');
  const importBtn = document.getElementById('backup-import-btn');
  const submitBtn = document.getElementById('backup-import-submit-btn');
  const fileNameSpan = document.getElementById('import-file-name');
  const exportBtn = document.getElementById('backup-export-btn');
  const msgEl = document.getElementById('backup-msg');

  exportBtn.addEventListener('click', exportBackup);

  importBtn.addEventListener('click', () => {
    importFileEl.click();
  });

  importFileEl.addEventListener('change', (e) => {
    const file = e.target.files[0];
    if (file) {
      selectedImportFile = file;
      fileNameSpan.textContent = file.name;
      submitBtn.disabled = false;
      msgEl.textContent = '';
    } else {
      selectedImportFile = null;
      fileNameSpan.textContent = 'No file selected';
      submitBtn.disabled = true;
    }
  });

  submitBtn.addEventListener('click', async () => {
    if (!selectedImportFile) return;

    msgEl.textContent = '';
    msgEl.className = 'settings-msg';
    submitBtn.disabled = true;
    submitBtn.textContent = 'Importing...';

    const reader = new FileReader();
    reader.onload = async (e) => {
      const csvContent = e.target.result;
      try {
        const response = await invoke('import_data', { jsonContent: csvContent });
        msgEl.textContent = response || '✓ Import successful!';
        msgEl.className = 'settings-msg';
        
        await loadRecipes();
        await loadEngineConfigs();
        
        importFileEl.value = '';
        selectedImportFile = null;
        fileNameSpan.textContent = 'No file selected';
      } catch (err) {
        console.error(err);
        msgEl.textContent = `❌ Import failed: ${err}`;
        msgEl.className = 'settings-msg error';
        submitBtn.disabled = false;
      } finally {
        submitBtn.textContent = 'Import Data';
      }
    };
    reader.onerror = () => {
      msgEl.textContent = '❌ Failed to read selected file';
      msgEl.className = 'settings-msg error';
      submitBtn.disabled = false;
      submitBtn.textContent = 'Import Data';
    };
    reader.readAsText(selectedImportFile);
  });
}


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

function buildHistoryTurnEl(r, label) {
  const responseText = (r.response     || '').trim();
  const finalPrompt  = (r.final_prompt || '').trim();
  const preview      = responseText.slice(0, 100).replace(/\n+/g, ' ') || '(no response)';

  // Always escape dynamic values inserted via innerHTML
  const safeRecipeId = escapeHtml(r.prompt_id || '—');
  const safeEngine   = escapeHtml(r.engine    || '');
  const safeTime     = escapeHtml(formatTimeAgo(r.created_at));
  const safeLabel    = label ? escapeHtml(label) : '';

  const el = document.createElement('div');
  el.className = 'history-item';

  // Build header
  const header = document.createElement('div');
  header.className = 'history-item-header';
  header.innerHTML = `
    <span class="history-recipe">${safeRecipeId}</span>
    ${safeLabel ? `<span class="history-turn-badge">${safeLabel}</span>` : ''}
    <span class="history-engine">${safeEngine}</span>
    <span class="history-time">${safeTime}</span>
  `;

  // Build preview
  const previewEl = document.createElement('div');
  previewEl.className = 'history-preview';
  previewEl.textContent = preview;

  // Build detail (hidden until expanded)
  const detail = document.createElement('div');
  detail.className = 'history-detail';

  if (finalPrompt) {
    const inputLabel = document.createElement('div');
    inputLabel.className = 'history-input-label';
    inputLabel.textContent = 'Prompt sent';

    const inputText = document.createElement('div');
    inputText.className = 'history-input-text';
    inputText.textContent = finalPrompt;   // textContent = safe, no XSS

    detail.appendChild(inputLabel);
    detail.appendChild(inputText);
  }

  const responseHeader = document.createElement('div');
  responseHeader.className = 'history-response-header';

  const responseLabel = document.createElement('span');
  responseLabel.className = 'history-response-label';
  responseLabel.textContent = 'Response';
  responseHeader.appendChild(responseLabel);

  // Copy button — only created when there IS a response
  let copyBtn = null;
  if (responseText) {
    copyBtn = document.createElement('button');
    copyBtn.className = 'history-copy-btn';
    copyBtn.title = 'Copy response';
    copyBtn.textContent = 'Copy';
    copyBtn.addEventListener('click', async (e) => {
      e.stopPropagation();
      await writeToClipboard(responseText);
      copyBtn.textContent = '✓ Copied';
      setTimeout(() => { copyBtn.textContent = 'Copy'; }, 1800);
    });
    responseHeader.appendChild(copyBtn);
  }

  const responseText_el = document.createElement('div');
  responseText_el.className = 'history-response-text';
  let hasRenderedMarkdown = false;

  if (!responseText) {
    responseText_el.innerHTML = '<em>(empty)</em>';
    hasRenderedMarkdown = true;
  }

  detail.appendChild(responseHeader);
  detail.appendChild(responseText_el);

  el.appendChild(header);
  el.appendChild(previewEl);
  el.appendChild(detail);

  // Toggle expanded/collapsed.
  // - Lazy render Markdown only when first expanding an item
  // - Clicking header or preview always toggles.
  // - Clicking INSIDE the detail area (text, links) when already expanded does nothing
  // - Copy button always stops propagation itself, so it never reaches here.
  el.addEventListener('click', (e) => {
    const targetNode = e.target;
    if (targetNode && typeof targetNode.closest === 'function' && targetNode.closest('.history-detail')) return;

    const willExpand = !el.classList.contains('expanded');
    if (willExpand && !hasRenderedMarkdown && responseText) {
      console.log(`[History Log] Lazy rendering Markdown for record ${r.id || 'unknown'}`);
      responseText_el.innerHTML = renderMarkdown(responseText);
      hasRenderedMarkdown = true;
    }

    el.classList.toggle('expanded');
  });

  return el;
}


// Helper: invoke with a timeout so a hung backend doesn't freeze the UI
function invokeWithTimeout(cmd, args, ms = 8000) {
  return Promise.race([
    invoke(cmd, args),
    new Promise((_, reject) =>
      setTimeout(() => reject(new Error(`Command '${cmd}' timed out after ${ms}ms`)), ms)
    ),
  ]);
}

async function loadHistory() {
  console.log('[History Log] Loading history requested...');
  historyListEl.innerHTML = '<p class="history-empty">Loading…</p>';
  try {
    const json = await invokeWithTimeout('list_history', {}, 8000);
    const records = JSON.parse(json) || [];
    console.log(`[History Log] Received ${records.length} history records from backend`);

    if (records.length === 0) {
      historyListEl.innerHTML = '<p class="history-empty">No history yet.</p>';
      return;
    }
    historyListEl.innerHTML = '';

    // Group records by session_id; records without a session go solo
    const sessions = new Map();
    for (const r of records) {
      const sid = r.session_id || '';
      if (!sid) continue;
      if (!sessions.has(sid)) sessions.set(sid, []);
      sessions.get(sid).push(r);
    }
    for (const turns of sessions.values()) {
      turns.sort((a, b) => (a.turn_index || 0) - (b.turn_index || 0));
    }

    // Build the ordered list of items to render
    const renderQueue = [];
    const seenSessions = new Set();
    for (const r of records) {
      const sid = r.session_id || '';
      if (!sid) {
        renderQueue.push({ type: 'solo', record: r });
        continue;
      }
      if (seenSessions.has(sid)) continue;
      seenSessions.add(sid);
      const turns = sessions.get(sid);
      if (turns.length === 1) {
        renderQueue.push({ type: 'solo', record: turns[0] });
      } else {
        renderQueue.push({ type: 'thread', turns });
      }
    }

    // Render in small batches to avoid blocking the main thread
    const BATCH = 5;
    let idx = 0;
    function renderBatch() {
      const end = Math.min(idx + BATCH, renderQueue.length);
      for (; idx < end; idx++) {
        const item = renderQueue[idx];
        if (item.type === 'solo') {
          historyListEl.appendChild(buildHistoryTurnEl(item.record, null));
        } else {
          const { turns } = item;
          const thread = document.createElement('div');
          thread.className = 'history-thread';
          const threadHeader = document.createElement('div');
          threadHeader.className = 'history-thread-header';
          threadHeader.innerHTML = `
            <span class="history-thread-icon">💬</span>
            <span class="history-thread-label">${escapeHtml(turns[0].prompt_id || 'Conversation')}</span>
            <span class="history-thread-count">${turns.length} turns</span>
            <span class="history-thread-time">${escapeHtml(formatTimeAgo(turns[0].created_at))}</span>
            <span class="history-thread-caret">›</span>
          `;
          const threadBody = document.createElement('div');
          threadBody.className = 'history-thread-body';
          for (const turn of turns) {
            const label = turn.turn_index === 0 ? 'Initial' : `Follow-up ${turn.turn_index}`;
            const turnEl = buildHistoryTurnEl(turn, label);
            turnEl.classList.add('history-thread-turn');
            threadBody.appendChild(turnEl);
          }
          threadHeader.addEventListener('click', () => thread.classList.toggle('expanded'));
          thread.appendChild(threadHeader);
          thread.appendChild(threadBody);
          historyListEl.appendChild(thread);
        }
      }
      if (idx < renderQueue.length) {
        setTimeout(renderBatch, 0);  // yield to browser between batches
      } else {
        console.log('[History Log] Batch rendering completed successfully');
      }
    }
    renderBatch();

  } catch (err) {
    console.error('[History Log ERROR] Failed to load history:', err);
    historyListEl.innerHTML = `<p class="history-empty">Error: ${err.message || err}</p>`;
  }
}

historyBtn.addEventListener('click', async () => {
  showView('history');   // show immediately — don't await loadHistory first
  loadHistory();
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

// Listen for tray menu event to open history
listen('open-history', () => {
  showView('history');   // show immediately
  loadHistory();
});

// ── Init ──────────────────────────────────────────────────────────────────────
showView('input');
loadRecipes();
loadEngineConfigs();
setupBackupTab();

// Load clipboard on startup — runs after all variables are declared
invoke('get_clipboard').then(clip => {
  if (clip) {
    currentSelection = clip;
    inputArea.value = clip;
    updateRunState();
  }
}).catch(() => {});
