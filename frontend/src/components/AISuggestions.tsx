import { useState } from 'react';
import axios from 'axios';

interface Props {
  recipeTitle: string;
  ingredients: string[];
}

export default function AISuggestions({ recipeTitle, ingredients }: Props) {
  const [suggestion, setSuggestion] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const askAI = async () => {
    setLoading(true);
    try {
      const prompt = `Give me 3 quick cooking tips for "${recipeTitle}". Main ingredients: ${ingredients.join(', ')}. Keep concise, one tip per line.`;
      const payload = {
        contents: [{ parts: [{ text: prompt }] }]
      };
      const response = await axios.post('/api/ai/tips', payload);
      const text = response.data.candidates?.[0]?.content?.parts?.[0]?.text;
      setSuggestion(text || "No tips generated.");
    } catch (err) {
      console.error(err);
      setSuggestion("Failed to fetch AI tips. Make sure backend is running and API key is set.");
    } finally {
      setLoading(false);
    }
  };

  if (suggestion === null && !loading) {
    return (
      <button
        onClick={askAI}
        className="mt-3 text-sm bg-purple-100 text-purple-700 px-3 py-1 rounded-md hover:bg-purple-200"
      >
        ✨ Ask AI for cooking tips
      </button>
    );
  }

  return (
    <div className="mt-3 p-3 bg-gray-50 rounded-md text-sm">
      {loading ? (
        <div className="flex items-center gap-2">
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-purple-700"></div>
          <span>AI is thinking...</span>
        </div>
      ) : (
        <div className="whitespace-pre-wrap">{suggestion}</div>
      )}
    </div>
  );
}