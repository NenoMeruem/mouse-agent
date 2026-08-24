"""
Health & System Diagnostic router.
"""

from fastapi import APIRouter
from sidecar.config import settings

router = APIRouter(prefix="/api/health", tags=["Health"])


@router.get("")
async def health_check():
    """Returns status and configured LLM providers."""
    return {
        "status": "healthy",
        "service": "promptly-langchain-sidecar",
        "version": "0.2.0",
        "providers": {
            "gemini": bool(settings.GEMINI_API_KEY),
            "openai": bool(settings.OPENAI_API_KEY),
            "claude": bool(settings.ANTHROPIC_API_KEY),
            "ollama": bool(settings.OLLAMA_BASE_URL),
        },
        "tavily_search": bool(settings.TAVILY_API_KEY),
    }
