import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import ProtectedRoute from './ProtectedRoute';
import { AuthProvider } from '../contexts/AuthContext';
import { tokenStore } from '../utils/api';
import type { Role } from '../types/auth';

function renderApp(path: string, role?: Role) {
  if (role) {
    tokenStore.setSession({
      accessToken: 'A',
      refreshToken: 'R',
      email: 'e@x',
      role,
    });
  } else {
    tokenStore.clear();
  }
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<div>Login Page</div>} />
          <Route element={<ProtectedRoute roles={['admin']} />}>
            <Route path="/admin" element={<div>Admin Area</div>} />
          </Route>
          <Route element={<ProtectedRoute />}>
            <Route path="/any" element={<div>Any Authed</div>} />
          </Route>
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => localStorage.clear());

describe('ProtectedRoute', () => {
  it('redirects unauthenticated users to /login', async () => {
    renderApp('/admin');
    expect(await screen.findByText('Login Page')).toBeInTheDocument();
  });

  it('renders the route for an allowed role', async () => {
    renderApp('/admin', 'admin');
    expect(await screen.findByText('Admin Area')).toBeInTheDocument();
  });

  it('shows Access denied when the role is not allowed', async () => {
    renderApp('/admin', 'doctor');
    expect(await screen.findByText('Access denied')).toBeInTheDocument();
  });

  it('allows any authenticated role when no roles are required', async () => {
    renderApp('/any', 'auditor');
    expect(await screen.findByText('Any Authed')).toBeInTheDocument();
  });
});
