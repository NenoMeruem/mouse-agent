import { invoke } from '@tauri-apps/api/core';
import { listen } from '@tauri-apps/api/event';

// State
let activeRecipe = null;
let currentSelection = '';
const paramValues = { tone: 'casual', length: 'short', complexity: 'normal' };

const outputEl = document.getElementById('output');
const runBtn = document.getElementById('run-btn');
const copyBtn = document.getElementById('copy-btn');
const closeBtn = document.getElementById('close-btn');
const recipeListEl = document.getElementById('recipe-list');

// ── Recipes ──────────────────────────────────────────────────────────────────

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
    el.innerHTML = `<span class="recipe-icon">${r.icon || '\u{1F4C4}'}</span><span class="recipe-name">${r.name}</span>`;
    el.addEventListener('click', () => selectRecipe(r));
    recipeListEl.appendChild(el);
  }
}

function selectRecipe(recipe) {
  activeRecipe = recipe;
  // Update active class
  document.querySelectorAll('.recipe-item').forEach(el => {
    el.classList.toggle('active', el.dataset.id === recipe.id);
  });
  // Show relevant param groups
  const params = recipe.params || [];
  ['tone', 'length', 'complexity'].forEach(p => {
    const el = document.getElementById(`param-${p}`);
    if (el) el.style.display = params.includes(p) ? '' : 'none';
  });
  // Enable run if we have selection
  runBtn.disabled = !currentSelection;
}

// ── Params ───────────────────────────────────────────────────────────────────

document.querySelectorAll('.pill').forEach(pill => {
  pill.addEventListener('click', () => {
    const param = pill.dataset.param;
    const value = pill.dataset.value;
    // Deactivate siblings
    document.querySelectorAll(`.pill[data-param="${param}"]`).forEach(p => p.classList.remove('active'));
    pill.classList.add('active');
    paramValues[param] = value;
  });
});

// ── Events from Tauri ─────────────────────────────────────────────────────────

await listen('selection', ({ payload }) => {
  currentSelection = payload || '';
  if (currentSelection) {
    runBtn.disabled = activeRecipe === null;
  }
});

await listen('chunk', ({ payload }) => {
  if (outputEl.classList.contains('placeholder')) {
    outputEl.classList.remove('placeholder');
    outputEl.textContent = '';
  }
  outputEl.textContent += payload;
  // Auto-scroll
  outputEl.parentElement.scrollTop = outputEl.parentElement.scrollHeight;
});

await listen('error', ({ payload }) => {
  outputEl.textContent += `\n[error] ${payload}`;
});

await listen('done', () => {
  runBtn.disabled = false;
  if (outputEl.textContent.trim()) {
    copyBtn.style.display = '';
  }
});

// ── Actions ───────────────────────────────────────────────────────────────────

runBtn.addEventListener('click', async () => {
  if (!activeRecipe || !currentSelection) return;
  outputEl.classList.remove('placeholder');
  outputEl.textContent = '';
  copyBtn.style.display = 'none';
  runBtn.disabled = true;

  try {
    await invoke('run_recipe', {
      recipeId: activeRecipe.id,
      selection: currentSelection,
      engine: activeRecipe.engine || '',
      tone: paramValues.tone || '',
      length: paramValues.length || '',
      complexity: paramValues.complexity || '',
    });
  } catch (err) {
    outputEl.textContent = `Error: ${err}`;
    runBtn.disabled = false;
  }
});

copyBtn.addEventListener('click', async () => {
  const text = outputEl.textContent;
  try {
    await navigator.clipboard.writeText(text);
    copyBtn.textContent = '\u2713 Copied';
    setTimeout(() => { copyBtn.textContent = '\u2398 Copy'; }, 1500);
  } catch {
    // Fallback
    const ta = document.createElement('textarea');
    ta.value = text; document.body.appendChild(ta);
    ta.select(); document.execCommand('copy');
    document.body.removeChild(ta);
  }
});

closeBtn.addEventListener('click', () => invoke('hide_window'));

// ── Keyboard shortcuts ────────────────────────────────────────────────────────

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') invoke('hide_window');
  if (e.key === 'Enter' && !e.shiftKey && !runBtn.disabled) {
    e.preventDefault();
    runBtn.click();
  }
});

// ── Init ──────────────────────────────────────────────────────────────────────

loadRecipes();
