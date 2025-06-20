// Language detection and localization utilities

/**
 * Detect if the browser language is Japanese
 */
export function isJapanese(): boolean {
  const language = navigator.language || navigator.languages?.[0] || 'en';
  return language.toLowerCase().startsWith('ja');
}

/**
 * Get the current locale based on browser language
 */
export function getCurrentLocale(): 'ja' | 'en' {
  return isJapanese() ? 'ja' : 'en';
}