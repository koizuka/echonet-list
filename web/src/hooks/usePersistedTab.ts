import { useState, useEffect, useCallback } from 'react';

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

  // Update selected tab when available tabs change
  useEffect(() => {
    const savedTab = getSavedTab();
    
    // If we don't have available tabs yet, wait
    if (availableTabs.length === 0) {
      return;
    }
    
    // If we have a saved tab and it's available, use it
    if (savedTab && availableTabs.includes(savedTab) && selectedTab !== savedTab) {
      setSelectedTab(savedTab);
      return;
    }
    
    // If current selected tab is no longer available, reset to a valid one
    if (!availableTabs.includes(selectedTab)) {
      const fallbackTab = defaultTab || availableTabs[0] || 'All';
      setSelectedTab(fallbackTab);
    }
  }, [availableTabs, getSavedTab, selectedTab, defaultTab]);

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