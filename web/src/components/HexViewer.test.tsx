import { describe, it, expect } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { HexViewer } from './HexViewer';

describe('HexViewer', () => {
  it('should render binary button when canShowHexViewer is true', () => {
    render(
      <HexViewer
        canShowHexViewer={true}
        currentValue={{ EDT: btoa('test') }}
      />
    );

    const button = screen.getByTitle('Show hex data');
    expect(button).toBeInTheDocument();
  });

  it('should not render when canShowHexViewer is false', () => {
    render(
      <HexViewer
        canShowHexViewer={false}
        currentValue={{ string: 'test' }}
      />
    );

    expect(screen.queryByTitle('Show hex data')).not.toBeInTheDocument();
  });

  it('should toggle hex data display when button is clicked', () => {
    const edt = btoa('Hello');
    render(
      <HexViewer
        canShowHexViewer={true}
        currentValue={{ EDT: edt }}
      />
    );

    // Initially hex data is not shown
    expect(screen.queryByText(/48656C6C6F/)).not.toBeInTheDocument();

    // Click to show hex data
    const button = screen.getByTitle('Show hex data');
    fireEvent.click(button);

    // Hex data should be shown (with spaces)
    expect(screen.getByText('48 65 6C 6C 6F')).toBeInTheDocument();
    expect(button).toHaveAttribute('title', 'Hide hex data');

    // Click to hide hex data
    fireEvent.click(button);

    // Hex data should be hidden
    expect(screen.queryByText('48 65 6C 6C 6F')).not.toBeInTheDocument();
    expect(button).toHaveAttribute('title', 'Show hex data');
  });

  it('should render inline when size is small', () => {
    render(
      <HexViewer
        canShowHexViewer={true}
        currentValue={{ EDT: btoa('test') }}
        size="sm"
      />
    );

    const button = screen.getByTitle('Show hex data');
    expect(button).toHaveClass('h-4', 'w-4');
  });

  it('should render normal size by default', () => {
    render(
      <HexViewer
        canShowHexViewer={true}
        currentValue={{ EDT: btoa('test') }}
      />
    );

    const button = screen.getByTitle('Show hex data');
    expect(button).toHaveClass('h-6', 'w-6');
  });

  it('should handle invalid EDT data gracefully', () => {
    render(
      <HexViewer
        canShowHexViewer={true}
        currentValue={{ EDT: 'invalid base64!' }}
      />
    );

    const button = screen.getByTitle('Show hex data');
    fireEvent.click(button);

    expect(screen.getByText('Invalid data')).toBeInTheDocument();
  });
});