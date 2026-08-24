"""
Desktop Agent Tools: Web search, URL fetching, local file operations, and math computation.
Compatible with LangChain and LangGraph Tool calling conventions.
"""

import subprocess
from pathlib import Path
from typing import List
import httpx
from langchain_core.tools import tool
from sidecar.config import settings


@tool
def fetch_web_url(url: str) -> str:
    """Fetches clean text content from a given HTTP/HTTPS URL."""
    if not (url.startswith("http://") or url.startswith("https://")):
        return "Error: URL must start with http:// or https://"

    try:
        with httpx.Client(timeout=10.0, follow_redirects=True) as client:
            resp = client.get(url, headers={"User-Agent": "PromptlyAgent/1.0"})
            resp.raise_for_status()
            text = resp.text
            # Basic text truncation to prevent token overflow
            if len(text) > 10000:
                return text[:10000] + "\n...[Content truncated]"
            return text
    except Exception as e:
        return f"Failed to fetch URL {url}: {str(e)}"


@tool
def search_web(query: str, max_results: int = 5) -> str:
    """Performs live web search for current information and documentation."""
    # Priority 1: Tavily if API key is configured
    if settings.TAVILY_API_KEY:
        try:
            from langchain_community.tools.tavily_search import TavilySearchResults
            tavily = TavilySearchResults(max_results=max_results, api_key=settings.TAVILY_API_KEY)
            results = tavily.invoke(query)
            return str(results)
        except Exception as e:
            pass  # Fallback to DuckDuckGo

    # Priority 2: DuckDuckGo Search (no API key required)
    try:
        from duckduckgo_search import DDGS
        with DDGS() as ddgs:
            results = list(ddgs.text(query, max_results=max_results))
            if not results:
                return f"No search results found for '{query}'."
            formatted = []
            for r in results:
                formatted.append(f"- **{r.get('title')}**: {r.get('body')}\n  Link: {r.get('href')}")
            return "\n\n".join(formatted)
    except Exception as e:
        return f"Web search error: {str(e)}"


@tool
def read_local_file(file_path: str) -> str:
    """Reads the text content of a local file from the user's filesystem."""
    try:
        path = Path(file_path).expanduser().resolve()
        if not path.exists():
            return f"Error: File '{file_path}' does not exist."
        if path.is_dir():
            return f"Error: Path '{file_path}' is a directory, not a file."
        if path.stat().st_size > 1024 * 1024:
            return f"Error: File '{file_path}' exceeds maximum size of 1MB."
        
        content = path.read_text(encoding="utf-8", errors="replace")
        return content
    except Exception as e:
        return f"Error reading file '{file_path}': {str(e)}"


@tool
def calc_math(expression: str) -> str:
    """Evaluates a basic mathematical expression (addition, subtraction, multiplication, division, powers)."""
    sanitized = "".join(c for c in expression if c.isdigit() or c in "+-*/().^ ")
    if not sanitized.strip():
        return "Error: Invalid math expression."
    try:
        # Safe eval using math operators
        result = eval(sanitized.replace("^", "**"), {"__builtins__": None}, {})
        return f"Result: {result}"
    except Exception as e:
        return f"Calculation error: {str(e)}"


@tool
def run_shell_command(command: str) -> str:
    """Executes a non-interactive shell command and returns stdout / stderr."""
    # Safety blacklist for destructive commands
    forbidden = ["rm -rf /", "mkfs", ":(){ :|:& };:", "dd if="]
    for bad in forbidden:
        if bad in command:
            return f"Error: Command rejected for security reasons: '{command}'"

    try:
        res = subprocess.run(
            command,
            shell=True,
            capture_output=True,
            text=True,
            timeout=15,
        )
        output = res.stdout if res.stdout else res.stderr
        return output.strip() if output else "[Command completed with no output]"
    except subprocess.TimeoutExpired:
        return "Error: Command timed out after 15 seconds."
    except Exception as e:
        return f"Error executing command: {str(e)}"


def get_desktop_tools(enable_web: bool = True, enable_local: bool = True) -> List[tool]:
    """Returns a list of active LangChain tools based on configuration."""
    tools = [calc_math]
    if enable_web:
        tools.extend([fetch_web_url, search_web])
    if enable_local:
        tools.extend([read_local_file, run_shell_command])
    return tools
