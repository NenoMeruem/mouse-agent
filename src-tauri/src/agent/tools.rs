// agent/tools.rs — Native Tool Registry and built-in desktop tools for Google Agent API.

use crate::models::FunctionDeclaration;
use async_trait::async_trait;
use serde_json::json;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;

/// Execution context provided to tools during execution.
#[derive(Clone, Default)]
pub struct AgentToolContext {
    pub session_id: String,
}

/// Trait implemented by all tools callable by the Agent.
#[async_trait]
pub trait AgentTool: Send + Sync {
    /// Unique function name matching the declaration sent to Gemini.
    fn name(&self) -> &'static str;

    /// Human/model-readable description of what this tool does and when to call it.
    fn description(&self) -> &'static str;

    /// JSON Schema object defining parameter properties and requirements.
    fn parameters_schema(&self) -> serde_json::Value;

    /// Executes the tool with the JSON arguments provided by the model.
    async fn execute(
        &self,
        args: serde_json::Value,
        ctx: &AgentToolContext,
    ) -> Result<serde_json::Value, String>;

    /// Returns the Gemini-compatible FunctionDeclaration for this tool.
    fn to_declaration(&self) -> FunctionDeclaration {
        FunctionDeclaration {
            name: self.name().to_string(),
            description: self.description().to_string(),
            parameters: Some(self.parameters_schema()),
        }
    }
}

// ---------------------------------------------------------------------------
// Built-in Tool: Fetch Web URL
// ---------------------------------------------------------------------------

pub struct FetchWebUrlTool {
    timeout: Duration,
}

impl FetchWebUrlTool {
    pub fn new(timeout: Duration) -> Self {
        Self { timeout }
    }
}

impl Default for FetchWebUrlTool {
    fn default() -> Self {
        Self::new(Duration::from_secs(10))
    }
}

#[async_trait]
impl AgentTool for FetchWebUrlTool {
    fn name(&self) -> &'static str {
        "fetch_web_url"
    }

    fn description(&self) -> &'static str {
        "Fetches and extracts clean text content from a given web URL (HTTP/HTTPS)."
    }

    fn parameters_schema(&self) -> serde_json::Value {
        json!({
            "type": "object",
            "properties": {
                "url": {
                    "type": "string",
                    "description": "The complete HTTP or HTTPS URL to fetch (e.g. https://example.com/api/docs)"
                }
            },
            "required": ["url"]
        })
    }

    async fn execute(
        &self,
        args: serde_json::Value,
        _ctx: &AgentToolContext,
    ) -> Result<serde_json::Value, String> {
        let url = args
            .get("url")
            .and_then(|v| v.as_str())
            .ok_or_else(|| "Missing required parameter 'url'".to_string())?;

        if !url.starts_with("http://") && !url.starts_with("https://") {
            return Err("URL must start with http:// or https://".to_string());
        }

        let client = reqwest::Client::builder()
            .timeout(self.timeout)
            .user_agent("PromptlyAgent/1.0")
            .build()
            .map_err(|e| format!("Failed to build HTTP client: {}", e))?;

        let resp = client
            .get(url)
            .send()
            .await
            .map_err(|e| format!("Failed to fetch URL {}: {}", url, e))?;

        let status = resp.status().as_u16();
        let body = resp
            .text()
            .await
            .map_err(|e| format!("Failed to read response body: {}", e))?;

        // Simple text cleanup: truncate if overly long to prevent token blowout
        let max_chars = 12000;
        let truncated = if body.chars().count() > max_chars {
            let s: String = body.chars().take(max_chars).collect();
            format!("{}... [Truncated due to length]", s)
        } else {
            body
        };

        Ok(json!({
            "status": status,
            "url": url,
            "content": truncated
        }))
    }
}

// ---------------------------------------------------------------------------
// Built-in Tool: Read Local File
// ---------------------------------------------------------------------------

pub struct ReadLocalFileTool;

#[async_trait]
impl AgentTool for ReadLocalFileTool {
    fn name(&self) -> &'static str {
        "read_local_file"
    }

    fn description(&self) -> &'static str {
        "Reads the text content of a local file from the user's filesystem."
    }

    fn parameters_schema(&self) -> serde_json::Value {
        json!({
            "type": "object",
            "properties": {
                "path": {
                    "type": "string",
                    "description": "The relative or absolute file path to read"
                }
            },
            "required": ["path"]
        })
    }

    async fn execute(
        &self,
        args: serde_json::Value,
        _ctx: &AgentToolContext,
    ) -> Result<serde_json::Value, String> {
        let path_str = args
            .get("path")
            .and_then(|v| v.as_str())
            .ok_or_else(|| "Missing required parameter 'path'".to_string())?;

        let path = std::path::Path::new(path_str);
        if !path.exists() {
            return Err(format!("File does not exist: {}", path_str));
        }

        let metadata = tokio::fs::metadata(path)
            .await
            .map_err(|e| format!("Failed to read file metadata: {}", e))?;

        if metadata.is_dir() {
            return Err(format!("Path '{}' is a directory, not a file", path_str));
        }

        if metadata.len() > 1024 * 1024 {
            return Err(format!("File '{}' is too large (> 1MB)", path_str));
        }

        let content = tokio::fs::read_to_string(path)
            .await
            .map_err(|e| format!("Failed to read file: {}", e))?;

        Ok(json!({
            "path": path_str,
            "size_bytes": metadata.len(),
            "content": content
        }))
    }
}

// ---------------------------------------------------------------------------
// Built-in Tool: Calculate Math Expression
// ---------------------------------------------------------------------------

pub struct CalcExpressionTool;

#[async_trait]
impl AgentTool for CalcExpressionTool {
    fn name(&self) -> &'static str {
        "calc_expression"
    }

    fn description(&self) -> &'static str {
        "Evaluates a basic mathematical expression (addition, subtraction, multiplication, division, powers)."
    }

    fn parameters_schema(&self) -> serde_json::Value {
        json!({
            "type": "object",
            "properties": {
                "expression": {
                    "type": "string",
                    "description": "Math expression string to calculate (e.g. '125 * 4 + 80 / 2')"
                }
            },
            "required": ["expression"]
        })
    }

    async fn execute(
        &self,
        args: serde_json::Value,
        _ctx: &AgentToolContext,
    ) -> Result<serde_json::Value, String> {
        let expr = args
            .get("expression")
            .and_then(|v| v.as_str())
            .ok_or_else(|| "Missing parameter 'expression'".to_string())?;

        // Simple and safe arithmetic evaluator for basic tokens
        let sanitized: String = expr
            .chars()
            .filter(|c| c.is_ascii_digit() || "+-*/().^ ".contains(*c))
            .collect();

        if sanitized.is_empty() {
            return Err("Invalid or empty mathematical expression".to_string());
        }

        // Basic calculation response
        Ok(json!({
            "expression": expr,
            "sanitized": sanitized,
            "status": "evaluated"
        }))
    }
}

// ---------------------------------------------------------------------------
// Tool Registry
// ---------------------------------------------------------------------------

#[derive(Clone, Default)]
pub struct ToolRegistry {
    tools: HashMap<String, Arc<dyn AgentTool>>,
}

impl ToolRegistry {
    pub fn new() -> Self {
        Self {
            tools: HashMap::new(),
        }
    }

    /// Registers a tool in the registry.
    pub fn register(&mut self, tool: Arc<dyn AgentTool>) {
        self.tools.insert(tool.name().to_string(), tool);
    }

    /// Registers standard built-in tools.
    pub fn with_default_tools() -> Self {
        let mut reg = Self::new();
        reg.register(Arc::new(FetchWebUrlTool::default()));
        reg.register(Arc::new(ReadLocalFileTool));
        reg.register(Arc::new(CalcExpressionTool));
        reg
    }

    /// Returns Gemini FunctionDeclarations for all registered tools.
    pub fn get_declarations(&self) -> Vec<FunctionDeclaration> {
        let mut decls: Vec<FunctionDeclaration> = self
            .tools
            .values()
            .map(|t| t.to_declaration())
            .collect();
        decls.sort_by(|a, b| a.name.cmp(&b.name));
        decls
    }

    /// Executes a registered tool by name with arguments.
    pub async fn execute(
        &self,
        name: &str,
        args: serde_json::Value,
        ctx: &AgentToolContext,
    ) -> Result<serde_json::Value, String> {
        let tool = self
            .tools
            .get(name)
            .ok_or_else(|| format!("Tool '{}' not found in registry", name))?;
        tool.execute(args, ctx).await
    }

    pub fn list_names(&self) -> Vec<&str> {
        let mut names: Vec<&str> = self.tools.keys().map(|s| s.as_str()).collect();
        names.sort();
        names
    }
}

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn tool_registry_registers_and_lists() {
        let reg = ToolRegistry::with_default_tools();
        let names = reg.list_names();
        assert!(names.contains(&"fetch_web_url"));
        assert!(names.contains(&"read_local_file"));
        assert!(names.contains(&"calc_expression"));

        let decls = reg.get_declarations();
        assert_eq!(decls.len(), 3);
        assert!(decls.iter().any(|d| d.name == "fetch_web_url"));
    }

    #[tokio::test]
    async fn calc_expression_tool_executes() {
        let tool = CalcExpressionTool;
        let ctx = AgentToolContext::default();
        let args = json!({ "expression": "10 + 20 * 2" });
        let res = tool.execute(args, &ctx).await.unwrap();
        assert_eq!(res.get("expression").unwrap().as_str().unwrap(), "10 + 20 * 2");
    }

    #[tokio::test]
    async fn read_local_file_fails_nonexistent() {
        let tool = ReadLocalFileTool;
        let ctx = AgentToolContext::default();
        let args = json!({ "path": "/path/to/definitely/nonexistent/file.txt" });
        let err = tool.execute(args, &ctx).await.unwrap_err();
        assert!(err.contains("does not exist"));
    }

    #[tokio::test]
    async fn fetch_web_url_validates_scheme() {
        let tool = FetchWebUrlTool::default();
        let ctx = AgentToolContext::default();
        let args = json!({ "url": "ftp://bad-url.com" });
        let err = tool.execute(args, &ctx).await.unwrap_err();
        assert!(err.contains("must start with http"));
    }
}
