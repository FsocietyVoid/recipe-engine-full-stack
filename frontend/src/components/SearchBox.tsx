import React, { useState } from 'react';

interface SearchBoxProps {
  onSearch: (ingredients: string) => void;
  loading: boolean;
}

export default function SearchBox({ onSearch, loading }: SearchBoxProps) {
  const [value, setValue] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSearch(value);
  };

  return (
    <form onSubmit={handleSubmit} className="max-w-2xl mx-auto">
      <label className="block text-lg font-medium mb-2">
        What ingredients do you have?
      </label>
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="e.g. chicken, garlic, yogurt, lemon"
        className="w-full p-3 border border-gray-300 rounded-md focus:ring-2 focus:ring-green-500 focus:border-green-500"
      />
      <button
        type="submit"
        disabled={loading}
        className="mt-4 w-full bg-green-600 text-white py-2 px-4 rounded-md hover:bg-green-700 transition disabled:bg-gray-400"
      >
        {loading ? 'Finding recipes...' : 'Find Recipes'}
      </button>
    </form>
  );
}