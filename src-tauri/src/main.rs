// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::{AppHandle, Emitter, Manager};
use tauri_plugin_shell::ShellExt;
use tauri_plugin_shell::process::CommandEvent;

/// Run a recipe by spawning the Go sidecar, piping selection text via stdin,
/// and streaming stdout chunks back to the WebView as events.
#[tauri::command]
async fn run_recipe(
    app: AppHandle,
    recipe_id: String,
    selection: String,
    engine: String,
    tone: String,
    length: String,
    complexity: String,
) -> Result<(), String> {
    let mut args = vec![
        "run".to_string(),
        recipe_id.clone(),
        "--no-select".to_string(),
        "--raw".to_string(),
    ];
    if !engine.is_empty() {
        args.push("--engine".to_string());
        args.push(engine);
    }
    if !tone.is_empty() {
        args.push("--tone".to_string());
        args.push(tone);
    }
    if !length.is_empty() {
        args.push("--length".to_string());
        args.push(length);
    }
    if !complexity.is_empty() {
        args.push("--complexity".to_string());
        args.push(complexity);
    }

    let sidecar = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(&args);

    let (mut rx, mut child) = sidecar.spawn().map_err(|e| e.to_string())?;

    // Write selection text to stdin, then close stdin so Go's io.ReadAll unblocks
    if !selection.is_empty() {
        child.write(selection.as_bytes()).ok();
    }
    drop(child); // closes stdin pipe → Go's io.ReadAll returns EOF

    let app_clone = app.clone();
    tauri::async_runtime::spawn(async move {
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    let text = String::from_utf8_lossy(&line).to_string();
                    app_clone.emit("chunk", text).ok();
                }
                CommandEvent::Stderr(line) => {
                    let text = String::from_utf8_lossy(&line).to_string();
                    app_clone.emit("error", text).ok();
                }
                CommandEvent::Terminated(_) => {
                    app_clone.emit("done", ()).ok();
                    break;
                }
                _ => {}
            }
        }
    });

    Ok(())
}

/// List all recipes by running `prompt-agent prompt list --output json`
#[tauri::command]
async fn list_recipes(app: AppHandle) -> Result<String, String> {
    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(["prompt", "list", "--output", "json"])
        .output()
        .await
        .map_err(|e| e.to_string())?;

    Ok(String::from_utf8_lossy(&output.stdout).to_string())
}

/// Hide the overlay window
#[tauri::command]
fn hide_window(app: AppHandle) {
    if let Some(window) = app.get_webview_window("overlay") {
        window.hide().ok();
    }
}

/// Save (create) a new recipe by calling the Go sidecar with flags
#[tauri::command]
async fn save_recipe(
    app: AppHandle,
    id: String,
    name: String,
    description: String,
    template: String,
    engine: String,
    params: String,
    icon: String,
) -> Result<String, String> {
    let mut args: Vec<String> = vec![
        "prompt".into(), "add".into(),
        "--id".into(), id,
        "--name".into(), name,
        "--template".into(), template,
    ];
    if !description.is_empty() {
        args.push("--description".into());
        args.push(description);
    }
    if !engine.is_empty() {
        args.push("--engine".into());
        args.push(engine);
    }
    if !params.is_empty() {
        args.push("--params".into());
        args.push(params);
    }
    if !icon.is_empty() {
        args.push("--icon".into());
        args.push(icon);
    }

    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(&args)
        .output()
        .await
        .map_err(|e| e.to_string())?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

/// Reorder recipes by providing IDs in the desired display order
#[tauri::command]
async fn reorder_recipes(app: AppHandle, ids: Vec<String>) -> Result<String, String> {
    let mut args = vec!["prompt".to_string(), "reorder".to_string()];
    args.extend(ids);

    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(&args)
        .output()
        .await
        .map_err(|e| e.to_string())?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

/// Delete a recipe by ID
#[tauri::command]
async fn delete_recipe(app: AppHandle, id: String) -> Result<String, String> {
    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(["prompt", "delete", &id, "--yes"])
        .output()
        .await
        .map_err(|e| e.to_string())?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

/// Get engine configs as JSON
#[tauri::command]
async fn get_engine_configs(app: AppHandle) -> Result<String, String> {
    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(["config", "get-engines"])
        .output()
        .await
        .map_err(|e| e.to_string())?;
    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

/// Save a single engine config (api_key + model)
#[tauri::command]
async fn save_engine_config(
    app: AppHandle,
    engine: String,
    api_key: String,
    model: String,
) -> Result<String, String> {
    let mut args = vec!["config".to_string(), "set-engine".to_string(), engine];
    args.extend(["--api-key".to_string(), api_key]);
    args.extend(["--model".to_string(), model]);

    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(&args)
        .output()
        .await
        .map_err(|e| e.to_string())?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

/// Delete an engine config by name
#[tauri::command]
async fn delete_engine_config(app: AppHandle, engine: String) -> Result<String, String> {
    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(["config", "delete-engine", &engine])
        .output()
        .await
        .map_err(|e| e.to_string())?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

/// Ping an engine to check if API key is valid and within quota
#[tauri::command]
async fn ping_engine(app: AppHandle, engine: String) -> Result<String, String> {
    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(["config", "ping-engine", &engine])
        .output()
        .await
        .map_err(|e| e.to_string())?;

    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

/// List run history as JSON (limit 50)
#[tauri::command]
async fn list_history(app: AppHandle) -> Result<String, String> {
    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(["history", "--limit", "50", "--output-json"])
        .output()
        .await
        .map_err(|e| e.to_string())?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

/// Clear all run history
#[tauri::command]
async fn clear_history(app: AppHandle) -> Result<String, String> {
    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(["history", "clear", "--all"])
        .output()
        .await
        .map_err(|e| e.to_string())?;

    if output.status.success() {
        Ok("cleared".to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

/// Update an existing recipe via flags
#[tauri::command]
async fn update_recipe(
    app: AppHandle,
    id: String,
    name: String,
    description: String,
    template: String,
    engine: String,
    params: String,
    icon: String,
) -> Result<String, String> {
    let mut args: Vec<String> = vec!["prompt".into(), "update".into(), id];
    if !name.is_empty()        { args.extend(["--name".into(), name]); }
    if !description.is_empty() { args.extend(["--description".into(), description]); }
    if !template.is_empty()    { args.extend(["--template".into(), template]); }
    if !engine.is_empty()      { args.extend(["--engine".into(), engine]); }
    args.extend(["--params".into(), params]);
    if !icon.is_empty()        { args.extend(["--icon".into(), icon]); }

    let output = app
        .shell()
        .sidecar("prompt-agent-cli")
        .map_err(|e| e.to_string())?
        .args(&args)
        .output()
        .await
        .map_err(|e| e.to_string())?;

    if output.status.success() {
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    } else {
        Err(String::from_utf8_lossy(&output.stderr).to_string())
    }
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_clipboard_manager::init())
        .plugin(tauri_plugin_positioner::init())
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
                                // Read clipboard and emit to UI before showing
                                use tauri_plugin_clipboard_manager::ClipboardExt;
                                let clip = app
                                    .clipboard()
                                    .read_text()
                                    .unwrap_or_default();
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
            // Register global hotkey Alt+Space
            use tauri_plugin_global_shortcut::GlobalShortcutExt;
            app.global_shortcut().register("Alt+Space")?;

            // System tray
            use tauri::tray::{TrayIconBuilder, TrayIconEvent};
            TrayIconBuilder::new()
                .tooltip("Prompt Agent")
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click { .. } = event {
                        let app = tray.app_handle();
                        if let Some(window) = app.get_webview_window("overlay") {
                            window.show().ok();
                            window.set_focus().ok();
                        }
                    }
                })
                .build(app)?;

            // macOS: hide from dock
            #[cfg(target_os = "macos")]
            app.set_activation_policy(tauri::ActivationPolicy::Accessory);

            Ok(())
        })
        .on_window_event(|window, event| {
            // Hide overlay when it loses focus — but delay to avoid hiding during drag.
            // On macOS, WKWebView briefly fires Focused(false) when a native drag starts,
            // which would immediately hide the window. We wait 400ms and re-check focus
            // so a genuine click-outside still hides, but drags are not interrupted.
            if let tauri::WindowEvent::Focused(false) = event {
                if window.label() == "overlay" {
                    let w = window.clone();
                    std::thread::spawn(move || {
                        std::thread::sleep(std::time::Duration::from_millis(400));
                        if !w.is_focused().unwrap_or(true) {
                            w.hide().ok();
                        }
                    });
                }
            }
        })
        .invoke_handler(tauri::generate_handler![run_recipe, list_recipes, hide_window, save_recipe, update_recipe, delete_recipe, reorder_recipes, get_engine_configs, save_engine_config, delete_engine_config, ping_engine, list_history, clear_history])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
