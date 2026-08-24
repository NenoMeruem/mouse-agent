"""
Routers package init.
"""

from sidecar.routers.health import router as health_router
from sidecar.routers.recipes import router as recipes_router
from sidecar.routers.rag import router as rag_router
from sidecar.routers.agent import router as agent_router

__all__ = ["health_router", "recipes_router", "rag_router", "agent_router"]
