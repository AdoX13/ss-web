import React from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../button';
import ThemeToggle from '../../theme/ThemeToggle';
import type { Role } from '../../types/auth';
import { ROLE_LABELS } from '../../types/auth';

interface ButtonProps {
  text: string;
  onClick?: () => void;
  variant?: 'primary' | 'secondary' | 'outline';
  size?: 'sm' | 'md' | 'lg';
}

interface NavbarProps {
  title: string;
  leftButtons?: ButtonProps[];
  rightButtons?: ButtonProps[];
  user?: { email: string; role: Role } | null;
}

const Navbar: React.FC<NavbarProps> = ({
  title,
  leftButtons = [],
  rightButtons = [],
  user = null,
}) => {
  const navigate = useNavigate();

  return (
    <nav className="fixed top-0 left-0 right-0 bg-sky-50 dark:bg-gray-900 border-b border-transparent dark:border-gray-800 shadow-sm z-50">
      <div className="container mx-auto px-4 py-3 flex items-center justify-between gap-3">
        <div className="flex space-x-2 overflow-x-auto">
          {leftButtons.map((button, index) => (
            <Button
              key={index}
              text={button.text}
              onClick={button.onClick}
              variant={button.variant || 'outline'}
              size="sm"
            />
          ))}
        </div>

        <button
          type="button"
          onClick={() => navigate('/')}
          className="text-lg sm:text-xl font-semibold text-sky-700 dark:text-sky-300 hover:text-sky-800 dark:hover:text-sky-200 transition-colors whitespace-nowrap rounded focus:outline-none focus:ring-2 focus:ring-sky-400"
        >
          {title}
        </button>

        <div className="flex items-center space-x-2">
          {user && (
            <span className="hidden md:flex items-center gap-2 text-sm">
              <span className="px-2 py-0.5 rounded-full bg-sky-100 dark:bg-sky-900/40 text-sky-700 dark:text-sky-300 text-xs font-medium">
                {ROLE_LABELS[user.role] ?? user.role}
              </span>
              <span className="max-w-[160px] truncate text-gray-600 dark:text-gray-300">
                {user.email}
              </span>
            </span>
          )}
          <ThemeToggle />
          {rightButtons.map((button, index) => (
            <Button
              key={index}
              text={button.text}
              onClick={button.onClick}
              variant={button.variant || 'outline'}
              size="sm"
            />
          ))}
        </div>
      </div>
    </nav>
  );
};

export default Navbar;
