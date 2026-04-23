import type { MatchResult } from '../types';

interface Props {
  match: MatchResult;
  onClick: () => void;
}

export default function RecipeCard({ match, onClick }: Props) {
  const { Recipe: recipe, MatchedCount, TotalRequired, Score } = match;
  return (
    <div onClick={onClick} className="border rounded-xl shadow-sm hover:shadow-md transition p-4 cursor-pointer bg-white">
      <h3 className="text-xl font-bold mb-1">{recipe.title}</h3>
      <p className="text-sm text-gray-600">
        Match: {MatchedCount}/{TotalRequired} ingredients ({Math.round(Score * 100)}%)
      </p>
      <p className="text-xs text-gray-500 mt-1">
        {recipe.cuisine} | {recipe.course} | {recipe.dietType}
      </p>
    </div>
  );
}