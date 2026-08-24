// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]
#![allow(dead_code)]

mod config;
mod llm;
mod models;
mod prompt;
mod storage;

use config::{
    get_engine_api_key, get_engine_model, get_engine_timeout_secs, get_hotkey,
    load_config, save_config, AppConfig,
};
use models::{LlmRequest, Message, RunRecord};
use storage::{
    history_append, history_clear_all, history_list, open_db, prompt_create,
    prompt_delete, prompt_get, prompt_list, prompt_next_sort_order, prompt_reorder,
    prompt_update, seed_default_recipes_if_empty,
};

use chrono::Utc;
use rusqlite::Connection;
use std::collections::HashMap;
use std::sync::Mutex;
use std::time::Duration;
use tauri::{AppHandle, Emitter, Manager, State};
use tokio::sync::mpsc;
use uuid::Uuid;

use llm::{
    sidecar::SidecarClient,
    LlmClient, Manager as LlmManager,
};

// ---------------------------------------------------------------------------
// App State — stored in Tauri's managed state
// ---------------------------------------------------------------------------

pub struct AppState {
    pub db: Mutex<Connection>,
    pub config: Mutex<AppConfig>,
}

// ---------------------------------------------------------------------------
// LLM Manager builder — routes requests through FastAPI + LangChain Sidecar
// ---------------------------------------------------------------------------

fn build_llm_manager(cfg: &AppConfig) -> LlmManager {
    let mut mgr = LlmManager::new();
    let sidecar_url = "http://127.0.0.1:8000";

    // Iterate ALL engine entries — each has a unique ID but may share a provider.
    for (id, ec) in &cfg.engines {
        let provider = config::infer_provider(id, ec);
        let api_key  = get_engine_api_key(cfg, id);
        let model    = get_engine_model(cfg, id);
        let timeout  = Duration::from_secs(get_engine_timeout_secs(cfg, id));

        // Connect via Python FastAPI LangChain Sidecar, passing api_key resolved from Rust
        let client: Box<dyn LlmClient> = Box::new(SidecarClient::new(
            sidecar_url,
            provider,
            model,
            api_key,
            timeout,
        ));

        // Register with the entry's unique ID so run_recipe can look up by id
        mgr.register_id(id.clone(), client);
    }

    mgr
}



// ---------------------------------------------------------------------------
// Tauri Commands
// ---------------------------------------------------------------------------

/// Run a recipe: build the prompt from selection + variables, call LLM,
/// stream chunks back to the WebView as `chunk` / `error` / `done` events.
#[tauri::command]
#[allow(clippy::too_many_arguments)]
async fn run_recipe(
    app: AppHandle,
    state: State<'_, AppState>,
    recipe_id: String,
    selection: String,
    engine: String,
    tone: String,
    length: String,
    complexity: String,
    session_id: String,
) -> Result<(), String> {
    // ── 1. Load prompt from DB ──────────────────────────────────────────────
    let (template, prompt_engine, prompt_id, variables) = {
        let db = state.db.lock().unwrap();
        let p = prompt_get(&db, &recipe_id).map_err(|e| e.to_string())?;
        (p.template, p.engine, p.id, p.variables)
    };

    // ── 2. Build template data map ─────────────────────────────────────────
    let mut data: HashMap<String, String> = HashMap::new();
    if !selection.is_empty() {
        data.insert("selection".into(), selection.clone());
    }
    // Fill any remaining {{variable}} from env vars
    for var in &variables {
        if !data.contains_key(var.as_str()) {
            let env_key = format!("PROMPT_{}", var.to_uppercase());
            if let Ok(val) = std::env::var(&env_key) {
                data.insert(var.clone(), val);
            }
        }
    }

    // ── 3. Build final prompt ──────────────────────────────────────────────
    let mut final_prompt = prompt::build_prompt(&template, &data);

    // ── 4. Inject params ──────────────────────────────────────────────────
    let mut param_values: HashMap<String, String> = HashMap::new();
    if !tone.is_empty() { param_values.insert("tone".into(), tone); }
    if !length.is_empty() { param_values.insert("length".into(), length); }
    if !complexity.is_empty() { param_values.insert("complexity".into(), complexity); }
    if !param_values.is_empty() {
        final_prompt = prompt::inject_params(&final_prompt, &param_values);
    }

    // ── 5. Resolve engine ─────────────────────────────────────────────────
    let resolved_engine = if engine.is_empty() { prompt_engine } else { engine };

    // ── 6. Build LLM manager and get client ───────────────────────────────
    let cfg = state.config.lock().unwrap().clone();
    let llm_mgr = build_llm_manager(&cfg);
    let model = get_engine_model(&cfg, &resolved_engine);

    let client: &dyn LlmClient = llm_mgr.get(&resolved_engine)
        .ok_or_else(|| format!("Engine '{}' not configured or API key missing", resolved_engine))?;

    // ── 7. Stream response ─────────────────────────────────────────────────
    let req = LlmRequest {
        prompt: final_prompt.clone(),
        model,
        messages: vec![],
    };

    let (tx, mut rx) = mpsc::channel(64);
    let start = std::time::Instant::now();
    let mut response_buf = String::new();

    client.stream(req, tx).await;

    let app_clone = app.clone();
    let sid = session_id.clone();
    let fp = final_prompt.clone();
    let sel = selection.clone();
    let eng = resolved_engine.clone();
    let pid = prompt_id.clone();

    // Spawn receiver task — this runs in the Tauri async runtime
    tauri::async_runtime::spawn(async move {
        while let Some(chunk) = rx.recv().await {
            if let Some(err) = &chunk.error {
                app_clone.emit("error", err.clone()).ok();
                break;
            }
            if !chunk.text.is_empty() {
                response_buf.push_str(&chunk.text);
                app_clone.emit("chunk", chunk.text.clone()).ok();
            }
            if chunk.done {
                let duration_ms = start.elapsed().as_millis() as i64;
                // Save to history (non-fatal)
                if let Ok(db_guard) = app_clone.state::<AppState>().db.lock() {
                    let record = RunRecord {
                        id: Uuid::new_v4().to_string(),
                        session_id: sid.clone(),
                        turn_index: 0,
                        prompt_id: pid.clone(),
                        engine: eng.clone(),
                        input_text: sel.clone(),
                        final_prompt: fp.clone(),
                        response: response_buf.clone(),
                        duration_ms,
                        error: String::new(),
                        created_at: Utc::now(),
                    };
                    history_append(&db_guard, &record).ok();
                }
                app_clone.emit("done", ()).ok();
                break;
            }
        }
    });

    Ok(())
}

/// Continue a multi-turn chat conversation.
#[tauri::command]
async fn send_chat(
    app: AppHandle,
    state: State<'_, AppState>,
    messages_json: String,
    engine: String,
    session_id: String,
    turn_index: u32,
    prompt_id: String,
) -> Result<(), String> {
    let messages: Vec<Message> = serde_json::from_str(&messages_json)
        .map_err(|e| format!("Invalid messages JSON: {}", e))?;

    if messages.is_empty() {
        return Err("messages array is empty".into());
    }

    // Extract last user message for history
    let user_input = messages.iter().rev()
        .find(|m| m.role == "user")
        .map(|m| m.content.clone())
        .unwrap_or_default();

    let cfg = state.config.lock().unwrap().clone();
    let llm_mgr = build_llm_manager(&cfg);
    let model = get_engine_model(&cfg, &engine);

    let client: &dyn LlmClient = llm_mgr.get(&engine)
        .ok_or_else(|| format!("Engine '{}' not configured or API key missing", engine))?;

    let req = LlmRequest { prompt: String::new(), model, messages: messages.clone() };

    let (tx, mut rx) = mpsc::channel(64);
    let start = std::time::Instant::now();
    let mut response_buf = String::new();

    client.stream(req, tx).await;

    let app_clone = app.clone();
    let eng = engine.clone();
    let sid = session_id.clone();
    let pid = prompt_id.clone();

    tauri::async_runtime::spawn(async move {
        while let Some(chunk) = rx.recv().await {
            if let Some(err) = &chunk.error {
                app_clone.emit("error", err.clone()).ok();
                break;
            }
            if !chunk.text.is_empty() {
                response_buf.push_str(&chunk.text);
                app_clone.emit("chunk", chunk.text.clone()).ok();
            }
            if chunk.done {
                let duration_ms = start.elapsed().as_millis() as i64;
                if let Ok(db_guard) = app_clone.state::<AppState>().db.lock() {
                    let record = RunRecord {
                        id: Uuid::new_v4().to_string(),
                        session_id: sid.clone(),
                        turn_index: turn_index as i64,
                        prompt_id: pid.clone(),
                        engine: eng.clone(),
                        input_text: user_input.clone(),
                        final_prompt: user_input.clone(),
                        response: response_buf.clone(),
                        duration_ms,
                        error: String::new(),
                        created_at: Utc::now(),
                    };
                    history_append(&db_guard, &record).ok();
                }
                app_clone.emit("done", ()).ok();
                break;
            }
        }
    });

    Ok(())
}

/// Returns all recipes as a JSON array string.
#[tauri::command]
fn list_recipes(state: State<'_, AppState>) -> Result<String, String> {
    let db = state.db.lock().unwrap();
    let recipes = prompt_list(&db).map_err(|e| e.to_string())?;
    serde_json::to_string(&recipes).map_err(|e| e.to_string())
}

/// Save (create) a new recipe.
#[tauri::command]
#[allow(clippy::too_many_arguments)]
fn save_recipe(
    state: State<'_, AppState>,
    id: String,
    name: String,
    description: String,
    template: String,
    engine: String,
    params: String,
    icon: String,
) -> Result<String, String> {
    let db = state.db.lock().unwrap();
    let variables = prompt::extract_variables(&template);
    let params_vec: Vec<String> = if params.is_empty() {
        vec![]
    } else {
        params.split(',').map(|s| s.trim().to_string()).collect()
    };
    let sort_order = prompt_next_sort_order(&db);
    let now = Utc::now();
    let p = models::Prompt {
        id,
        name,
        description,
        engine,
        template,
        variables,
        params: params_vec,
        icon,
        sort_order,
        created_at: now,
        updated_at: now,
    };
    prompt_create(&db, &p).map_err(|e| e.to_string())?;
    Ok("created".into())
}

/// Update an existing recipe.
#[tauri::command]
#[allow(clippy::too_many_arguments)]
fn update_recipe(
    state: State<'_, AppState>,
    id: String,
    name: String,
    description: String,
    template: String,
    engine: String,
    params: String,
    icon: String,
) -> Result<String, String> {
    let db = state.db.lock().unwrap();
    let mut existing = prompt_get(&db, &id).map_err(|e| e.to_string())?;
    if !name.is_empty() { existing.name = name; }
    if !description.is_empty() { existing.description = description; }
    if !template.is_empty() {
        existing.variables = prompt::extract_variables(&template);
        existing.template = template;
    }
    if !engine.is_empty() { existing.engine = engine; }
    existing.params = if params.is_empty() {
        vec![]
    } else {
        params.split(',').map(|s| s.trim().to_string()).collect()
    };
    if !icon.is_empty() { existing.icon = icon; }
    prompt_update(&db, &existing).map_err(|e| e.to_string())?;
    Ok("updated".into())
}

/// Delete a recipe by ID.
#[tauri::command]
fn delete_recipe(state: State<'_, AppState>, id: String) -> Result<String, String> {
    let db = state.db.lock().unwrap();
    prompt_delete(&db, &id).map_err(|e| e.to_string())?;
    Ok("deleted".into())
}

/// Reorder recipes by providing their IDs in the desired order.
#[tauri::command]
fn reorder_recipes(state: State<'_, AppState>, ids: Vec<String>) -> Result<String, String> {
    let db = state.db.lock().unwrap();
    prompt_reorder(&db, &ids).map_err(|e| e.to_string())?;
    Ok("reordered".into())
}

/// Returns all engine configs as a JSON array.
#[tauri::command]
fn get_engine_configs(state: State<'_, AppState>) -> Result<String, String> {
    let cfg = state.config.lock().unwrap();
    Ok(config::get_engine_configs_json(&cfg))
}

/// Upsert (create or update) a single engine config.
/// `engine` = unique user-defined ID (e.g. "gemini-flash", "my-gpt4")
/// `provider` = API provider type: "gemini" | "openai" | "claude"
#[tauri::command]
fn save_engine_config(
    state: State<'_, AppState>,
    engine: String,
    provider: String,
    api_key: String,
    model: String,
    name: String,
) -> Result<String, String> {
    if engine.trim().is_empty() {
        return Err("Engine ID cannot be empty".into());
    }
    let mut cfg = state.config.lock().unwrap();
    // Always overwrite all fields so the UI can intentionally clear them
    let ec = cfg.engines.entry(engine).or_default();
    if !provider.is_empty() {
        ec.provider = provider;
    }
    ec.api_key = api_key;
    ec.model   = model;
    ec.name    = name;
    save_config(&cfg)?;
    Ok("saved".into())
}

/// Delete an engine config by name.
#[tauri::command]
fn delete_engine_config(state: State<'_, AppState>, engine: String) -> Result<String, String> {
    let mut cfg = state.config.lock().unwrap();
    cfg.engines.remove(&engine);
    save_config(&cfg)?;
    Ok("deleted".into())
}

/// Quick connectivity check: send a minimal prompt to the engine and return
/// "ok" on success or an error message.
#[tauri::command]
async fn ping_engine(state: State<'_, AppState>, engine: String) -> Result<String, String> {
    let cfg = state.config.lock().unwrap().clone();
    let llm_mgr = build_llm_manager(&cfg);
    let model = get_engine_model(&cfg, &engine);

    let client = llm_mgr.get(&engine)
        .ok_or_else(|| format!("Engine '{}' not configured", engine))?;

    let (tx, mut rx) = mpsc::channel(8);
    client.stream(
        LlmRequest { prompt: "Respond with exactly: ok".into(), model, messages: vec![] },
        tx,
    ).await;

    while let Some(chunk) = rx.recv().await {
        if let Some(err) = chunk.error {
            return Err(err);
        }
        if chunk.done {
            return Ok("ok".into());
        }
    }
    Ok("ok".into())
}

/// Returns the run history as a JSON array (most recent 50 entries).
#[tauri::command]
fn list_history(state: State<'_, AppState>) -> Result<String, String> {
    let db = state.db.lock().unwrap();
    let records = history_list(&db, 50, "").map_err(|e| e.to_string())?;
    serde_json::to_string(&records).map_err(|e| e.to_string())
}

/// Deletes all run history records.
#[tauri::command]
fn clear_history(state: State<'_, AppState>) -> Result<String, String> {
    let db = state.db.lock().unwrap();
    history_clear_all(&db).map_err(|e| e.to_string())?;
    Ok("cleared".into())
}

/// Returns the configured global hotkey string.
#[tauri::command]
fn get_hotkey_cmd(state: State<'_, AppState>) -> String {
    let cfg = state.config.lock().unwrap();
    get_hotkey(&cfg)
}

/// Persists a new hotkey and re-registers the global shortcut immediately.
#[tauri::command]
fn set_hotkey(
    app: AppHandle,
    state: State<'_, AppState>,
    hotkey: String,
) -> Result<(), String> {
    use tauri_plugin_global_shortcut::GlobalShortcutExt;

    // Read old hotkey before overwriting config, so we can unregister it specifically
    let old_hotkey = {
        let cfg = state.config.lock().unwrap();
        get_hotkey(&cfg)
    };

    // Persist new hotkey to config
    {
        let mut cfg = state.config.lock().unwrap();
        cfg.hotkey = Some(hotkey.clone());
        save_config(&cfg)?;
    }

    // Unregister old hotkey specifically (avoids permission issues with unregister_all)
    if let Err(e) = app.global_shortcut().unregister(old_hotkey.as_str()) {
        eprintln!("Warning: failed to unregister old hotkey '{}': {}", old_hotkey, e);
    }

    // Register new hotkey — the global handler set up via with_handler() still applies
    app.global_shortcut().register(hotkey.as_str()).map_err(|e| {
        format!(
            "Saved to config, but failed to bind global shortcut '{}': {}",
            hotkey, e
        )
    })?;

    Ok(())
}

/// Read clipboard text — called by JS on startup to pre-populate the selection field.
#[tauri::command]
fn get_clipboard(app: AppHandle) -> String {
    use tauri_plugin_clipboard_manager::ClipboardExt;
    app.clipboard().read_text().unwrap_or_default()
}

/// Hide the overlay window.
#[tauri::command]
fn hide_window(app: AppHandle) {
    if let Some(window) = app.get_webview_window("overlay") {
        window.hide().ok();
    }
}

/// Export all prompt recipes as JSON.
#[tauri::command]
fn export_data(state: State<'_, AppState>) -> Result<String, String> {
    let db = state.db.lock().unwrap();
    let recipes = prompt_list(&db).map_err(|e| e.to_string())?;
    serde_json::to_string_pretty(&recipes).map_err(|e| e.to_string())
}

/// Import prompt recipes from a JSON string (array of Prompt objects).
#[tauri::command]
fn import_data(state: State<'_, AppState>, json_content: String) -> Result<String, String> {
    let recipes: Vec<models::Prompt> = serde_json::from_str(&json_content)
        .map_err(|e| format!("Invalid JSON: {}", e))?;

    let db = state.db.lock().unwrap();
    let mut imported = 0usize;
    for mut recipe in recipes {
        recipe.variables = prompt::extract_variables(&recipe.template);
        if recipe.sort_order == 0 {
            recipe.sort_order = prompt_next_sort_order(&db);
        }
        // Skip if already exists
        if prompt_get(&db, &recipe.id).is_err() {
            prompt_create(&db, &recipe).map_err(|e| e.to_string())?;
            imported += 1;
        }
    }
    Ok(format!("Imported {} recipe(s)", imported))
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

fn main() {
    let cfg = load_config();
    let db_path = config::db_path();
    let conn = open_db(&db_path).expect("Failed to open SQLite database");
    seed_default_recipes_if_empty(&conn).ok();

    let state = AppState {
        db: Mutex::new(conn),
        config: Mutex::new(cfg),
    };

    tauri::Builder::default()
        .manage(state)
        .plugin(tauri_plugin_clipboard_manager::init())
        .plugin(tauri_plugin_positioner::init())
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            if let Some(window) = app.get_webview_window("overlay") {
                let visible = window.is_visible().unwrap_or(false);
                if visible {
                    window.hide().ok();
                } else {
                    use tauri_plugin_clipboard_manager::ClipboardExt;
                    let clip = app.clipboard().read_text().unwrap_or_default();
                    window.emit("selection", clip).ok();
                    window.show().ok();
                    window.set_focus().ok();
                }
            }
        }))
        .plugin(
            tauri_plugin_global_shortcut::Builder::new()
                .with_handler(|app, _shortcut, event| {
                    use tauri_plugin_global_shortcut::ShortcutState;
                    if event.state() == ShortcutState::Pressed {
                        if let Some(window) = app.get_webview_window("overlay") {
                            let visible = window.is_visible().unwrap_or(false);
                            if visible {
                                window.hide().ok();
                            } else {
                                use tauri_plugin_clipboard_manager::ClipboardExt;
                                let clip = app.clipboard().read_text().unwrap_or_default();
                                window.emit("selection", clip).ok();
                                window.show().ok();
                                window.set_focus().ok();
                            }
                        }
                    }
                })
                .build(),
        )
        .setup(|app| {
            use tauri_plugin_global_shortcut::GlobalShortcutExt;

            // Register hotkey from config (or fall back to default)
            let hk = {
                let app_state = app.state::<AppState>();
                let cfg_guard = app_state.config.lock().unwrap();
                get_hotkey(&cfg_guard)
            };
            if let Err(e) = app.global_shortcut().register(hk.as_str()) {
                eprintln!("Warning: failed to register hotkey '{}': {}", hk, e);
            }

            // macOS: hide from Dock
            #[cfg(target_os = "macos")]
            app.set_activation_policy(tauri::ActivationPolicy::Accessory);

            // Show overlay window on startup
            if let Some(window) = app.get_webview_window("overlay") {
                window.show().ok();
                window.set_focus().ok();
            }

            // System tray menu & icon — load the template icon (white PNG, transparent bg)
            use tauri::menu::{Menu, MenuItem};
            use tauri::tray::{TrayIconBuilder, TrayIconEvent};

            let show_i = MenuItem::with_id(app, "show", "Show / Toggle Promptly", true, None::<&str>)?;
            let history_i = MenuItem::with_id(app, "history", "Show History", true, None::<&str>)?;
            let quit_i = MenuItem::with_id(app, "quit", "Force Quit", true, None::<&str>)?;

            let tray_menu = Menu::with_items(app, &[&show_i, &history_i, &quit_i])?;

            let mut tray_builder = TrayIconBuilder::new()
                .tooltip("Promptly")
                .icon(tauri::include_image!("icons/trayTemplate.png"))
                .menu(&tray_menu)
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "show" => {
                        if let Some(window) = app.get_webview_window("overlay") {
                            let visible = window.is_visible().unwrap_or(false);
                            if visible {
                                window.hide().ok();
                            } else {
                                window.show().ok();
                                window.set_focus().ok();
                            }
                        }
                    }
                    "history" => {
                        if let Some(window) = app.get_webview_window("overlay") {
                            window.emit("open-history", ()).ok();
                            window.show().ok();
                            window.set_focus().ok();
                        }
                    }
                    "quit" => {
                        app.exit(0);
                    }
                    _ => {}
                });

            // Tell macOS to treat this as a template image (auto adapts to dark/light mode)
            #[cfg(target_os = "macos")]
            { tray_builder = tray_builder.icon_as_template(true); }

            tray_builder
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click { button: tauri::tray::MouseButton::Left, button_state: tauri::tray::MouseButtonState::Up, .. } = event {
                        let app = tray.app_handle();
                        if let Some(window) = app.get_webview_window("overlay") {
                            let visible = window.is_visible().unwrap_or(false);
                            if visible {
                                window.hide().ok();
                            } else {
                                window.show().ok();
                                window.set_focus().ok();
                            }
                        }
                    }
                })
                .build(app)?;

            Ok(())
        })
        .on_window_event(|window, event| {
            // Hide overlay on focus loss (with 400ms debounce for drag)
            if let tauri::WindowEvent::Focused(false) = event {
                if window.label() == "overlay" {
                    let w = window.clone();
                    std::thread::spawn(move || {
                        std::thread::sleep(Duration::from_millis(400));
                        if !w.is_focused().unwrap_or(true) {
                            w.hide().ok();
                        }
                    });
                }
            }
        })
        .invoke_handler(tauri::generate_handler![
            run_recipe,
            send_chat,
            list_recipes,
            save_recipe,
            update_recipe,
            delete_recipe,
            reorder_recipes,
            get_engine_configs,
            save_engine_config,
            delete_engine_config,
            ping_engine,
            list_history,
            clear_history,
            get_hotkey_cmd,
            set_hotkey,
            get_clipboard,
            hide_window,
            export_data,
            import_data,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
