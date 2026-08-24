"""
Pydantic schemas for API request, response, and SSE streaming payloads.
Directly accepts engine/provider, model, and api_key from caller (Tauri / Rust).
"""

from typing import Any, Dict, List, Optional
from pydantic import BaseModel, Field


class Message(BaseModel):
    role: str = Field(description="Role: user, assistant, system, tool")
    content: str = Field(description="Text content of the message")


# ── Recipe & Chain Schemas ───────────────────────────────────────────────────

class RecipeStep(BaseModel):
    step_id: str = Field(description="Unique identifier of this step")
    template: str = Field(description="Prompt template containing {{variables}}")
    provider: Optional[str] = Field(default="gemini", description="gemini, openai, claude, ollama")
    model: Optional[str] = Field(default=None, description="Model override for this step (e.g. gpt-4o-mini, gemini-2.5-flash)")
    api_key: Optional[str] = Field(default=None, description="API key for this specific step if different")
    output_key: str = Field(default="result", description="Key where output will be stored for subsequent steps")


class RecipeChainRequest(BaseModel):
    recipe_id: str
    selection: str = Field(default="", description="Highlighted text or clipboard content")
    provider: Optional[str] = Field(default="gemini", description="LLM Engine/Provider: gemini, openai, claude, ollama")
    model: Optional[str] = Field(default=None, description="Specific Model Name: gemini-2.5-flash-lite, gpt-4o-mini, claude-3-5-sonnet, llama3.2")
    api_key: Optional[str] = Field(default=None, description="LLM API key passed directly from client/Rust")
    parameters: Dict[str, Any] = Field(default_factory=dict, description="Style, Length, Tone, etc.")
    steps: Optional[List[RecipeStep]] = Field(default=None, description="Custom multi-step pipeline definition")
    session_id: Optional[str] = None


# ── RAG Schemas ─────────────────────────────────────────────────────────────

class IngestDocumentRequest(BaseModel):
    directory_or_file: str = Field(description="Local path to directory, markdown, or PDF to index")
    collection_name: str = Field(default="default_knowledge", description="Collection / namespace name")
    chunk_size: int = Field(default=1000, description="Character chunk size")
    chunk_overlap: int = Field(default=150, description="Chunk overlap")
    api_key: Optional[str] = Field(default=None, description="API key for embedding model")


class IngestResponse(BaseModel):
    status: str
    documents_indexed: int
    chunks_created: int
    collection: str


class RagQueryRequest(BaseModel):
    query: str = Field(description="User search query or question")
    collection_name: str = Field(default="default_knowledge", description="Vector collection to search")
    top_k: int = Field(default=4, description="Number of context chunks to retrieve")
    provider: str = Field(default="gemini", description="LLM Engine: gemini, openai, claude, ollama")
    model: Optional[str] = Field(default=None, description="Model identifier: gpt-4o, gemini-2.5-flash, etc.")
    api_key: Optional[str] = Field(default=None, description="LLM API key passed directly from client/Rust")


# ── Agent Schemas ────────────────────────────────────────────────────────────

class AgentRunRequest(BaseModel):
    prompt: str = Field(description="Initial instruction or question for the agent")
    history: List[Message] = Field(default_factory=list, description="Previous conversation turns")
    session_id: Optional[str] = None
    provider: str = Field(default="gemini", description="LLM Engine: gemini, openai, claude, ollama")
    model: Optional[str] = Field(default=None, description="Model identifier: gpt-4o, gemini-2.5-flash, etc.")
    api_key: Optional[str] = Field(default=None, description="LLM API key passed directly from client/Rust")
    enable_web_search: bool = True
    enable_local_tools: bool = True
    max_iterations: int = Field(default=6, description="Max ReAct execution steps")


# ── Stream Event Payloads ────────────────────────────────────────────────────

class StreamEvent(BaseModel):
    event: str = Field(description="status, chunk, tool_call, tool_result, error, done")
    data: Dict[str, Any] = Field(default_factory=dict)
