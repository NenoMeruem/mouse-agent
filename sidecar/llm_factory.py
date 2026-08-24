"""
LLM Factory: Creates LangChain BaseChatModel instances across multiple providers
(Google Gemini, OpenAI, Anthropic Claude, Ollama) with direct request API key injection.
"""

import logging
import os
import warnings
from typing import Optional, List
from langchain_core.language_models.chat_models import BaseChatModel
from sidecar.config import settings

# Suppress known informational warnings from third-party SDKs
warnings.filterwarnings("ignore", category=UserWarning, module="langchain_google_genai")
warnings.filterwarnings("ignore", message=".*fixed sampling defaults.*")
warnings.filterwarnings("ignore", message=".*automatic function calling.*")
warnings.filterwarnings("ignore", message=".*Automatic Function Calling.*")
logging.getLogger("google.genai").setLevel(logging.ERROR)
logging.getLogger("google_genai").setLevel(logging.ERROR)

try:
    from google.genai.models import AsyncModels, Models
    AsyncModels._logged_afc_warning = True
    Models._logged_afc_warning = True
except Exception:
    pass



def get_chat_model(
    provider: str = "gemini",
    model: Optional[str] = None,
    api_key: Optional[str] = None,
    temperature: Optional[float] = None,
    streaming: bool = True,
) -> BaseChatModel:
    """
    Instantiates and returns a configured LangChain ChatModel based on provider name.
    Prioritizes api_key passed directly from the request.
    """
    provider = (provider or "gemini").lower()

    if provider == "gemini":
        from langchain_google_genai import ChatGoogleGenerativeAI
        key = api_key or settings.GEMINI_API_KEY or os.environ.get("GEMINI_API_KEY")
        if not key:
            raise ValueError("Missing Gemini API key. Please pass 'api_key' in the request or configure GEMINI_API_KEY.")
        model_name = model or "gemini-2.5-flash-lite"
        
        # Only pass temperature if explicitly provided to avoid warnings on fixed-sampling models
        kwargs = {
            "model": model_name,
            "google_api_key": key,
            "streaming": streaming,
        }
        if temperature is not None:
            kwargs["temperature"] = temperature

        return ChatGoogleGenerativeAI(**kwargs)

    elif provider == "openai":
        from langchain_openai import ChatOpenAI
        key = api_key or settings.OPENAI_API_KEY or os.environ.get("OPENAI_API_KEY")
        if not key:
            raise ValueError("Missing OpenAI API key. Please pass 'api_key' in the request or configure OPENAI_API_KEY.")
        model_name = model or "gpt-4o-mini"
        temp = temperature if temperature is not None else 0.7
        return ChatOpenAI(
            model=model_name,
            api_key=key,
            temperature=temp,
            streaming=streaming,
        )

    elif provider == "claude" or provider == "anthropic":
        from langchain_anthropic import ChatAnthropic
        key = api_key or settings.ANTHROPIC_API_KEY or os.environ.get("ANTHROPIC_API_KEY")
        if not key:
            raise ValueError("Missing Anthropic API key. Please pass 'api_key' in the request or configure ANTHROPIC_API_KEY.")
        model_name = model or "claude-3-5-sonnet-latest"
        temp = temperature if temperature is not None else 0.7
        return ChatAnthropic(
            model=model_name,
            api_key=key,
            temperature=temp,
            streaming=streaming,
        )

    elif provider == "ollama":
        from langchain_community.chat_models import ChatOllama
        model_name = model or "llama3.2"
        temp = temperature if temperature is not None else 0.7
        return ChatOllama(
            base_url=settings.OLLAMA_BASE_URL,
            model=model_name,
            temperature=temp,
        )

    else:
        raise ValueError(f"Unsupported LLM provider: '{provider}'. Supported: gemini, openai, claude, ollama")


def get_fallback_model(
    primary_provider: str = "gemini",
    primary_api_key: Optional[str] = None,
    fallback_providers: Optional[List[str]] = None,
) -> BaseChatModel:
    """
    Returns a primary ChatModel with configured LangChain fallbacks for high availability.
    """
    fallbacks = fallback_providers or ["openai", "claude"]
    primary = get_chat_model(provider=primary_provider, api_key=primary_api_key)

    fallback_models = []
    for p in fallbacks:
        if p != primary_provider:
            try:
                fallback_models.append(get_chat_model(provider=p))
            except Exception:
                pass

    if fallback_models:
        return primary.with_fallbacks(fallback_models)
    return primary
