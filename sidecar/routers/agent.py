"""
Autonomous Agent Router using LangGraph with SSE event streaming.
"""

import json
from fastapi import APIRouter
from sse_starlette.sse import EventSourceResponse

from sidecar.schemas import AgentRunRequest
from sidecar.agents.graph_runner import run_agent_graph_stream

router = APIRouter(prefix="/api/agent", tags=["Agent"])


@router.post("/stream")
async def stream_agent_session(req: AgentRunRequest):
    """
    Runs multi-step LangGraph agent session, streaming status, tool calls, results, and tokens via SSE.
    """
    async def event_generator():
        try:
            async for event_payload in run_agent_graph_stream(req):
                yield {
                    "event": event_payload.get("event", "message"),
                    "data": json.dumps(event_payload.get("data", {})),
                }
        except Exception as e:
            yield {
                "event": "error",
                "data": json.dumps({"message": f"Agent execution error: {str(e)}"}),
            }

    return EventSourceResponse(event_generator())
