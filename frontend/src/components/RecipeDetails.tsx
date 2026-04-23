import { type Recipe } from '../types';

interface Props {
  recipe: Recipe;
}

export default function RecipeDetails({ recipe }: Props) {
  return (
    <div className="space-y-3 text-sm">
      <div className="flex flex-wrap gap-3 text-gray-700">
        {recipe.prep_time && <span>⏱️ Prep: {recipe.prep_time}</span>}
        {recipe.cook_time && <span>🍳 Cook: {recipe.cook_time}</span>}
        {recipe.servings && <span>👥 Serves: {recipe.servings}</span>}
        {recipe.taste && <span>👅 Taste: {recipe.taste}</span>}
        {recipe.cooking_level && <span>📊 Level: {recipe.cooking_level}</span>}
      </div>

      <div>
        <h4 className="font-semibold">Ingredients</h4>
        <ul className="list-disc list-inside ml-2">
          {recipe.ingredients.map((ing, idx) => (
            <li key={idx}>{ing.raw || `${ing.quantity} ${ing.unit} ${ing.name}`}</li>
          ))}
        </ul>
      </div>

      <div>
        <h4 className="font-semibold">Method</h4>
        <ol className="list-decimal list-inside ml-2 space-y-1">
          {recipe.steps.map((step, idx) => (
            <li key={idx}>{step}</li>
          ))}
        </ol>
      </div>

      <div className="pt-2">
        <a
          href={recipe.url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-blue-600 hover:underline inline-flex items-center gap-1"
        >
          View full recipe on Sanjeev Kapoor →
        </a>
      </div>
    </div>
  );
}