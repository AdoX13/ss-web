import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../../contexts/AuthContext';
import { postJson, ApiError } from '../../utils/api';
import type { LoginResponse } from '../../types/auth';
import { card, heading, label, input, errorBanner } from '../../theme/styles';

const LoginPage: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const { login } = useAuth();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const data = await postJson<LoginResponse>('/api/v1/auth/login', {
        email,
        password,
      });
      login(data);
      navigate('/');
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : 'An error occurred during login. Please try again.',
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex justify-center items-center min-h-[80vh]">
      <div className={`${card} p-8 w-full max-w-md`}>
        <h2 className={`${heading} mb-6 text-center`}>Login</h2>

        {error && (
          <div className={`mb-4 ${errorBanner}`} role="alert">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} noValidate>
          <div className="mb-4">
            <label htmlFor="email" className={label}>
              Email Address
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={input}
              required
              disabled={loading}
            />
          </div>

          <div className="mb-6">
            <label htmlFor="password" className={label}>
              Password
            </label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={input}
              required
              disabled={loading}
            />
          </div>

          <button
            type="submit"
            className="w-full px-6 py-3 text-lg bg-sky-700 text-white hover:bg-sky-800 inline-flex items-center justify-center rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-sky-400 focus:ring-offset-2 dark:focus:ring-offset-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
            disabled={loading}
          >
            {loading ? 'Logging in…' : 'Login'}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-gray-600 dark:text-gray-400">
          Don&rsquo;t have an account?{' '}
          <Link
            to="/register"
            className="text-sky-700 dark:text-sky-400 hover:underline font-medium"
          >
            Register
          </Link>
        </p>
      </div>
    </div>
  );
};

export default LoginPage;
