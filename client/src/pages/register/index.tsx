import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { apiFetch, ApiError } from '../../utils/api';
import {
  card,
  heading,
  label,
  input,
  errorBanner,
  successBanner,
} from '../../theme/styles';

const MIN_PASSWORD_LENGTH = 8;

const RegisterPage: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    // Mirror the backend rule (handoff §1.1) for instant client feedback.
    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(`Password must be at least ${MIN_PASSWORD_LENGTH} characters.`);
      return;
    }

    setLoading(true);
    try {
      const res = await apiFetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) {
        const message = (await res.text()).trim();
        if (res.status === 409) {
          throw new ApiError(409, 'That email is already registered.');
        }
        throw new ApiError(res.status, message || 'Registration failed.');
      }
      setSuccess('Registration successful! Redirecting to login…');
      setTimeout(() => navigate('/login'), 1500);
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : 'An error occurred during registration. Please try again.',
      );
    } finally {
      setLoading(false);
    }
  };

  const done = success !== null;

  return (
    <div className="flex justify-center items-center min-h-[80vh]">
      <div className={`${card} p-8 w-full max-w-md`}>
        <h2 className={`${heading} mb-6 text-center`}>Register</h2>

        {error && (
          <div className={`mb-4 ${errorBanner}`} role="alert">
            {error}
          </div>
        )}
        {success && (
          <div className={`mb-4 ${successBanner}`} role="status">
            {success}
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
              disabled={loading || done}
            />
          </div>

          <div className="mb-6">
            <label htmlFor="password" className={label}>
              Password
            </label>
            <input
              id="password"
              type="password"
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={input}
              required
              minLength={MIN_PASSWORD_LENGTH}
              disabled={loading || done}
              aria-describedby="password-hint"
            />
            <p
              id="password-hint"
              className="mt-1 text-xs text-gray-500 dark:text-gray-400"
            >
              At least {MIN_PASSWORD_LENGTH} characters.
            </p>
          </div>

          <button
            type="submit"
            className="w-full px-6 py-3 text-lg bg-sky-700 text-white hover:bg-sky-800 inline-flex items-center justify-center rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-sky-400 focus:ring-offset-2 dark:focus:ring-offset-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
            disabled={loading || done}
          >
            {loading ? 'Registering…' : 'Register'}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-gray-600 dark:text-gray-400">
          Already have an account?{' '}
          <Link
            to="/login"
            className="text-sky-700 dark:text-sky-400 hover:underline font-medium"
          >
            Login
          </Link>
        </p>
      </div>
    </div>
  );
};

export default RegisterPage;
