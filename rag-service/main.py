from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import chromadb, ollama, os

app = FastAPI()

CHROMA_HOST = os.getenv("CHROMA_HOST", "localhost")
OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://localhost:11434")

chroma = chromadb.HttpClient(host=CHROMA_HOST, port=8000)
collection = chroma.get_or_create_collection("recipes")

class TipsRequest(BaseModel):
    recipe_title: str
    ingredients: list[str] = []
    question: str = "Give me cooking tips for this recipe."

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/rag/tips")
def get_tips(req: TipsRequest):
    query = f"{req.recipe_title}: {', '.join(req.ingredients)}"

    # 1. Embed the query
    emb = ollama.embeddings(
        model="nomic-embed-text",
        prompt=query,
        options={"base_url": OLLAMA_HOST}
    )

    # 2. Retrieve 3 closest recipes from ChromaDB
    results = collection.query(
        query_embeddings=[emb["embedding"]],
        n_results=3
    )
    context = "\n\n---\n\n".join(results["documents"][0])

    # 3. Generate answer with local Ollama LLM
    prompt = f"""You are a helpful culinary assistant. Use only the recipe context below to answer.

Context:
{context}

Question: {req.question} for "{req.recipe_title}"

Give practical, specific tips. Be concise."""

    response = ollama.chat(
        model="llama3.2:3b-instruct-q4_K_M",
        messages=[{"role": "user", "content": prompt}],
        options={"base_url": OLLAMA_HOST}
    )
    return {"tips": response["message"]["content"]}
