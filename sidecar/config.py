"""
Configuration management using pydantic-settings.
Reads configuration from environment variables or .env file.
"""

from pathlib import Path
from pydantic_settings import BaseSettings, SettingsConfigDict
from typing import Optional


class Settings(BaseSettings):
    # Server Settings
    PORT: int = 8000
    HOST: str = "127.0.0.1"
    DEBUG: bool = True

    # API Keys
    GEMINI_API_KEY: Optional[str] = None
    OPENAI_API_KEY: Optional[str] = None
    ANTHROPIC_API_KEY: Optional[str] = None

    # Local Ollama
    OLLAMA_BASE_URL: str = "http://localhost:11434"

    # Search & Tool Keys
    TAVILY_API_KEY: Optional[str] = None

    # RAG Vector Store Path
    CHROMA_PERSIST_DIRECTORY: str = str(Path.home() / ".promptly" / "chroma_db")

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )


settings = Settings()
