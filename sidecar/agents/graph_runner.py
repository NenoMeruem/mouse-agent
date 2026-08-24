"""
LangGraph / LangChain Agent Runner.
Executes multi-step ReAct agent loops with real-time tool calling and streaming SSE events.
Directly uses the API key provided in the request payload.
"""

from typing import AsyncGenerator, Dict, Any, List
from langchain_core.messages import HumanMessage, AIMessage, SystemMessage
from langgraph.prebuilt import create_react_agent

from sidecar.llm_factory import get_chat_model
from sidecar.tools.desktop_tools import get_desktop_tools
from sidecar.schemas import AgentRunRequest


async def run_agent_graph_stream(
    request: AgentRunRequest,
) -> AsyncGenerator[Dict[str, Any], None]:
    """
    Runs the agent loop with LangGraph, yielding fine-grained events for UI streaming.
    """
    yield {"event": "status", "data": {"message": "Initializing autonomous agent..."}}

    # 1. Prepare Tools & Model with Request API Key
    tools = get_desktop_tools(
        enable_web=request.enable_web_search,
        enable_local=request.enable_local_tools,
    )
    llm = get_chat_model(
        provider=request.provider,
        model=request.model,
        api_key=request.api_key,
        streaming=True,
    )

    system_instruction = (
        "You are Promptly, an autonomous AI desktop co-pilot. "
        "You have access to tools for web search, reading local files, math calculation, and shell commands. "
        "Think step-by-step. Use tools when needed to gather facts before providing your final answer. "
        "Format responses cleanly in Markdown."
    )

    # 2. Build Agent Graph
    agent_executor = create_react_agent(
        model=llm,
        tools=tools,
        prompt=system_instruction,
    )

    # 3. Format Message History
    messages = []
    for msg in request.history:
        if msg.role in ["assistant", "model"]:
            messages.append(AIMessage(content=msg.content))
        elif msg.role == "system":
            messages.append(SystemMessage(content=msg.content))
        else:
            messages.append(HumanMessage(content=msg.content))

    messages.append(HumanMessage(content=request.prompt))

    # 4. Stream Agent Execution
    yield {"event": "status", "data": {"message": "Agent thinking & planning..."}}

    try:
        async for chunk in agent_executor.astream(
            {"messages": messages},
            stream_mode="updates",
        ):
            # Inspect node outputs from LangGraph state transitions
            for node_name, node_update in chunk.items():
                if "messages" in node_update:
                    for m in node_update["messages"]:
                        # Case A: Model issued tool calls
                        if hasattr(m, "tool_calls") and m.tool_calls:
                            for tc in m.tool_calls:
                                yield {
                                    "event": "tool_call",
                                    "data": {
                                        "name": tc.get("name"),
                                        "args": tc.get("args"),
                                        "id": tc.get("id"),
                                    },
                                }
                                yield {
                                    "event": "status",
                                    "data": {"message": f"Executing tool: {tc.get('name')}..."},
                                }

                        # Case B: Tool execution completed
                        elif getattr(m, "type", "") == "tool" or m.__class__.__name__ == "ToolMessage":
                            yield {
                                "event": "tool_result",
                                "data": {
                                    "name": getattr(m, "name", "tool"),
                                    "result": m.content,
                                },
                            }

                        # Case C: Final response message from agent
                        elif isinstance(m, AIMessage) and m.content:
                            content_str = str(m.content)
                            yield {"event": "chunk", "data": {"text": content_str}}

        yield {"event": "done", "data": {"status": "completed"}}

    except Exception as e:
        yield {"event": "error", "data": {"message": f"Agent error: {str(e)}"}}
