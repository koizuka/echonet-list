import { render, screen } from '@testing-library/react';
import App from './App';

describe('App', () => {
  it('renders ECHONET List title', () => {
    render(<App />);
    expect(screen.getByText('ECHONET List')).toBeInTheDocument();
  });
});