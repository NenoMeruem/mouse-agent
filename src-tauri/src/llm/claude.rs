// llm/claude.rs — Anthropic Claude streaming client via SSE.
// Mirrors Go's internal/llm/claude/client.go.

use crate::llm::{ChunkSender, LlmClient};
use crate::models::{Chunk, LlmRequest};
use futures_util::StreamExt;
use serde::{Deserialize, Serialize};
use std::time::Duration;

const CLAUDE_BASE_URL: &str = "https://api.anthropic.com/v1";
const ANTHROPIC_VERSION: &str = "2023-06-01";
const DEFAULT_MAX_TOKENS: u32 = 4096;

pub struct ClaudeClient {
    api_key: String,
    model: String,
    timeout: Duration,
}

impl ClaudeClient {
    pub fn new(api_key: impl Into<String>, model: impl Into<String>, timeout: Duration) -> Self {
        Self { api_key: api_key.into(), model: model.into(), timeout }
    }
}

// ---------------------------------------------------------------------------
// Request / Response shapes (Anthropic Messages API)
// ---------------------------------------------------------------------------

#[derive(Serialize)]
struct ClaudeRequest<'a> {
    model: &'a str,
    messages: Vec<ClaudeMessage<'a>>,
    max_tokens: u32,
    stream: bool,
}

#[derive(Serialize)]
struct ClaudeMessage<'a> {
    role: &'a str,
    content: &'a str,
}

#[derive(Deserialize)]
struct StreamEvent {
    #[serde(rename = "type")]
    event_type: String,
    delta: Option<Delta>,
}

#[derive(Deserialize)]
struct Delta {
    #[serde(rename = "type")]
    delta_type: Option<String>,
    text: Option<String>,
}

// ---------------------------------------------------------------------------
// LlmClient impl
// ---------------------------------------------------------------------------

#[async_trait::async_trait]
impl LlmClient for ClaudeClient {
    fn name(&self) -> &str {
        "claude"
    }

    async fn stream(&self, req: LlmRequest, tx: ChunkSender) {
        let messages: Vec<ClaudeMessage> = if req.messages.is_empty() {
            vec![ClaudeMessage { role: "user", content: &req.prompt }]
        } else {
            req.messages
                .iter()
                .map(|m| ClaudeMessage { role: &m.role, content: &m.content })
                .collect()
        };

        let body = ClaudeRequest {
            model: &self.model,
            messages,
            max_tokens: DEFAULT_MAX_TOKENS,
            stream: true,
        };

        let url = format!("{}/messages", CLAUDE_BASE_URL);

        let client = match reqwest::Client::builder().timeout(self.timeout).build() {
            Ok(c) => c,
            Err(e) => {
                let _ = tx.send(Chunk { text: String::new(), done: true, error: Some(e.to_string()) }).await;
                return;
            }
        };

        let resp = match client
            .post(&url)
            .header("x-api-key", &self.api_key)
            .header("anthropic-version", ANTHROPIC_VERSION)
            .header("content-type", "application/json")
            .json(&body)
            .send()
            .await
        {
            Ok(r) => r,
            Err(e) => {
                let _ = tx.send(Chunk { text: String::new(), done: true, error: Some(e.to_string()) }).await;
                return;
            }
        };

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body_text = resp.text().await.unwrap_or_default();
            let _ = tx.send(Chunk {
                text: String::new(),
                done: true,
                error: Some(format!("Claude API error {}: {}", status, body_text)),
            }).await;
            return;
        }

        let mut stream = resp.bytes_stream();
        let mut buf = String::new();

        'outer: while let Some(chunk_result) = stream.next().await {
            match chunk_result {
                Err(e) => {
                    let _ = tx.send(Chunk { text: String::new(), done: true, error: Some(e.to_string()) }).await;
                    return;
                }
                Ok(bytes) => {
                    buf.push_str(&String::from_utf8_lossy(&bytes));
                    while let Some(pos) = buf.find('\n') {
                        let line = buf[..pos].trim().to_string();
                        buf = buf[pos + 1..].to_string();

                        if line.is_empty() {
                            continue;
                        }

                        let data = if let Some(d) = line.strip_prefix("data: ") {
                            d.trim()
                        } else {
                            continue;
                        };

                        if let Ok(event) = serde_json::from_str::<StreamEvent>(data) {
                            match event.event_type.as_str() {
                                "content_block_delta" => {
                                    if let Some(delta) = event.delta {
                                        if delta.delta_type.as_deref() == Some("text_delta") {
                                            if let Some(text) = delta.text {
                                                if !text.is_empty() {
                                                    if tx.send(Chunk { text, done: false, error: None }).await.is_err() {
                                                        break 'outer;
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                                "message_stop" => {
                                    let _ = tx.send(Chunk { text: String::new(), done: true, error: None }).await;
                                    return;
                                }
                                _ => {}
                            }
                        }
                    }
                }
            }
        }

        let _ = tx.send(Chunk { text: String::new(), done: true, error: None }).await;
    }
}

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn client_name_is_claude() {
        let c = ClaudeClient::new("key", "claude-sonnet-4-6", Duration::from_secs(60));
        assert_eq!(c.name(), "claude");
    }

    #[test]
    fn request_payload_serializes_correctly() {
        let model = "claude-sonnet-4-6";
        let msgs = vec![ClaudeMessage { role: "user", content: "Hello Claude" }];
        let payload = ClaudeRequest {
            model,
            messages: msgs,
            max_tokens: DEFAULT_MAX_TOKENS,
            stream: true,
        };
        let json = serde_json::to_string(&payload).unwrap();
        assert!(json.contains("claude-sonnet-4-6"));
        assert!(json.contains("Hello Claude"));
        assert!(json.contains("\"stream\":true"));
        assert!(json.contains("\"max_tokens\":4096"));
    }

    #[test]
    fn stream_event_parses_content_block_delta() {
        let raw = r#"{
            "type": "content_block_delta",
            "delta": {"type": "text_delta", "text": "Hello"}
        }"#;
        let ev: StreamEvent = serde_json::from_str(raw).unwrap();
        assert_eq!(ev.event_type, "content_block_delta");
        let delta = ev.delta.unwrap();
        assert_eq!(delta.delta_type.as_deref(), Some("text_delta"));
        assert_eq!(delta.text.as_deref(), Some("Hello"));
    }

    #[test]
    fn stream_event_parses_message_stop() {
        let raw = r#"{"type": "message_stop"}"#;
        let ev: StreamEvent = serde_json::from_str(raw).unwrap();
        assert_eq!(ev.event_type, "message_stop");
        assert!(ev.delta.is_none());
    }

    #[test]
    fn stream_event_parses_unknown_type_gracefully() {
        let raw = r#"{"type": "ping"}"#;
        let ev: StreamEvent = serde_json::from_str(raw).unwrap();
        assert_eq!(ev.event_type, "ping");
    }
}
