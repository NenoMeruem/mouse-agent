// llm/mod.rs — LLM Client trait, Manager, and streaming types.
// Mirrors Go's internal/llm/types.go + internal/llm/manager.go.

pub mod claude;
pub mod gemini;
pub mod openai;
pub mod sidecar;


use crate::models::{Chunk, LlmRequest};
use std::collections::HashMap;
use tokio::sync::mpsc;

// ---------------------------------------------------------------------------
// Streaming sender alias
// ---------------------------------------------------------------------------

pub type ChunkSender = mpsc::Sender<Chunk>;

// ---------------------------------------------------------------------------
// LlmClient trait
// ---------------------------------------------------------------------------

/// All LLM providers implement this trait.
/// Mirrors Go's `llm.Client` interface.
#[async_trait::async_trait]
pub trait LlmClient: Send + Sync {
    /// Provider name (e.g. `"gemini"`, `"openai"`, `"claude"`).
    fn name(&self) -> &str;

    /// Streams response chunks into `tx`. Closes the channel when done or on error.
    async fn stream(&self, req: LlmRequest, tx: ChunkSender);
}

// ---------------------------------------------------------------------------
// Manager
// ---------------------------------------------------------------------------

/// Registry for LLM provider clients.
/// Mirrors Go's `llm.Manager`.
pub struct Manager {
    clients: HashMap<String, Box<dyn LlmClient>>,
}

impl Manager {
    pub fn new() -> Self {
        Self { clients: HashMap::new() }
    }

    /// Registers a client under `client.name()` as the key.
    /// Silently overwrites if the name already exists.
    pub fn register(&mut self, client: Box<dyn LlmClient>) {
        self.clients.insert(client.name().to_string(), client);
    }

    /// Registers a client under an explicit `id`, independent of `client.name()`.
    /// Use this to register multiple instances of the same provider with different IDs.
    pub fn register_id(&mut self, id: String, client: Box<dyn LlmClient>) {
        self.clients.insert(id, client);
    }

    /// Returns a reference to the client for `engine`, or `None` if not registered.
    pub fn get(&self, engine: &str) -> Option<&dyn LlmClient> {
        self.clients.get(engine).map(|c| c.as_ref())
    }

    /// Returns all registered engine names.
    pub fn list(&self) -> Vec<&str> {
        self.clients.keys().map(|s| s.as_str()).collect()
    }
}
