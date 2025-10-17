import { useState, useEffect, useCallback, useMemo, useRef } from 'react';

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

  // Track if this is the initial render to avoid unnecessary state updates
  const isInitialMount = useRef(true);

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

  // Update state only when validTab changes and it's different from current
  useEffect(() => {
    if (availableTabs.length === 0) {
      return;
    }

    // On initial mount, apply the valid tab
    if (isInitialMount.current) {
      isInitialMount.current = false;
      if (validTab !== selectedTab) {
        setSelectedTab(validTab);
      }
      return;
    }

    // On subsequent updates, only update if the valid tab changed
    if (validTab !== selectedTab) {
      setSelectedTab(validTab);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [validTab]); // Only depend on validTab, not on availableTabs or selectedTab directly to avoid infinite loops

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