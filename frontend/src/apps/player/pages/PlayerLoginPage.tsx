import { useEffect } from 'react';
import { Navigate } from 'react-router-dom';

export function PlayerLoginPage() {
  useEffect(() => {
    window.dispatchEvent(new CustomEvent('show-player-auth', { detail: 'login' }));
  }, []);

  return <Navigate to="/dashboard" replace />;
}
