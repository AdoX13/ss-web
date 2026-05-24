import React, { useCallback, useEffect, useState } from 'react';
import { apiJson, apiSend, postJson, ApiError } from '../../utils/api';
import type { User } from '../../types/user';
import type { Role } from '../../types/auth';
import { ALL_ROLES, ROLE_LABELS } from '../../types/auth';
import { useAuth } from '../../contexts/AuthContext';
import {
  card,
  heading,
  label,
  input,
  subtleText,
  errorBanner,
  successBanner,
  tableHeadCell,
  tableCell,
} from '../../theme/styles';

const MIN_PASSWORD_LENGTH = 8;

const AdminUsersPage: React.FC = () => {
  const { email: currentEmail } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [newEmail, setNewEmail] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newRole, setNewRole] = useState<Role>('doctor');
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    try {
      const data = await apiJson<User[]>('/api/v1/users');
      setUsers(Array.isArray(data) ? data : []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Failed to load users.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setNotice(null);
    if (newPassword.length < MIN_PASSWORD_LENGTH) {
      setError(`Password must be at least ${MIN_PASSWORD_LENGTH} characters.`);
      return;
    }
    setCreating(true);
    try {
      await postJson('/api/v1/users', {
        email: newEmail,
        password: newPassword,
        role: newRole,
      });
      setNotice(`Created ${newEmail}.`);
      setNewEmail('');
      setNewPassword('');
      setNewRole('doctor');
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to create user.');
    } finally {
      setCreating(false);
    }
  };

  const changeRole = async (user: User, role: Role) => {
    setError(null);
    setNotice(null);
    try {
      await apiSend(`/api/v1/users/${encodeURIComponent(user.email)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role }),
      });
      setNotice(`Updated role for ${user.email}.`);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to update role.');
    }
  };

  const deactivate = async (user: User) => {
    if (!window.confirm(`Deactivate ${user.email}? They will lose access.`)) {
      return;
    }
    setError(null);
    setNotice(null);
    try {
      await apiSend(`/api/v1/users/${encodeURIComponent(user.email)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ deactivate: true }),
      });
      setNotice(`Deactivated ${user.email}.`);
      await load();
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : 'Failed to deactivate user.',
      );
    }
  };

  return (
    <div className="container mx-auto pb-10">
      <h1 className={`${heading} mb-6`}>User Management</h1>

      {error && (
        <div className={`mb-4 ${errorBanner}`} role="alert">
          {error}
        </div>
      )}
      {notice && (
        <div className={`mb-4 ${successBanner}`} role="status">
          {notice}
        </div>
      )}

      {/* Create user */}
      <form onSubmit={handleCreate} className={`${card} p-4 mb-6`}>
        <h2 className="text-lg font-medium text-gray-800 dark:text-gray-100 mb-3">
          Add user
        </h2>
        <div className="flex flex-wrap items-end gap-4">
          <div className="flex-1 min-w-[200px]">
            <label htmlFor="new-email" className={label}>
              Email
            </label>
            <input
              id="new-email"
              type="email"
              className={input}
              value={newEmail}
              onChange={(e) => setNewEmail(e.target.value)}
              required
              disabled={creating}
            />
          </div>
          <div className="flex-1 min-w-[200px]">
            <label htmlFor="new-password" className={label}>
              Password
            </label>
            <input
              id="new-password"
              type="password"
              className={input}
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              minLength={MIN_PASSWORD_LENGTH}
              autoComplete="new-password"
              disabled={creating}
            />
          </div>
          <div>
            <label htmlFor="new-role" className={label}>
              Role
            </label>
            <select
              id="new-role"
              className={input}
              value={newRole}
              onChange={(e) => setNewRole(e.target.value as Role)}
              disabled={creating}
            >
              {ALL_ROLES.map((r) => (
                <option key={r} value={r}>
                  {ROLE_LABELS[r]}
                </option>
              ))}
            </select>
          </div>
          <button
            type="submit"
            disabled={creating}
            className="px-4 py-2 bg-sky-600 text-white rounded-md hover:bg-sky-700 transition-colors disabled:opacity-50 focus:outline-none focus:ring-2 focus:ring-sky-400"
          >
            {creating ? 'Creating…' : 'Create'}
          </button>
        </div>
      </form>

      {/* User list */}
      {loading ? (
        <div className="flex justify-center items-center h-40">
          <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-sky-500" />
        </div>
      ) : users.length === 0 ? (
        <div className={`${card} text-center py-12 ${subtleText}`}>
          No users found.
        </div>
      ) : (
        <div className={`${card} overflow-x-auto`}>
          <table className="min-w-full">
            <thead className="bg-gray-50 dark:bg-gray-700/50">
              <tr>
                <th scope="col" className={tableHeadCell}>
                  Email
                </th>
                <th scope="col" className={tableHeadCell}>
                  Role
                </th>
                <th scope="col" className={tableHeadCell}>
                  Status
                </th>
                <th scope="col" className={tableHeadCell}>
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.email}>
                  <td className={tableCell}>
                    {u.email}
                    {u.email === currentEmail && (
                      <span className="ml-2 text-xs text-gray-400">(you)</span>
                    )}
                  </td>
                  <td className={tableCell}>
                    <select
                      aria-label={`Role for ${u.email}`}
                      className={`${input} max-w-[180px]`}
                      value={u.role}
                      onChange={(e) =>
                        void changeRole(u, e.target.value as Role)
                      }
                    >
                      {ALL_ROLES.map((r) => (
                        <option key={r} value={r}>
                          {ROLE_LABELS[r]}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className={tableCell}>
                    {u.active ? (
                      <span className="px-2 py-0.5 rounded-full text-xs bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300">
                        Active
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 rounded-full text-xs bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300">
                        Inactive
                      </span>
                    )}
                  </td>
                  <td className={tableCell}>
                    <button
                      onClick={() => void deactivate(u)}
                      disabled={!u.active || u.email === currentEmail}
                      className="px-3 py-1 text-sm bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-red-400"
                    >
                      Deactivate
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
};

export default AdminUsersPage;
