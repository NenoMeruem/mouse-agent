// llm/openai.rs — OpenAI Chat Completions streaming client via SSE.
// Mirrors Go's internal/llm/openai/client.go.

use crate::llm::{ChunkSender, LlmClient};
use crate::models::{Chunk, LlmRequest};
use futures_util::StreamExt;
use serde::{Deserialize, Serialize};
use std::time::Duration;

const OPENAI_BASE_URL: &str = "https://api.openai.com/v1";

pub struct OpenAiClient {
    api_key: String,
    model: String,
    timeout: Duration,
    base_url: String,
}

impl OpenAiClient {
    pub fn new(api_key: impl Into<String>, model: impl Into<String>, timeout: Duration) -> Self {
        Self {
            api_key: api_key.into(),
            model: model.into(),
            timeout,
            base_url: OPENAI_BASE_URL.to_string(),
        }
    }

    /// Override base URL (useful for OpenAI-compatible endpoints).
    #[allow(dead_code)]
    pub fn with_base_url(mut self, url: impl Into<String>) -> Self {
        self.base_url = url.into();
        self
    }
}

// ---------------------------------------------------------------------------
// Request / Response shapes
// ---------------------------------------------------------------------------

#[derive(Serialize)]
struct OpenAiRequest<'a> {
    model: &'a str,
    messages: Vec<OpenAiMessage<'a>>,
    stream: bool,
    temperature: f32,
}

#[derive(Serialize)]
struct OpenAiMessage<'a> {
    role: &'a str,
    content: &'a str,
}

#[derive(Deserialize)]
struct StreamResponse {
    choices: Vec<StreamChoice>,
}

#[derive(Deserialize)]
struct StreamChoice {
    delta: Delta,
}

#[derive(Deserialize)]
struct Delta {
    content: Option<String>,
}

// ---------------------------------------------------------------------------
// LlmClient impl
// ---------------------------------------------------------------------------

#[async_trait::async_trait]
impl LlmClient for OpenAiClient {
    fn name(&self) -> &str {
        "openai"
    }

    async fn stream(&self, req: LlmRequest, tx: ChunkSender) {
        let messages: Vec<OpenAiMessage> = if req.messages.is_empty() {
            vec![OpenAiMessage { role: "user", content: &req.prompt }]
        } else {
            req.messages
                .iter()
                .map(|m| OpenAiMessage { role: &m.role, content: &m.content })
                .collect()
        };

        let body = OpenAiRequest {
            model: &self.model,
            messages,
            stream: true,
            temperature: 0.7,
        };

        let url = format!("{}/chat/completions", self.base_url);

        let client = match reqwest::Client::builder().timeout(self.timeout).build() {
            Ok(c) => c,
            Err(e) => {
                let _ = tx.send(Chunk { text: String::new(), done: true, error: Some(e.to_string()) }).await;
                return;
            }
        };

        let resp = match client
            .post(&url)
            .bearer_auth(&self.api_key)
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
                error: Some(format!("OpenAI API error {}: {}", status, body_text)),
            }).await;
            return;
        }

        let mut stream = resp.bytes_stream();
        let mut buf = String::new();

        while let Some(chunk_result) = stream.next().await {
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

                        if line == "data: [DONE]" {
                            let _ = tx.send(Chunk { text: String::new(), done: true, error: None }).await;
                            return;
                        }

                        let data = if let Some(d) = line.strip_prefix("data: ") {
                            d.trim()
                        } else {
                            continue;
                        };

                        if let Ok(sse) = serde_json::from_str::<StreamResponse>(data) {
                            for choice in sse.choices {
                                if let Some(text) = choice.delta.content {
                                    if !text.is_empty() && tx.send(Chunk { text, done: false, error: None }).await.is_err() {
                                        return;
                                    }
                                }
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
    fn client_name_is_openai() {
        let c = OpenAiClient::new("key", "gpt-4o-mini", Duration::from_secs(60));
        assert_eq!(c.name(), "openai");
    }

    #[test]
    fn request_payload_serializes_correctly_single_turn() {
        // Build a sample prompt message manually to test serde output
        let model = "gpt-4o-mini";
        let prompt_text = "Hello OpenAI";
        let msgs = vec![OpenAiMessage { role: "user", content: prompt_text }];
        let payload = OpenAiRequest {
            model,
            messages: msgs,
            stream: true,
            temperature: 0.7,
        };
        let json = serde_json::to_string(&payload).unwrap();
        assert!(json.contains("gpt-4o-mini"));
        assert!(json.contains("Hello OpenAI"));
        assert!(json.contains("\"stream\":true"));
    }

    #[test]
    fn stream_response_parses_delta_content() {
        let raw = r#"{"choices":[{"delta":{"content":"Hello"}}]}"#;
        let resp: StreamResponse = serde_json::from_str(raw).unwrap();
        assert_eq!(resp.choices[0].delta.content.as_deref(), Some("Hello"));
    }

    #[test]
    fn stream_response_handles_empty_delta() {
        let raw = r#"{"choices":[{"delta":{}}]}"#;
        let resp: StreamResponse = serde_json::from_str(raw).unwrap();
        assert!(resp.choices[0].delta.content.is_none());
    }

    #[test]
    fn stream_response_handles_no_choices() {
        let raw = r#"{"choices":[]}"#;
        let resp: StreamResponse = serde_json::from_str(raw).unwrap();
        assert!(resp.choices.is_empty());
    }
}
