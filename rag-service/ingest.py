import json, glob, chromadb, ollama, os

CHROMA_HOST = os.getenv("CHROMA_HOST", "localhost")
client = chromadb.HttpClient(host=CHROMA_HOST, port=8000)
collection = client.get_or_create_collection("recipes")

recipe_files = glob.glob("/app/output/*.json")
print(f"Found {len(recipe_files)} recipe files")

for path in recipe_files:
    with open(path) as f:
        recipes = json.load(f)
    if not isinstance(recipes, list):
        recipes = [recipes]

    for recipe in recipes:
        doc = f"""Title: {recipe.get('title', '')}
Cuisine: {recipe.get('cuisine', '')}
Diet: {recipe.get('diet_type', '')}
Ingredients: {', '.join(i.get('name', '') for i in recipe.get('ingredients', []))}
Steps: {' '.join(recipe.get('steps', [])[:3])}""".strip()

        emb = ollama.embeddings(model="nomic-embed-text", prompt=doc)
        try:
            collection.add(
                ids=[recipe["id"]],
                documents=[doc],
                embeddings=[emb["embedding"]],
                metadatas=[{"title": recipe.get("title", "")}]
            )
            print(f"Indexed: {recipe.get('title')}")
        except Exception as e:
            print(f"Skipped {recipe.get('id')}: {e}")
