// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::{AppHandle, Manager};
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
        .sidecar("prompt-agent")
        .map_err(|e| e.to_string())?
        .args(&args);

    let (mut rx, mut child) = sidecar.spawn().map_err(|e| e.to_string())?;

    // Write selection text to stdin, then close stdin
    if !selection.is_empty() {
        child.write(selection.as_bytes()).ok();
    }

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
        .sidecar("prompt-agent")
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
            // Hide overlay when it loses focus
            if let tauri::WindowEvent::Focused(false) = event {
                if window.label() == "overlay" {
                    window.hide().ok();
                }
            }
        })
        .invoke_handler(tauri::generate_handler![run_recipe, list_recipes, hide_window])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
