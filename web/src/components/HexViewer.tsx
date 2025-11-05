import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Binary } from 'lucide-react';
import { edtToHexString } from '@/libs/propertyHelper';
import type { PropertyValue } from '@/hooks/types';

interface HexViewerProps {
  canShowHexViewer: boolean;
  currentValue: PropertyValue;
  size?: 'sm' | 'normal';
}

// Mobile breakpoint matching Tailwind's 'sm' breakpoint
const MOBILE_BREAKPOINT = 640;

export function HexViewer({ canShowHexViewer, currentValue, size = 'normal' }: HexViewerProps) {
  const [showHexData, setShowHexData] = useState(false);
  const hexViewerRef = useRef<HTMLDivElement>(null);
  const [leftOffset, setLeftOffset] = useState(0);

  useEffect(() => {
    const RESIZE_DEBOUNCE_MS = 150;
    const DEFAULT_PADDING_PX = 12; // Tailwind px-3 = 0.75rem = 12px

    const calculateOffset = () => {
      if (!showHexData || !hexViewerRef.current || window.innerWidth >= MOBILE_BREAKPOINT) {
        setLeftOffset(0);
        return;
      }

      // Calculate offset from PropertyRow (the row) to card's content edge on mobile
      // Find the PropertyRow which is the nearest .relative ancestor
      const propertyRow = hexViewerRef.current.closest('.relative');
      const card = hexViewerRef.current.closest('[data-testid^="device-card-"]');

      if (propertyRow && card) {
        const rowRect = propertyRow.getBoundingClientRect();
        const cardRect = card.getBoundingClientRect();
        const cardContent = card.querySelector('[class*="px-3"]');
        const padding = cardContent
          ? parseFloat(window.getComputedStyle(cardContent).paddingLeft)
          : DEFAULT_PADDING_PX;

        // Calculate how far left from PropertyRow to card's content left edge
        const offset = rowRect.left - cardRect.left - padding;
        setLeftOffset(-offset);
      }
    };

    calculateOffset(); // Initial calculation

    // Debounce resize events to prevent performance issues
    let resizeTimer: ReturnType<typeof setTimeout>;
    const debouncedCalculateOffset = () => {
      clearTimeout(resizeTimer);
      resizeTimer = setTimeout(calculateOffset, RESIZE_DEBOUNCE_MS);
    };

    // Add resize listener to recalculate on window resize or device rotation
    window.addEventListener('resize', debouncedCalculateOffset);
    return () => {
      window.removeEventListener('resize', debouncedCalculateOffset);
      clearTimeout(resizeTimer);
    };
  }, [showHexData]);

  if (!canShowHexViewer) return null;

  const sizeClasses = size === 'sm'
    ? { button: 'h-4 w-4', text: 'text-xs' }
    : { button: 'h-6 w-6', text: 'text-xs' };

  return (
    <>
      <Button
        variant={showHexData ? "default" : "outline"}
        size="sm"
        onClick={() => setShowHexData(!showHexData)}
        className={`${sizeClasses.button} p-0`}
        title={showHexData ? "Hide hex data" : "Show hex data"}
      >
        <Binary className={size === 'sm' ? "h-2 w-2" : "h-3 w-3"} />
      </Button>
      {showHexData && currentValue.EDT && (
        <div
          ref={hexViewerRef}
          className={`absolute top-full mt-1 ${sizeClasses.text} font-mono bg-muted ${size === 'sm' ? 'p-1' : 'p-2'} rounded border break-words shadow-md z-[100] left-0 right-0 sm:right-auto sm:min-w-[400px] sm:max-w-[600px]`}
          style={{
            // On mobile, dynamically calculate offset to reach card's content left edge
            left: window.innerWidth < MOBILE_BREAKPOINT && leftOffset !== 0 ? `${leftOffset}px` : undefined,
          }}
          role="status"
          aria-live="polite"
        >
          {edtToHexString(currentValue.EDT) || 'Invalid data'}
        </div>
      )}
    </>
  );
}