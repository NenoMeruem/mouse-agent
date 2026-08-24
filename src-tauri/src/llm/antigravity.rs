// llm/antigravity.rs — Google Antigravity Agent API client via REST Interactions.
// Endpoint: POST /v1beta/interactions
// Auth: x-goog-api-key header + Api-Revision: 2026-05-20
// Reference: https://aistudio.google.com/u/1/docs/antigravity-agent?codelanguage=rest

use crate::llm::{ChunkSender, LlmClient};
use crate::models::{Chunk, LlmRequest};
use serde::{Deserialize, Serialize};
use std::time::Duration;

const INTERACTIONS_URL: &str = "https://generativelanguage.googleapis.com/v1beta/interactions";
pub const DEFAULT_API_REVISION: &str = "2026-05-20";

#[derive(Clone, Debug)]
pub struct AntigravityClient {
    pub api_key: String,
    pub agent: String,
    pub api_revision: String,
    pub environment: String,
    pub timeout: Duration,
}

impl AntigravityClient {
    pub fn new(api_key: impl Into<String>, agent: impl Into<String>, timeout: Duration) -> Self {
        let agent_str = agent.into();
        let final_agent = if agent_str.is_empty() {
            "antigravity-preview-05-2026".to_string()
        } else {
            agent_str
        };

        let effective_timeout = if timeout.is_zero() {
            Duration::from_secs(300)
        } else {
            timeout
        };

        Self {
            api_key: api_key.into(),
            agent: final_agent,
            api_revision: DEFAULT_API_REVISION.to_string(),
            environment: "remote".to_string(),
            timeout: effective_timeout,
        }
    }
}

// ---------------------------------------------------------------------------
// Request and Response Shapes
// ---------------------------------------------------------------------------

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct InteractionRequest {
    pub agent: String,
    pub input: String,
    #[serde(default = "default_environment")]
    pub environment: String,
}

fn default_environment() -> String {
    "remote".to_string()
}

#[derive(Debug, Clone, Deserialize)]
pub struct InteractionResponse {
    pub id: Option<String>,
    pub status: Option<String>,
    pub agent: Option<String>,
    pub steps: Option<Vec<InteractionStep>>,
    pub usage: Option<serde_json::Value>,
    pub error: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct InteractionStep {
    #[serde(rename = "type")]
    pub step_type: Option<String>,
    pub content: Option<Vec<InteractionContent>>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct InteractionContent {
    #[serde(rename = "type")]
    pub content_type: Option<String>,
    pub text: Option<String>,
    pub name: Option<String>,
    pub arguments: Option<serde_json::Value>,
}

// ---------------------------------------------------------------------------
// LlmClient Implementation
// ---------------------------------------------------------------------------

#[async_trait::async_trait]
impl LlmClient for AntigravityClient {
    fn name(&self) -> &str {
        "antigravity"
    }

    async fn stream(&self, req: LlmRequest, tx: ChunkSender) {
        // Build prompt text: either from single prompt or combined multi-turn messages
        let prompt_text = if !req.messages.is_empty() {
            let mut buf = String::new();
            for msg in &req.messages {
                let role_label = if msg.role == "user" { "User" } else { "Agent" };
                buf.push_str(&format!("{}: {}\n\n", role_label, msg.content));
            }
            buf.trim().to_string()
        } else {
            req.prompt
        };

        let body = InteractionRequest {
            agent: if req.model.is_empty() { self.agent.clone() } else { req.model },
            input: prompt_text,
            environment: self.environment.clone(),
        };

        let client = match reqwest::Client::builder()
            .timeout(self.timeout)
            .build()
        {
            Ok(c) => c,
            Err(e) => {
                let _ = tx.send(Chunk {
                    text: String::new(),
                    done: true,
                    error: Some(format!("Failed to build HTTP client: {}", e)),
                }).await;
                return;
            }
        };

        let resp = match client
            .post(INTERACTIONS_URL)
            .header("Content-Type", "application/json")
            .header("x-goog-api-key", &self.api_key)
            .header("Api-Revision", &self.api_revision)
            .json(&body)
            .send()
            .await
        {
            Ok(r) => r,
            Err(e) => {
                let err_msg = if e.is_timeout() {
                    format!("Antigravity Agent timed out after {}s waiting for remote agent response.", self.timeout.as_secs())
                } else if e.is_connect() {
                    format!("Antigravity Agent connection error: {}", e)
                } else {
                    format!("Antigravity Agent request failed: {}", e)
                };
                let _ = tx.send(Chunk {
                    text: String::new(),
                    done: true,
                    error: Some(err_msg),
                }).await;
                return;
            }
        };

        if !resp.status().is_success() {
            let status = resp.status().as_u16();
            let body_text = resp.text().await.unwrap_or_default();
            let error_msg = if let Ok(parsed) = serde_json::from_str::<serde_json::Value>(&body_text) {
                if let Some(msg) = parsed.get("error").and_then(|e| e.get("message")).and_then(|m| m.as_str()) {
                    format!("Antigravity API error ({}): {}", status, msg)
                } else {
                    format!("Antigravity API error {}: {}", status, body_text)
                }
            } else {
                format!("Antigravity API error {}: {}", status, body_text)
            };

            let _ = tx.send(Chunk {
                text: String::new(),
                done: true,
                error: Some(error_msg),
            }).await;
            return;
        }

        let resp_json: InteractionResponse = match resp.json().await {
            Ok(parsed) => parsed,
            Err(e) => {
                let _ = tx.send(Chunk {
                    text: String::new(),
                    done: true,
                    error: Some(format!("Failed to parse Antigravity response: {}", e)),
                }).await;
                return;
            }
        };

        if let Some(ref err) = resp_json.error {
            let _ = tx.send(Chunk {
                text: String::new(),
                done: true,
                error: Some(format!("Agent returned error: {}", err)),
            }).await;
            return;
        }

        let mut emitted_any = false;

        if let Some(steps) = resp_json.steps {
            for step in steps {
                if let Some(contents) = step.content {
                    for c in contents {
                        if let Some(text) = c.text {
                            if !text.is_empty() {
                                emitted_any = true;
                                if tx.send(Chunk { text, done: false, error: None }).await.is_err() {
                                    return;
                                }
                            }
                        }
                    }
                }
            }
        }

        if !emitted_any {
            let default_msg = match resp_json.status.as_deref() {
                Some(s) => format!("Interaction completed with status: {}", s),
                None => "Interaction completed successfully.".to_string(),
            };
            let _ = tx.send(Chunk { text: default_msg, done: false, error: None }).await;
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
    fn client_name_is_antigravity() {
        let c = AntigravityClient::new("fake-key", "antigravity-preview-05-2026", Duration::from_secs(60));
        assert_eq!(c.name(), "antigravity");
        assert_eq!(c.agent, "antigravity-preview-05-2026");
        assert_eq!(c.api_revision, "2026-05-20");
    }

    #[test]
    fn interaction_request_serializes_correctly() {
        let req = InteractionRequest {
            agent: "antigravity-preview-05-2026".into(),
            input: "Summarize Hacker News".into(),
            environment: "remote".into(),
        };
        let json = serde_json::to_string(&req).unwrap();
        assert!(json.contains("\"agent\":\"antigravity-preview-05-2026\""));
        assert!(json.contains("\"input\":\"Summarize Hacker News\""));
        assert!(json.contains("\"environment\":\"remote\""));
    }

    #[test]
    fn interaction_response_parses_model_output() {
        let raw = r#"{
            "id": "v1_test_123",
            "status": "completed",
            "agent": "antigravity-preview-05-2026",
            "steps": [
                {
                    "type": "model_output",
                    "content": [
                        {
                            "type": "text",
                            "text": "Top 10 Hacker News stories summarized."
                        }
                    ]
                }
            ]
        }"#;

        let res: InteractionResponse = serde_json::from_str(raw).unwrap();
        assert_eq!(res.status.as_deref(), Some("completed"));
        let steps = res.steps.unwrap();
        assert_eq!(steps.len(), 1);
        let text = steps[0].content.as_ref().unwrap()[0].text.as_ref().unwrap();
        assert_eq!(text, "Top 10 Hacker News stories summarized.");
    }

    #[tokio::test]
    #[ignore]
    async fn test_live_call() {
        let home = dirs::home_dir().unwrap();
        let cfg_path = home.join(".promptly").join("config.yaml");
        if !cfg_path.exists() {
            return;
        }
        let content = std::fs::read_to_string(&cfg_path).unwrap();
        let cfg: crate::config::AppConfig = serde_yaml::from_str(&content).unwrap();
        let key = crate::config::get_engine_api_key(&cfg, "antigravity-main");
        if key.is_empty() {
            return;
        }

        let client = AntigravityClient::new(key, "antigravity-preview-05-2026", Duration::from_secs(180));
        let (tx, mut rx) = tokio::sync::mpsc::channel(64);
        let req = LlmRequest {
            prompt: "Read Hacker News, summarize the top 10 stories, and save the results as a PDF.".into(),
            model: "antigravity-preview-05-2026".into(),
            messages: vec![],
        };

        client.stream(req, tx).await;

        let mut received = Vec::new();
        while let Some(chunk) = rx.recv().await {
            println!("CHUNK: {:?}", chunk);
            received.push(chunk);
        }
        println!("TOTAL CHUNKS: {}", received.len());
        assert!(!received.is_empty());
    }
}
