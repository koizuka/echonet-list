import { useState, useCallback } from 'react';

/**
 * Hook for managing card expansion state across multiple devices
 * Each device card can be expanded/collapsed independently
 */
export function useCardExpansion() {
  // Track expanded state for each device by device key (ip + eoj)
  const [expandedCards, setExpandedCards] = useState<Set<string>>(new Set());

  // Toggle expansion state for a specific device
  const toggleCard = useCallback((deviceKey: string) => {
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

  // Check if a specific card is expanded
  const isCardExpanded = useCallback((deviceKey: string) => {
    return expandedCards.has(deviceKey);
  }, [expandedCards]);

  // Expand a specific card
  const expandCard = useCallback((deviceKey: string) => {
    setExpandedCards(prev => new Set(prev).add(deviceKey));
  }, []);

  // Collapse a specific card
  const collapseCard = useCallback((deviceKey: string) => {
    setExpandedCards(prev => {
      const newSet = new Set(prev);
      newSet.delete(deviceKey);
      return newSet;
    });
  }, []);

  // Expand all cards
  const expandAll = useCallback((deviceKeys: string[]) => {
    setExpandedCards(new Set(deviceKeys));
  }, []);

  // Collapse all cards
  const collapseAll = useCallback(() => {
    setExpandedCards(new Set());
  }, []);

  return {
    isCardExpanded,
    toggleCard,
    expandCard,
    collapseCard,
    expandAll,
    collapseAll,
    expandedCount: expandedCards.size
  };
}