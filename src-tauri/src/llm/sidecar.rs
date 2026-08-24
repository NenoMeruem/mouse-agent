// llm/sidecar.rs — FastAPI LangChain Sidecar streaming client.
// Connects to http://127.0.0.1:8000/api/recipes/run or /api/agent/stream via HTTP POST with SSE streaming,
// passing resolved API keys directly from Rust to FastAPI.

use crate::llm::{ChunkSender, LlmClient};
use crate::models::{Chunk, LlmRequest};
use futures_util::StreamExt;
use serde_json::json;
use std::time::Duration;

pub struct SidecarClient {
    pub base_url: String,
    pub provider: String,
    pub model: String,
    pub api_key: String,
    pub timeout: Duration,
}

impl SidecarClient {
    pub fn new(
        base_url: impl Into<String>,
        provider: impl Into<String>,
        model: impl Into<String>,
        api_key: impl Into<String>,
        timeout: Duration,
    ) -> Self {
        Self {
            base_url: base_url.into(),
            provider: provider.into(),
            model: model.into(),
            api_key: api_key.into(),
            timeout,
        }
    }
}

/// Helper function to parse raw SSE text buffer and extract chunks.
pub fn parse_sse_buffer(
    buffer: &mut String,
    current_event: &mut String,
) -> Vec<Chunk> {
    let mut chunks = Vec::new();

    while let Some(pos) = buffer.find('\n') {
        let mut line = buffer[..pos].to_string();
        buffer.drain(..pos + 1);

        if line.ends_with('\r') {
            line.pop();
        }

        let trimmed = line.trim();
        if trimmed.is_empty() {
            *current_event = "message".to_string();
            continue;
        }

        if let Some(ev) = trimmed.strip_prefix("event:") {
            *current_event = ev.trim().to_string();
        } else if let Some(dt) = trimmed.strip_prefix("data:") {
            let data_str = dt.trim();
            if data_str.is_empty() {
                continue;
            }

            if let Ok(parsed) = serde_json::from_str::<serde_json::Value>(data_str) {
                match current_event.as_str() {
                    "chunk" => {
                        let text = if let Some(t) = parsed.get("text").and_then(|v| v.as_str()) {
                            t.to_string()
                        } else if parsed.is_string() {
                            parsed.as_str().unwrap_or_default().to_string()
                        } else {
                            parsed.to_string()
                        };

                        if !text.is_empty() {
                            chunks.push(Chunk {
                                text,
                                done: false,
                                error: None,
                            });
                        }
                    }
                    "error" => {
                        let err_text = parsed
                            .get("message")
                            .and_then(|m| m.as_str())
                            .unwrap_or(data_str);
                        chunks.push(Chunk {
                            text: String::new(),
                            done: true,
                            error: Some(err_text.to_string()),
                        });
                    }
                    "done" => {
                        chunks.push(Chunk {
                            text: String::new(),
                            done: true,
                            error: None,
                        });
                    }
                    _ => {
                        // Fallback for default or unhandled events with a "text" field
                        if let Some(t) = parsed.get("text").and_then(|v| v.as_str()) {
                            chunks.push(Chunk {
                                text: t.to_string(),
                                done: false,
                                error: None,
                            });
                        }
                    }
                }
            } else if *current_event == "chunk" || *current_event == "message" {
                chunks.push(Chunk {
                    text: data_str.to_string(),
                    done: false,
                    error: None,
                });
            }
        }
    }

    chunks
}

#[async_trait::async_trait]
impl LlmClient for SidecarClient {
    fn name(&self) -> &str {
        "sidecar"
    }

    async fn stream(&self, req: LlmRequest, tx: ChunkSender) {
        let endpoint = if req.messages.is_empty() {
            format!("{}/api/recipes/run", self.base_url)
        } else {
            format!("{}/api/agent/stream", self.base_url)
        };

        let effective_model = if !req.model.is_empty() {
            &req.model
        } else {
            &self.model
        };

        let payload = if req.messages.is_empty() {
            json!({
                "recipe_id": "prompt-stream",
                "selection": req.prompt,
                "provider": self.provider,
                "model": effective_model,
                "api_key": self.api_key,
                "parameters": {
                    "provider": self.provider,
                    "model": effective_model,
                    "api_key": self.api_key,
                    "template": "{{selection}}"
                }
            })
        } else {
            // Multi-turn chat / agent execution
            let history_json: Vec<serde_json::Value> = req
                .messages
                .iter()
                .map(|m| {
                    json!({
                        "role": m.role,
                        "content": m.content
                    })
                })
                .collect();

            let last_user_prompt = req
                .messages
                .iter()
                .rev()
                .find(|m| m.role == "user")
                .map(|m| m.content.clone())
                .unwrap_or_else(|| req.prompt.clone());

            json!({
                "prompt": last_user_prompt,
                "history": history_json,
                "provider": self.provider,
                "model": effective_model,
                "api_key": self.api_key,
                "enable_web_search": true,
                "enable_local_tools": true
            })
        };

        let client = match reqwest::Client::builder().timeout(self.timeout).build() {
            Ok(c) => c,
            Err(e) => {
                let _ = tx
                    .send(Chunk {
                        text: String::new(),
                        done: true,
                        error: Some(format!("Failed to create HTTP client: {}", e)),
                    })
                    .await;
                return;
            }
        };

        let mut req_builder = client
            .post(&endpoint)
            .header("Content-Type", "application/json")
            .header("Accept", "text/event-stream");

        if !self.api_key.is_empty() {
            req_builder = req_builder
                .header("X-API-Key", &self.api_key)
                .bearer_auth(&self.api_key);
        }

        let resp = match req_builder.json(&payload).send().await {
            Ok(r) => r,
            Err(e) => {
                let err_msg = if e.is_connect() {
                    format!(
                        "Could not connect to FastAPI LangChain sidecar at {}. Please make sure the server is running (run 'make dev'). Error: {}",
                        self.base_url, e
                    )
                } else {
                    format!("Sidecar request failed: {}", e)
                };
                let _ = tx
                    .send(Chunk {
                        text: String::new(),
                        done: true,
                        error: Some(err_msg),
                    })
                    .await;
                return;
            }
        };

        if !resp.status().is_success() {
            let status = resp.status();
            let body = resp.text().await.unwrap_or_default();
            let _ = tx
                .send(Chunk {
                    text: String::new(),
                    done: true,
                    error: Some(format!("Sidecar error HTTP {}: {}", status, body)),
                })
                .await;
            return;
        }

        // Process SSE stream with robust line ending handling (\r\n and \n)
        let mut stream = resp.bytes_stream();
        let mut buffer = String::new();
        let mut current_event = "message".to_string();

        while let Some(item) = stream.next().await {
            let bytes = match item {
                Ok(b) => b,
                Err(e) => {
                    let _ = tx
                        .send(Chunk {
                            text: String::new(),
                            done: true,
                            error: Some(format!("Stream read error: {}", e)),
                        })
                        .await;
                    return;
                }
            };

            buffer.push_str(&String::from_utf8_lossy(&bytes));

            let parsed_chunks = parse_sse_buffer(&mut buffer, &mut current_event);
            for chunk in parsed_chunks {
                let is_done = chunk.done;
                let _ = tx.send(chunk).await;
                if is_done {
                    return;
                }
            }
        }

        // Final done chunk if stream ended without explicit done event
        let _ = tx
            .send(Chunk {
                text: String::new(),
                done: true,
                error: None,
            })
            .await;
    }
}

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sidecar_client_creates_correctly() {
        let client = SidecarClient::new(
            "http://127.0.0.1:8000",
            "gemini",
            "gemini-2.5-flash",
            "test-api-key",
            Duration::from_secs(30),
        );
        assert_eq!(client.name(), "sidecar");
        assert_eq!(client.base_url, "http://127.0.0.1:8000");
        assert_eq!(client.api_key, "test-api-key");
    }

    #[test]
    fn parse_sse_buffer_handles_crlf() {
        let mut buffer = "event: chunk\r\ndata: {\"text\": \"Hello World\"}\r\n\r\nevent: done\r\ndata: {}\r\n\r\n".to_string();
        let mut current_event = "message".to_string();
        let chunks = parse_sse_buffer(&mut buffer, &mut current_event);

        assert_eq!(chunks.len(), 2);
        assert_eq!(chunks[0].text, "Hello World");
        assert!(!chunks[0].done);
        assert!(chunks[1].done);
    }
}
