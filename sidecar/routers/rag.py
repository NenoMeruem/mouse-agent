"""
RAG (Retrieval-Augmented Generation) API Router.
"""

import json
from fastapi import APIRouter, HTTPException
from sse_starlette.sse import EventSourceResponse

from sidecar.schemas import IngestDocumentRequest, IngestResponse, RagQueryRequest
from sidecar.chains.rag_engine import get_rag_engine

router = APIRouter(prefix="/api/rag", tags=["RAG"])


@router.post("/ingest", response_model=IngestResponse)
async def ingest_documents(req: IngestDocumentRequest):
    """
    Ingests local markdown, code, or PDF documents into ChromaDB.
    """
    engine = get_rag_engine()
    try:
        res = engine.ingest_path(
            path_str=req.directory_or_file,
            collection_name=req.collection_name,
            chunk_size=req.chunk_size,
            chunk_overlap=req.chunk_overlap,
            api_key=req.api_key,
        )
        return IngestResponse(**res)
    except FileNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Ingestion failed: {str(e)}")


@router.post("/query")
async def query_rag_endpoint(req: RagQueryRequest):
    """
    Queries local knowledge base with RAG retrieval and streams LLM synthesis response.
    """
    engine = get_rag_engine()

    async def event_generator():
        try:
            async for event_payload in engine.query_rag_stream(
                query=req.query,
                collection_name=req.collection_name,
                top_k=req.top_k,
                provider=req.provider,
                model=req.model,
                api_key=req.api_key,
            ):
                yield {
                    "event": event_payload.get("event", "message"),
                    "data": json.dumps(event_payload.get("data", {})),
                }
        except Exception as e:
            yield {
                "event": "error",
                "data": json.dumps({"message": f"RAG query failed: {str(e)}"}),
            }

    return EventSourceResponse(event_generator())
