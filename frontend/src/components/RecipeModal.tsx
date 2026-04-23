import { useState } from 'react';
import type { Recipe } from '../types';
import AISuggestions from './AISuggestions';

interface Props {
  recipe: Recipe;
  matchedCount: number;
  totalRequired: number;
  score: number;
  onClose: () => void;
}

export default function RecipeModal({ recipe, matchedCount, totalRequired, score, onClose }: Props) {
  const [showAI, setShowAI] = useState(false);

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div className="bg-white rounded-xl max-w-2xl w-full max-h-[85vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <div className="sticky top-0 bg-white border-b p-4 flex justify-between items-center">
          <h2 className="text-2xl font-bold">{recipe.title}</h2>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-700 text-2xl">&times;</button>
        </div>
        <div className="p-6 space-y-4">
          <div className="flex flex-wrap gap-3 text-sm text-gray-600">
            {recipe.prep_time && <span>⏱️ Prep: {recipe.prep_time}</span>}
            {recipe.cook_time && <span>🍳 Cook: {recipe.cook_time}</span>}
            {recipe.servings && <span>👥 Serves: {recipe.servings}</span>}
            {recipe.taste && <span>👅 Taste: {recipe.taste}</span>}
            {recipe.cooking_level && <span>📊 Level: {recipe.cooking_level}</span>}
          </div>
          <div>
            <p className="text-sm text-gray-500 mb-1">Match: {matchedCount}/{totalRequired} ingredients ({Math.round(score * 100)}%)</p>
            <p className="text-xs text-gray-400">{recipe.cuisine} | {recipe.course} | {recipe.dietType}</p>
          </div>
          <div>
            <h3 className="font-semibold text-lg">Ingredients</h3>
            <ul className="list-disc list-inside mt-2 space-y-1">
              {recipe.ingredients.map((ing, i) => (
                <li key={i}>{ing.raw || `${ing.quantity} ${ing.unit} ${ing.name}`}</li>
              ))}
            </ul>
          </div>
          <div>
            <h3 className="font-semibold text-lg">Method</h3>
            <ol className="list-decimal list-inside mt-2 space-y-2">
              {recipe.steps.map((step, i) => (
                <li key={i}>{step}</li>
              ))}
            </ol>
          </div>
          <div>
            <a href={recipe.url} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">View full recipe on original site →</a>
          </div>
          <div className="border-t pt-4">
            <button
              onClick={() => setShowAI(!showAI)}
              className="bg-purple-600 text-white px-4 py-2 rounded-md hover:bg-purple-700"
            >
              {showAI ? 'Hide AI Tips' : '✨ Get AI Cooking Tips'}
            </button>
            {showAI && <AISuggestions recipeTitle={recipe.title} ingredients={recipe.ingredients.map(i => i.name)} />}
          </div>
        </div>
      </div>
    </div>
  );
}