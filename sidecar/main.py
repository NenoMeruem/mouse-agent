"""
Promptly LangChain Sidecar API
Main entrypoint for FastAPI application.
"""

import logging
import warnings
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from sidecar.config import settings
from sidecar.routers import health_router, recipes_router, rag_router, agent_router

# Suppress non-critical third-party SDK warnings & loggers
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




@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Ensure data dirs and initial setup
    print(f"🚀 Promptly LangChain Sidecar starting on {settings.HOST}:{settings.PORT}")
    yield
    # Shutdown: Clean up any background threads or vector DB connections
    print("🛑 Promptly LangChain Sidecar stopped.")


app = FastAPI(
    title="Promptly AI Engine (LangChain Sidecar)",
    description="FastAPI microservice providing LangChain LCEL, LangGraph Agent loops, and Local RAG for Promptly Desktop.",
    version="0.2.0",
    lifespan=lifespan,
)

# Configure CORS to allow local Tauri frontend IPC
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Mount API Routers
app.include_router(health_router)
app.include_router(recipes_router)
app.include_router(rag_router)
app.include_router(agent_router)


@app.get("/")
async def root():
    return {
        "app": "Promptly LangChain Sidecar",
        "status": "online",
        "docs": "/docs",
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "sidecar.main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
    )
