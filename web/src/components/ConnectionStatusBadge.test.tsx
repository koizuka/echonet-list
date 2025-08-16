import { render, screen } from '@testing-library/react';
import { ConnectionStatusBadge } from './ConnectionStatusBadge';
import type { ConnectionState } from '@/hooks/types';

describe('ConnectionStatusBadge', () => {
  it('should display Wifi icon and Connected text when connected', () => {
    render(<ConnectionStatusBadge connectionState="connected" />);
    
    expect(screen.getByTestId('connection-status')).toBeInTheDocument();
    expect(screen.getByText('Connected')).toBeInTheDocument();
    expect(screen.getByTestId('connection-icon')).toBeInTheDocument();
  });

  it('should display WifiOff icon and Disconnected text when disconnected', () => {
    render(<ConnectionStatusBadge connectionState="disconnected" />);
    
    expect(screen.getByTestId('connection-status')).toBeInTheDocument();
    expect(screen.getByText('Disconnected')).toBeInTheDocument();
    expect(screen.getByTestId('connection-icon')).toBeInTheDocument();
  });

  it('should display rotating Loader2 icon and Connecting text when connecting', () => {
    render(<ConnectionStatusBadge connectionState="connecting" />);
    
    const badge = screen.getByTestId('connection-status');
    const icon = screen.getByTestId('connection-icon');
    
    expect(badge).toBeInTheDocument();
    expect(screen.getByText('Connecting')).toBeInTheDocument();
    expect(icon).toBeInTheDocument();
    expect(icon).toHaveClass('animate-spin');
  });

  it('should display AlertCircle icon and Error text when error', () => {
    render(<ConnectionStatusBadge connectionState="error" />);
    
    expect(screen.getByTestId('connection-status')).toBeInTheDocument();
    expect(screen.getByText('Error')).toBeInTheDocument();
    expect(screen.getByTestId('connection-icon')).toBeInTheDocument();
  });

  it('should apply correct color classes for each state', () => {
    const states: ConnectionState[] = ['connected', 'disconnected', 'connecting', 'error'];
    const expectedColors = {
      connected: 'bg-green-500',
      disconnected: 'bg-gray-500',
      connecting: 'bg-yellow-500',
      error: 'bg-red-500'
    };

    states.forEach(state => {
      const { rerender } = render(<ConnectionStatusBadge connectionState={state} />);
      const badge = screen.getByTestId('connection-status');
      expect(badge).toHaveClass(expectedColors[state]);
      rerender(<div />); // Clean up for next iteration
    });
  });

  it('should apply animation classes for connecting state', () => {
    render(<ConnectionStatusBadge connectionState="connecting" />);
    
    const icon = screen.getByTestId('connection-icon');
    expect(icon).toHaveClass('animate-spin');
  });

  it('should not apply badge animation for connected state', () => {
    render(<ConnectionStatusBadge connectionState="connected" />);
    
    const badge = screen.getByTestId('connection-status');
    expect(badge).not.toHaveClass('animate-pulse');
  });

  it('should not apply badge animation for disconnected and error states', () => {
    ['disconnected', 'error'].forEach(state => {
      const { rerender } = render(<ConnectionStatusBadge connectionState={state as ConnectionState} />);
      const badge = screen.getByTestId('connection-status');
      expect(badge).not.toHaveClass('animate-pulse');
      rerender(<div />); // Clean up for next iteration
    });
  });
});