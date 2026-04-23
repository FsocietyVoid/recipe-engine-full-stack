import { useState } from 'react';
import axios from 'axios';
import SearchBox from './components/SearchBox';
import RecipeCard from './components/RecipeCard';
import RecipeModal from './components/RecipeModal';
import type { MatchResult } from './types';
import './index.css';

function App() {
  const [results, setResults] = useState<MatchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedMatch, setSelectedMatch] = useState<MatchResult | null>(null);

  const handleSearch = async (query: string) => {
    if (!query.trim()) return;
    setLoading(true);
    try {
      const response = await axios.get<MatchResult[]>('/api/suggest', {
        params: { ingredients: query },
      });
      setResults(response.data);
    } catch (err) {
      console.error(err);
      alert('Failed to fetch recipes. Make sure backend is running on port 8080.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="container mx-auto px-4 py-8 max-w-6xl">
      <h1 className="text-4xl font-bold text-center mb-8">🍽️ Recipe Engine</h1>
      <SearchBox onSearch={handleSearch} loading={loading} />
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-8">
        {results.map((match, idx) => (
          <RecipeCard key={idx} match={match} onClick={() => setSelectedMatch(match)} />
        ))}
      </div>
      {results.length === 0 && !loading && (
        <p className="text-center text-gray-500 mt-8">
          No recipes yet. Try adding some ingredients above.
        </p>
      )}
      {selectedMatch && (
        <RecipeModal
          recipe={selectedMatch.Recipe}
          matchedCount={selectedMatch.MatchedCount}
          totalRequired={selectedMatch.TotalRequired}
          score={selectedMatch.Score}
          onClose={() => setSelectedMatch(null)}
        />
      )}
    </div>
  );
}

export default App;