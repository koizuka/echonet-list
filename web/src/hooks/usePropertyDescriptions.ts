import { useEffect, useCallback } from 'react';
import { useECHONET } from './useECHONET';
import { extractClassCodeFromEOJ } from '@/libs/propertyHelper';
import { getCurrentLocale } from '@/libs/languageHelper';
import type { Device, ServerMessage } from './types';

/**
 * Hook that automatically fetches property descriptions for devices
 * and provides utility functions for working with property names
 */

export function usePropertyDescriptions(
  wsUrl: string,
  onMessage?: (message: ServerMessage) => void,
  onWebSocketConnected?: () => void
) {
  const echonet = useECHONET(wsUrl, onMessage, onWebSocketConnected);

  // Get unique class codes from all devices (including language-specific keys)
  const getUniqueClassCodes = useCallback((devices: Record<string, Device>): Set<string> => {
    const classCodes = new Set<string>();
    const currentLang = getCurrentLocale();

    // Always include common properties (empty classCode)
    classCodes.add('');
    // Add language-specific common properties if not English
    if (currentLang !== 'en') {
      classCodes.add(`:${currentLang}`);
    }

    // Add class codes from devices
    Object.values(devices).forEach(device => {
      const classCode = extractClassCodeFromEOJ(device.eoj);
      if (classCode) {
        classCodes.add(classCode);
        // Add language-specific class code if not English
        if (currentLang !== 'en') {
          classCodes.add(`${classCode}:${currentLang}`);
        }
      }
    });

    return classCodes;
  }, []);

  // Fetch property descriptions for missing class codes
  useEffect(() => {
    if (echonet.connectionState !== 'connected') {
      return;
    }

    const uniqueClassCodes = getUniqueClassCodes(echonet.devices);
    const missingClassCodes = Array.from(uniqueClassCodes).filter(
      classCode => !echonet.propertyDescriptions[classCode]
    );

    // Fetch descriptions for missing class codes
    missingClassCodes.forEach(classCode => {
      // Extract actual class code and language from the key (e.g., "0130:ja" -> classCode="0130", lang="ja")
      const [actualClassCode, lang] = classCode.includes(':') ? classCode.split(':') : [classCode, undefined];

      echonet.getPropertyDescription(actualClassCode, lang).catch(error => {
        if (import.meta.env.DEV) {
          console.warn(`Failed to fetch property description for class code ${classCode}:`, error);
        }
      });
    });
  }, [echonet, getUniqueClassCodes]);

  return {
    ...echonet,
    
    // Helper function to get class code from device
    getDeviceClassCode: useCallback((device: Device): string => {
      return extractClassCodeFromEOJ(device.eoj);
    }, []),
  };
}