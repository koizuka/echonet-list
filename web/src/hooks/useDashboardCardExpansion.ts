import { useState, useCallback, useEffect } from 'react';

const STORAGE_KEY = 'echonet-list-dashboard-expanded-cards';

/**
 * Hook for managing and persisting DashboardCard expansion state
 * Stores expanded card keys in localStorage so they persist across page reloads
 */
export function useDashboardCardExpansion() {
  const [expandedCards, setExpandedCards] = useState<Set<string>>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        return new Set(JSON.parse(saved) as string[]);
      }
    } catch (error) {
      console.warn('Failed to load dashboard expansion state:', error);
    }
    return new Set();
  });

  // Persist to localStorage whenever expandedCards changes
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify([...expandedCards]));
    } catch (error) {
      console.warn('Failed to save dashboard expansion state:', error);
    }
  }, [expandedCards]);

  const isExpanded = useCallback((deviceKey: string) => {
    return expandedCards.has(deviceKey);
  }, [expandedCards]);

  const toggleExpansion = useCallback((deviceKey: string) => {
    setExpandedCards(prev => {
      const newSet = new Set(prev);
      if (newSet.has(deviceKey)) {
        newSet.delete(deviceKey);
      } else {
        newSet.add(deviceKey);
      }
      return newSet;
    });
  }, []);

  return {
    isExpanded,
    toggleExpansion
  };
}
