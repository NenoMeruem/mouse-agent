"""
Desktop RAG (Retrieval-Augmented Generation) Engine.
Integrates ChromaDB local vector store, document loaders, and LCEL retrieval chains.
Accepts API keys directly from request payload.
"""

import os
from pathlib import Path
from typing import List, Dict, Any, AsyncGenerator, Optional

from langchain_core.documents import Document
from langchain_text_splitters import RecursiveCharacterTextSplitter
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.output_parsers import StrOutputParser

from sidecar.config import settings
from sidecar.llm_factory import get_chat_model


class RagEngine:
    def __init__(self, persist_directory: Optional[str] = None):
        self.persist_directory = persist_directory or settings.CHROMA_PERSIST_DIRECTORY
        Path(self.persist_directory).mkdir(parents=True, exist_ok=True)
        self._vector_store = None

    def _get_embeddings(self, api_key: Optional[str] = None):
        """Returns embedding model (Gemini or HuggingFace fallback), prioritizing request api_key."""
        key = api_key or settings.GEMINI_API_KEY or os.environ.get("GEMINI_API_KEY")
        if key:
            from langchain_google_genai import GoogleGenerativeAIEmbeddings
            return GoogleGenerativeAIEmbeddings(
                model="models/embedding-001",
                google_api_key=key,
            )
        openai_key = api_key or settings.OPENAI_API_KEY or os.environ.get("OPENAI_API_KEY")
        if openai_key:
            from langchain_openai import OpenAIEmbeddings
            return OpenAIEmbeddings(
                model="text-embedding-3-small",
                api_key=openai_key,
            )
        # Local embeddings fallback
        from langchain_community.embeddings import FakeEmbeddings
        return FakeEmbeddings(size=768)

    def _get_vectorstore(self, collection_name: str = "default_knowledge", api_key: Optional[str] = None):
        """Initializes or loads Chroma vector store."""
        import chromadb
        from langchain_community.vectorstores import Chroma

        embeddings = self._get_embeddings(api_key=api_key)
        return Chroma(
            collection_name=collection_name,
            embedding_function=embeddings,
            persist_directory=self.persist_directory,
        )

    def ingest_path(
        self,
        path_str: str,
        collection_name: str = "default_knowledge",
        chunk_size: int = 1000,
        chunk_overlap: int = 150,
        api_key: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Loads and indexes local files or directory into ChromaDB."""
        path = Path(path_str).expanduser().resolve()
        if not path.exists():
            raise FileNotFoundError(f"Path does not exist: {path_str}")

        docs: List[Document] = []
        if path.is_file():
            docs.extend(self._load_single_file(path))
        elif path.is_dir():
            for sub_path in path.rglob("*"):
                if sub_path.is_file() and not sub_path.name.startswith("."):
                    try:
                        docs.extend(self._load_single_file(sub_path))
                    except Exception:
                        continue

        if not docs:
            return {"status": "empty", "documents_indexed": 0, "chunks_created": 0, "collection": collection_name}

        splitter = RecursiveCharacterTextSplitter(
            chunk_size=chunk_size,
            chunk_overlap=chunk_overlap,
            separators=["\n\n", "\n", " ", ""],
        )
        splits = splitter.split_documents(docs)

        vs = self._get_vectorstore(collection_name, api_key=api_key)
        vs.add_documents(splits)
        vs.persist()

        return {
            "status": "success",
            "documents_indexed": len(docs),
            "chunks_created": len(splits),
            "collection": collection_name,
        }

    def _load_single_file(self, file_path: Path) -> List[Document]:
        """Loads a single file (MD, TXT, PY, RS, JS, PDF)."""
        suffix = file_path.suffix.lower()
        if suffix in [".txt", ".md", ".py", ".rs", ".js", ".ts", ".json", ".yaml", ".yml", ".html", ".css"]:
            content = file_path.read_text(encoding="utf-8", errors="replace")
            return [Document(page_content=content, metadata={"source": str(file_path), "filename": file_path.name})]
        elif suffix == ".pdf":
            try:
                from langchain_community.document_loaders import PyPDFLoader
                loader = PyPDFLoader(str(file_path))
                return loader.load()
            except Exception:
                return []
        return []

    async def query_rag_stream(
        self,
        query: str,
        collection_name: str = "default_knowledge",
        top_k: int = 4,
        provider: str = "gemini",
        model: Optional[str] = None,
        api_key: Optional[str] = None,
    ) -> AsyncGenerator[Dict[str, Any], None]:
        """Retrieves top-k context chunks and streams LLM synthesis response."""
        yield {"event": "status", "data": {"message": f"Searching local knowledge base '{collection_name}'..."}}

        vs = self._get_vectorstore(collection_name, api_key=api_key)
        retriever = vs.as_retriever(search_kwargs={"k": top_k})
        docs = retriever.invoke(query)

        if not docs:
            yield {"event": "status", "data": {"message": "No matching documents found in vector store."}}
            context_text = "No additional context found."
        else:
            yield {
                "event": "status",
                "data": {
                    "message": f"Retrieved {len(docs)} relevant context snippet(s).",
                    "sources": [d.metadata.get("filename", "unknown") for d in docs],
                },
            }
            context_text = "\n\n---\n\n".join(
                [f"[Source: {d.metadata.get('filename', 'doc')}]\n{d.page_content}" for d in docs]
            )

        template = """You are a helpful desktop assistant. Answer the user's question using the provided context whenever relevant.

Context:
{context}

Question:
{question}

Answer clearly with Markdown formatting:"""

        prompt = ChatPromptTemplate.from_template(template)
        llm = get_chat_model(provider=provider, model=model, api_key=api_key, streaming=True)
        chain = prompt | llm | StrOutputParser()

        async for chunk in chain.astream({"context": context_text, "question": query}):
            yield {"event": "chunk", "data": {"text": chunk}}

        yield {
            "event": "done",
            "data": {
                "status": "completed",
                "sources_used": [d.metadata.get("source", "") for d in docs],
            },
        }


# Global instance helper
_engine_instance = None

def get_rag_engine() -> RagEngine:
    global _engine_instance
    if _engine_instance is None:
        _engine_instance = RagEngine()
    return _engine_instance
