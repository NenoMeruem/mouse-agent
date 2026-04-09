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
let editingRecipe = null;   // null = create mode, recipe object = edit mode
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

function capitalize(s) { return s.charAt(0).toUpperCase() + s.slice(1); }

function slugify(s) {
  return s.toLowerCase().trim().replace(/\s+/g, '_').replace(/[^a-z0-9_]/g, '');
}

// ── View switching ────────────────────────────────────────────────────────────
function showView(name) {
  viewInput.style.display  = name === 'input'  ? 'flex' : 'none';
  viewOutput.classList.toggle('visible', name === 'output');
  viewForm.classList.toggle('visible',   name === 'form');
}

// ── Recipes ───────────────────────────────────────────────────────────────────
async function loadRecipes() {
  try {
    const json = await invoke('list_recipes');
    const recipes = JSON.parse(json);
    renderRecipeList(recipes);
    if (recipes.length > 0) selectRecipe(recipes[0]);
  } catch (err) {
    console.error('Failed to load recipes:', err);
  }
}

function renderRecipeList(recipes) {
  recipeListEl.innerHTML = '';
  for (const r of recipes) {
    const el = document.createElement('div');
    el.className = 'recipe-item';
    el.dataset.id = r.id;
    el.innerHTML = `
      <span class="recipe-icon">${sparkleIcon()}</span>
      <span class="recipe-name">${r.name}</span>
      <button class="recipe-edit-btn" title="Edit" data-id="${r.id}">${editIcon()}</button>
    `;
    el.addEventListener('click', (e) => {
      if (!e.target.closest('.recipe-edit-btn')) selectRecipe(r);
    });
    el.querySelector('.recipe-edit-btn').addEventListener('click', (e) => {
      e.stopPropagation();
      openEditForm(r);
    });
    recipeListEl.appendChild(el);
  }
}

function selectRecipe(recipe) {
  activeRecipe = recipe;
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
  outputEl.textContent += payload;
  outputEl.parentElement.scrollTop = outputEl.parentElement.scrollHeight;
});

await listen('error', ({ payload }) => {
  outputEl.textContent += `\n[error] ${payload}`;
});

await listen('done', () => {
  runBtn.disabled = false;
  if (outputEl.textContent.trim()) copyBtn.style.display = '';
});

// ── Run / Output ──────────────────────────────────────────────────────────────
let currentSelection = '';
inputArea.addEventListener('input', updateRunState);

runBtn.addEventListener('click', async () => {
  const text = inputArea.value.trim();
  if (!activeRecipe || !text) return;
  showView('output');
  outputEl.textContent = '';
  copyBtn.style.display = 'none';
  runBtn.disabled = true;
  try {
    await invoke('run_recipe', {
      recipeId:   activeRecipe.id,
      selection:  text,
      engine:     activeRecipe.engine || '',
      tone:       paramValues.tone || '',
      length:     paramValues.length || '',
      complexity: paramValues.complexity || '',
    });
  } catch (err) {
    outputEl.textContent = `Error: ${err}`;
    runBtn.disabled = false;
  }
});

backBtn.addEventListener('click', () => {
  showView('input');
  updateRunState();
});

copyBtn.addEventListener('click', async () => {
  const text = outputEl.textContent;
  try {
    await navigator.clipboard.writeText(text);
    copyBtn.textContent = '✓ Copied';
    setTimeout(() => { copyBtn.textContent = 'Copy'; }, 1500);
  } catch {
    const ta = document.createElement('textarea');
    ta.value = text;
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
  }
});

// ── Recipe Form (Create + Edit) ───────────────────────────────────────────────
function openCreateForm() {
  editingRecipe = null;
  formTitle.textContent = 'New Recipe';
  document.getElementById('f-name').value = '';
  document.getElementById('f-desc').value = '';
  document.getElementById('f-template').value = '';
  document.getElementById('f-engine').value = 'gemini';
  document.getElementById('f-icon').value = '';
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
  document.getElementById('f-engine').value = recipe.engine || 'gemini';
  document.getElementById('f-icon').value = recipe.icon || '';
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
  const icon          = document.getElementById('f-icon').value.trim();
  const checkedParams = ['tone', 'length', 'complexity']
    .filter(k => document.getElementById(`fc-${k}`).checked)
    .join(',');

  try {
    if (editingRecipe) {
      await invoke('update_recipe', {
        id: editingRecipe.id, name, description, template, engine,
        params: checkedParams, icon,
      });
    } else {
      const id = slugify(name) || `recipe_${Date.now()}`;
      await invoke('save_recipe', { id, name, description, template, engine, params: checkedParams, icon });
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

// ── Drag to move window ───────────────────────────────────────────────────────
const dragTargets = [
  document.getElementById('header'),
  document.getElementById('sidebar'),
];

for (const el of dragTargets) {
  el.addEventListener('mousedown', (e) => {
    // Only left-click, and not on interactive elements
    if (e.button !== 0) return;
    if (e.target.closest('button, input, select, textarea, a')) return;
    _tauri?.window?.getCurrentWindow()?.startDragging?.();
  });
}

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

// ── Init ──────────────────────────────────────────────────────────────────────
showView('input');
loadRecipes();
