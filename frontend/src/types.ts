export interface Ingredient {
  raw: string;
  quantity: string;
  unit: string;
  name: string;
}

export interface Recipe {
  id: string;
  title: string;
  url: string;
  ingredients: Ingredient[];
  steps: string[];
  cuisine: string;
  course: string;
  dietType: string;
  prep_time: string;
  cook_time: string;
  servings: string;
  taste: string;
  cooking_level: string;
  image_url?: string;
}

export interface MatchResult {
  Recipe: Recipe;
  MatchedCount: number;
  TotalRequired: number;
  Score: number;
}