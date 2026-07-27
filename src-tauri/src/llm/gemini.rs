// llm/gemini.rs — Google Gemini streaming client via REST API.
// Endpoint: POST /v1beta/models/{model}:streamGenerateContent?alt=sse
// Auth: x-goog-api-key header (preferred) + ?key= query param as fallback
// SSE format: each "data: {...}" line is a complete GenerateContentResponse JSON.
// Stream ends when candidates[].finishReason is present in a chunk.
// Reference: https://ai.google.dev/gemini-api/docs/text-generation?lang=rest

use crate::llm::{ChunkSender, LlmClient};
use crate::models::{Chunk, LlmRequest};
use futures_util::StreamExt;
use serde::Deserialize;
use std::time::Duration;

const BASE_URL: &str = "https://generativelanguage.googleapis.com/v1beta/models";

pub struct GeminiClient {
    api_key: String,
    model: String,
    timeout: Duration,
}

impl GeminiClient {
    pub fn new(api_key: impl Into<String>, model: impl Into<String>, timeout: Duration) -> Self {
        Self { api_key: api_key.into(), model: model.into(), timeout }
    }
}

// ---------------------------------------------------------------------------
// Response shapes (GenerateContentResponse per chunk)
// ---------------------------------------------------------------------------

#[derive(Debug, Deserialize)]
struct GeminiResponse {
    candidates: Option<Vec<Candidate>>,
    // promptFeedback present in first chunk if prompt was blocked
    #[serde(rename = "promptFeedback")]
    prompt_feedback: Option<serde_json::Value>,
}

#[derive(Debug, Deserialize)]
struct Candidate {
    content: Option<Content>,
    // Only present in the FINAL chunk — signals stream complete
    #[serde(rename = "finishReason")]
    finish_reason: Option<String>,
}

#[derive(Debug, Deserialize)]
struct Content {
    parts: Option<Vec<Part>>,
}

#[derive(Debug, Deserialize)]
struct Part {
    text: Option<String>,
}

// ---------------------------------------------------------------------------
// Request payload shapes
// ---------------------------------------------------------------------------

#[derive(serde::Serialize)]
struct GeminiRequest<'a> {
    contents: Vec<GeminiContent<'a>>,
    #[serde(rename = "generationConfig", skip_serializing_if = "Option::is_none")]
    generation_config: Option<GenerationConfig>,
}

#[derive(serde::Serialize)]
struct GenerationConfig {
    temperature: f32,
}

#[derive(serde::Serialize)]
struct GeminiContent<'a> {
    role: &'a str,
    parts: Vec<GeminiPart<'a>>,
}

#[derive(serde::Serialize)]
struct GeminiPart<'a> {
    text: &'a str,
}

// ---------------------------------------------------------------------------
// LlmClient impl
// ---------------------------------------------------------------------------

#[async_trait::async_trait]
impl LlmClient for GeminiClient {
    fn name(&self) -> &str {
        "gemini"
    }

    async fn stream(&self, req: LlmRequest, tx: ChunkSender) {
        // Build contents array — multi-turn or single-turn
        let contents: Vec<GeminiContent> = if req.messages.is_empty() {
            vec![GeminiContent {
                role: "user",
                parts: vec![GeminiPart { text: &req.prompt }],
            }]
        } else {
            req.messages
                .iter()
                .map(|m| GeminiContent {
                    // Gemini uses "model" instead of "assistant"
                    role: if m.role == "assistant" { "model" } else { "user" },
                    parts: vec![GeminiPart { text: &m.content }],
                })
                .collect()
        };

        let body = GeminiRequest {
            contents,
            generation_config: Some(GenerationConfig { temperature: 0.7 }),
        };

        // URL: alt=sse tells the API to return Server-Sent Events stream
        let url = format!(
            "{}/{}:streamGenerateContent?alt=sse&key={}",
            BASE_URL, self.model, self.api_key
        );

        let client = match reqwest::Client::builder()
            .timeout(self.timeout)
            .build()
        {
            Ok(c) => c,
            Err(e) => {
                let _ = tx.send(Chunk { text: String::new(), done: true, error: Some(e.to_string()) }).await;
                return;
            }
        };

        let resp = match client
            .post(&url)
            // x-goog-api-key header is the preferred auth method per Google docs
            .header("x-goog-api-key", &self.api_key)
            .header("Content-Type", "application/json")
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
                error: Some(format!("Gemini API error {}: {}", status, body_text)),
            }).await;
            return;
        }

        // Stream SSE — each line: "data: <GenerateContentResponse JSON>"
        // Stream ends when a chunk has candidates[].finishReason set (e.g. "STOP")
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

                    // Process all complete lines in the buffer
                    while let Some(pos) = buf.find('\n') {
                        let line = buf[..pos].trim().to_string();
                        buf = buf[pos + 1..].to_string();

                        if line.is_empty() {
                            continue;
                        }

                        // Skip SSE comment / event-type lines
                        if line.starts_with(':') || line.starts_with("event:") {
                            continue;
                        }

                        // Strip "data: " prefix
                        let data = match line.strip_prefix("data: ") {
                            Some(d) => d.trim(),
                            None => continue,
                        };

                        // Gemini does NOT send [DONE] — end is signalled by finishReason
                        if data == "[DONE]" {
                            break 'outer;
                        }

                        match serde_json::from_str::<GeminiResponse>(data) {
                            Ok(gr) => {
                                // Check if prompt was blocked
                                if let Some(ref pf) = gr.prompt_feedback {
                                    if let Some(reason) = pf.get("blockReason").and_then(|v| v.as_str()) {
                                        let _ = tx.send(Chunk {
                                            text: String::new(),
                                            done: true,
                                            error: Some(format!("Prompt blocked: {}", reason)),
                                        }).await;
                                        return;
                                    }
                                }

                                for candidate in gr.candidates.unwrap_or_default() {
                                    // Extract text from parts
                                    if let Some(content) = candidate.content {
                                        for part in content.parts.unwrap_or_default() {
                                            if let Some(text) = part.text {
                                                if !text.is_empty() && tx.send(Chunk { text, done: false, error: None }).await.is_err() {
                                                    return;
                                                }
                                            }
                                        }
                                    }

                                    // finishReason present → stream complete
                                    if candidate.finish_reason.is_some() {
                                        break 'outer;
                                    }
                                }
                            }
                            Err(_) => {
                                // Non-JSON data line (e.g. keep-alive) — skip silently
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
    fn client_name_is_gemini() {
        let c = GeminiClient::new("key", "gemini-3.1-flash-lite", Duration::from_secs(30));
        assert_eq!(c.name(), "gemini");
    }

    #[test]
    fn gemini_request_payload_single_turn() {
        let parts = vec![GeminiPart { text: "hello" }];
        let contents = vec![GeminiContent { role: "user", parts }];
        let req = GeminiRequest { contents, generation_config: None };
        let json = serde_json::to_string(&req).unwrap();
        assert!(json.contains("\"role\":\"user\""));
        assert!(json.contains("\"text\":\"hello\""));
    }

    #[test]
    fn gemini_request_includes_generation_config() {
        let parts = vec![GeminiPart { text: "hi" }];
        let contents = vec![GeminiContent { role: "user", parts }];
        let req = GeminiRequest {
            contents,
            generation_config: Some(GenerationConfig { temperature: 0.7 }),
        };
        let json = serde_json::to_string(&req).unwrap();
        assert!(json.contains("generationConfig"));
        assert!(json.contains("temperature"));
    }

    #[test]
    fn gemini_response_parses_text_parts() {
        let raw = r#"{
            "candidates": [{
                "content": {
                    "parts": [{"text": "Hello world"}]
                }
            }]
        }"#;
        let resp: GeminiResponse = serde_json::from_str(raw).unwrap();
        let candidates = resp.candidates.unwrap();
        let text = candidates[0]
            .content.as_ref().unwrap()
            .parts.as_ref().unwrap()[0]
            .text.as_ref().unwrap();
        assert_eq!(text, "Hello world");
    }

    #[test]
    fn gemini_response_detects_finish_reason() {
        let raw = r#"{
            "candidates": [{
                "content": {"parts": [{"text": "done"}]},
                "finishReason": "STOP"
            }]
        }"#;
        let resp: GeminiResponse = serde_json::from_str(raw).unwrap();
        let c = &resp.candidates.unwrap()[0];
        assert_eq!(c.finish_reason.as_deref(), Some("STOP"));
    }

    #[test]
    fn gemini_response_handles_missing_candidates() {
        let raw = r#"{}"#;
        let resp: GeminiResponse = serde_json::from_str(raw).unwrap();
        assert!(resp.candidates.is_none());
    }

    #[test]
    fn gemini_url_contains_alt_sse_and_key() {
        let client = GeminiClient::new("MY_KEY", "gemini-3.1-flash-lite", Duration::from_secs(30));
        let url = format!(
            "{}/{}:streamGenerateContent?alt=sse&key={}",
            BASE_URL, client.model, client.api_key
        );
        assert!(url.contains("alt=sse"));
        assert!(url.contains("MY_KEY"));
        assert!(url.contains("gemini-3.1-flash-lite"));
    }
}
