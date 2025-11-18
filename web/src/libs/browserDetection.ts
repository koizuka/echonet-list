/**
 * Browser detection utilities for handling browser-specific behavior
 */

/**
 * Detects if the current browser is iOS Safari (excluding Chrome/Firefox/Edge/Opera on iOS)
 *
 * iOS Safari has specific WebSocket behavior when pages are restored from background:
 * - WebSocket connections may become "zombies" (report OPEN but are actually dead)
 * - bfcache (back-forward cache) restoration may not properly restore connections
 *
 * Why User-Agent detection instead of feature detection:
 * - The WebSocket zombie connection issue is browser-specific, not feature-detectable
 * - WebSocket.readyState still reports OPEN even when connection is dead (iOS Safari bug)
 * - No reliable JavaScript API to detect this condition without actually sending data
 * - Other iOS browsers (Chrome, Firefox) use different WebView engines and don't exhibit this behavior
 *
 * @returns true if running on iOS Safari, false otherwise
 */
export function isIOSSafari(): boolean {
  if (typeof navigator === 'undefined') {
    return false;
  }

  const userAgent = navigator.userAgent;

  // Check for iOS device
  const isIOS = /iPhone|iPad|iPod/.test(userAgent);

  // Check for Safari browser (and not Chrome, Firefox, Edge, or Opera on iOS)
  const isSafari = /Safari/.test(userAgent);
  const isNotOtherBrowser = !/CriOS|FxiOS|EdgiOS|OPiOS/.test(userAgent);

  return isIOS && isSafari && isNotOtherBrowser;
}

/**
 * Gets the current user agent string
 *
 * @returns user agent string or empty string if not available
 */
export function getUserAgent(): string {
  return typeof navigator !== 'undefined' ? navigator.userAgent : '';
}
