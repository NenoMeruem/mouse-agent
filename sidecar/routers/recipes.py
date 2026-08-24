"""
Recipe execution router with Server-Sent Events (SSE) token streaming.
"""

import json
from fastapi import APIRouter
from sse_starlette.sse import EventSourceResponse

from sidecar.schemas import RecipeChainRequest
from sidecar.chains.recipe_pipeline import execute_recipe_pipeline

router = APIRouter(prefix="/api/recipes", tags=["Recipes"])


@router.post("/run")
async def run_recipe_endpoint(request: RecipeChainRequest):
    """
    Executes a single or chained recipe with real-time SSE streaming.
    """
    async def event_generator():
        try:
            async for event_payload in execute_recipe_pipeline(request):
                yield {
                    "event": event_payload.get("event", "message"),
                    "data": json.dumps(event_payload.get("data", {})),
                }
        except Exception as e:
            yield {
                "event": "error",
                "data": json.dumps({"message": f"Pipeline execution error: {str(e)}"}),
            }

    return EventSourceResponse(event_generator())
