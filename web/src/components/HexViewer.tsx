import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Binary } from 'lucide-react';
import { edtToHexString } from '@/libs/propertyHelper';
import type { PropertyValue } from '@/hooks/types';

interface HexViewerProps {
  canShowHexViewer: boolean;
  currentValue: PropertyValue;
  size?: 'sm' | 'normal';
}

export function HexViewer({ canShowHexViewer, currentValue, size = 'normal' }: HexViewerProps) {
  const [showHexData, setShowHexData] = useState(false);
  
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
          className={`absolute top-full left-0 right-0 mt-1 ${sizeClasses.text} font-mono bg-muted ${size === 'sm' ? 'p-1' : 'p-2'} rounded border break-all shadow-md z-10 overflow-x-auto`}
          role="status"
          aria-live="polite"
        >
          {edtToHexString(currentValue.EDT) || 'Invalid data'}
        </div>
      )}
    </>
  );
}