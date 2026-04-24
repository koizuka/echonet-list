import { useState, useCallback, useMemo } from 'react';

const STORAGE_KEY = 'echonet-list-selected-tab';

/**
 * Hook for persisting tab selection across page reloads
 * Automatically saves and restores the selected tab from localStorage
 */
export function usePersistedTab(availableTabs: string[], defaultTab?: string) {
  // Get saved tab from localStorage on first load
  const getSavedTab = useCallback(() => {
    try {
      return localStorage.getItem(STORAGE_KEY);
    } catch (error) {
      console.warn('Failed to read from localStorage:', error);
      return null;
    }
  }, []);

  // Initialize with saved tab if available, otherwise use default
  const [selectedTab, setSelectedTab] = useState<string>(() => {
    const savedTab = getSavedTab();
    return savedTab || defaultTab || 'All';
  });

  // Compute the valid tab based on available tabs
  const validTab = useMemo(() => {
    if (availableTabs.length === 0) {
      return selectedTab; // Keep current selection if no tabs available yet
    }

    const savedTab = getSavedTab();

    // If we have a saved tab and it's available, use it
    if (savedTab && availableTabs.includes(savedTab)) {
      return savedTab;
    }

    // If current selected tab is available, keep it
    if (availableTabs.includes(selectedTab)) {
      return selectedTab;
    }

    // Otherwise, use fallback
    return defaultTab || availableTabs[0] || 'All';
  }, [availableTabs, getSavedTab, selectedTab, defaultTab]);

  // Adjust state during render (React recommended pattern), instead of in an
  // effect, to satisfy react-hooks/set-state-in-effect and avoid extra renders.
  if (availableTabs.length > 0 && validTab !== selectedTab) {
    setSelectedTab(validTab);
  }

  // Save to localStorage whenever tab changes
  const selectTab = useCallback((tabName: string) => {
    if (!availableTabs.includes(tabName)) {
      console.warn(`Attempted to select invalid tab: ${tabName}`);
      return;
    }

    setSelectedTab(tabName);
    
    try {
      localStorage.setItem(STORAGE_KEY, tabName);
    } catch (error) {
      // localStorage might not be available
      console.warn('Failed to save to localStorage:', error);
    }
  }, [availableTabs]);

  // Clear saved tab state (useful for testing or reset functionality)
  const clearPersistedTab = useCallback(() => {
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (error) {
      console.warn('Failed to clear localStorage:', error);
    }
  }, []);

  return {
    selectedTab,
    selectTab,
    clearPersistedTab
  };
}