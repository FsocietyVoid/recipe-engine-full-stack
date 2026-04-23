import { GoogleGenAI } from "@google/genai";

const ai = new GoogleGenAI({ apiKey: import.meta.env.VITE_GEMINI_API_KEY });

export async function getCookingTips(recipeTitle: string, ingredients: string[]) {
  const prompt = `Give me 3 quick cooking tips for "${recipeTitle}". Main ingredients: ${ingredients.join(', ')}. Keep it concise, one tip per line.`;
  
  try {
    const response = await ai.models.generateContent({
      model: 'gemini-2.0-flash-exp',
      contents: prompt,
    });
    return response.text;
  } catch (error) {
    console.error('Gemini API error:', error);
    return "Could not fetch tips at this time.";
  }
}