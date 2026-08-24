// agent/runner.rs — ReAct agent loop for multi-step execution using Google Agent API.

use crate::agent::tools::{AgentToolContext, ToolRegistry};
use crate::llm::gemini::{GeminiClient, GeminiContent, GeminiTool};
use crate::models::{AgentConfig, AgentEvent, GroundingMetadata, Message};
use serde_json::json;
use std::time::Instant;
use tokio::sync::mpsc;

pub struct AgentRunner {
    client: GeminiClient,
    registry: ToolRegistry,
}

impl AgentRunner {
    pub fn new(client: GeminiClient, registry: ToolRegistry) -> Self {
        Self { client, registry }
    }

    /// Runs a full agent session until completion or max_steps reached.
    pub async fn run(
        &self,
        prompt: &str,
        history: &[Message],
        config: &AgentConfig,
        session_id: &str,
        event_tx: mpsc::Sender<AgentEvent>,
    ) -> Result<String, String> {
        let start_time = Instant::now();
        let ctx = AgentToolContext {
            session_id: session_id.to_string(),
        };

        // 1. Configure Gemini tools
        let mut gemini_tools = Vec::new();
        let mut tool_def = GeminiTool::default();
        let mut has_tools = false;

        if config.enable_google_search {
            tool_def.google_search = Some(json!({}));
            has_tools = true;
        }

        if config.enable_code_execution {
            tool_def.code_execution = Some(json!({}));
            has_tools = true;
        }

        if config.enable_local_tools {
            let decls = self.registry.get_declarations();
            if !decls.is_empty() {
                tool_def.function_declarations = Some(decls);
                has_tools = true;
            }
        }

        if has_tools {
            gemini_tools.push(tool_def);
        }

        let tools_param = if gemini_tools.is_empty() {
            None
        } else {
            Some(gemini_tools)
        };

        // 2. Initialize conversation contents
        let mut contents: Vec<GeminiContent> = Vec::new();

        for msg in history {
            if msg.role == "assistant" || msg.role == "model" {
                contents.push(GeminiContent::model_text(&msg.content));
            } else {
                contents.push(GeminiContent::user_text(&msg.content));
            }
        }

        contents.push(GeminiContent::user_text(prompt));

        let mut accumulated_full_response = String::new();
        let mut collected_grounding = GroundingMetadata::default();
        let mut steps_executed = 0;

        let _ = event_tx
            .send(AgentEvent::Status {
                message: "Starting Agent execution...".into(),
            })
            .await;

        // 3. Multi-step ReAct Loop
        for step in 1..=config.max_steps {
            steps_executed = step;

            let _ = event_tx
                .send(AgentEvent::Status {
                    message: format!("Processing step {}/{}...", step, config.max_steps),
                })
                .await;

            // Channel for streaming tokens of the current turn to the UI
            let (chunk_tx, mut chunk_rx) = mpsc::channel::<String>(32);
            let event_tx_clone = event_tx.clone();

            let chunk_forwarder = tokio::spawn(async move {
                while let Some(text) = chunk_rx.recv().await {
                    let _ = event_tx_clone.send(AgentEvent::Chunk { text }).await;
                }
            });

            let turn_result = self
                .client
                .execute_agent_turn(
                    contents.clone(),
                    config.system_instruction.clone(),
                    tools_param.clone(),
                    Some(chunk_tx),
                )
                .await;

            let _ = chunk_forwarder.await;

            let turn = match turn_result {
                Ok(t) => t,
                Err(err) => {
                    let _ = event_tx
                        .send(AgentEvent::Error {
                            message: err.clone(),
                        })
                        .await;
                    return Err(err);
                }
            };

            // Collect text output
            if !turn.text.is_empty() {
                accumulated_full_response.push_str(&turn.text);
            }

            // Collect grounding metadata if any
            if let Some(gm) = turn.grounding {
                for q in gm.search_queries {
                    if !collected_grounding.search_queries.contains(&q) {
                        collected_grounding.search_queries.push(q);
                    }
                }
                for s in gm.sources {
                    if !collected_grounding.sources.iter().any(|existing| existing.url == s.url) {
                        collected_grounding.sources.push(s);
                    }
                }

                let _ = event_tx
                    .send(AgentEvent::Grounding {
                        queries: collected_grounding.search_queries.clone(),
                        sources: collected_grounding.sources.clone(),
                    })
                    .await;
            }

            // Case A: Model wants to call one or more tools
            if !turn.function_calls.is_empty() {
                if let Some(mc) = turn.model_content {
                    contents.push(mc);
                }

                for fc in turn.function_calls {
                    let _ = event_tx
                        .send(AgentEvent::ToolCall {
                            name: fc.name.clone(),
                            args: fc.args.clone(),
                        })
                        .await;

                    let _ = event_tx
                        .send(AgentEvent::Status {
                            message: format!("Executing tool: {}...", fc.name),
                        })
                        .await;

                    let (tool_res, is_err) = match self.registry.execute(&fc.name, fc.args, &ctx).await {
                        Ok(val) => (val, false),
                        Err(e) => (json!({ "error": e }), true),
                    };

                    let _ = event_tx
                        .send(AgentEvent::ToolResult {
                            name: fc.name.clone(),
                            result: tool_res.clone(),
                            is_error: is_err,
                        })
                        .await;

                    // Append function response back to Gemini conversation
                    contents.push(GeminiContent::function_response(fc.name, tool_res));
                }

                // Loop continues to next turn so Gemini can process tool output
                continue;
            }

            // Case B: No tools called; model finished generating response
            break;
        }

        let duration_ms = start_time.elapsed().as_millis() as i64;

        let _ = event_tx
            .send(AgentEvent::Done {
                duration_ms,
                steps_executed,
                response: accumulated_full_response.clone(),
            })
            .await;

        Ok(accumulated_full_response)
    }
}

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Duration;

    #[tokio::test]
    async fn agent_runner_instantiates() {
        let client = GeminiClient::new("fake-key", "gemini-2.5-flash", Duration::from_secs(10));
        let registry = ToolRegistry::with_default_tools();
        let runner = AgentRunner::new(client, registry);
        assert_eq!(runner.registry.list_names().len(), 3);
    }
}
