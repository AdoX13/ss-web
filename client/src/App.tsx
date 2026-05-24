import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  Outlet,
  useNavigate,
} from 'react-router-dom';
import Navbar from './components/navbar';
import HomePage from './pages/homePage';
import LoginPage from './pages/loginPage';
import RegisterPage from './pages/register';
import PhotosPage from './pages/photosPage';
import DevicesPage from './pages/devicesPage';
import StatisticsPage from './pages/statisticsPage';
import ReviewQueuePage from './pages/reviewQueuePage';
import ReportsPage from './pages/reportsPage';
import ReportDetailPage from './pages/reportsPage/ReportDetail';
import AdminUsersPage from './pages/adminPage';
import ProtectedRoute from './components/ProtectedRoute';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { ThemeProvider } from './theme/ThemeContext';

interface NavButton {
  text: string;
  variant: 'primary' | 'secondary' | 'outline';
  onClick: () => void;
}

const Layout = () => {
  const navigate = useNavigate();
  const { isLoggedIn, role, email, logout, hasRole } = useAuth();

  const navTo = (text: string, path: string): NavButton => ({
    text,
    variant: 'secondary',
    onClick: () => navigate(path),
  });

  // Role-aware navigation. RBAC is enforced server-side too; this just hides
  // links a role can't use.
  const leftButtons: NavButton[] = [];
  if (isLoggedIn) {
    if (hasRole('admin', 'doctor')) leftButtons.push(navTo('Photos', '/photos'));
    if (hasRole('admin', 'doctor')) leftButtons.push(navTo('Devices', '/devices'));
    if (hasRole('admin', 'doctor', 'researcher'))
      leftButtons.push(navTo('Statistics', '/statistics'));
    leftButtons.push(navTo('Reports', '/reports'));
    if (hasRole('admin', 'doctor'))
      leftButtons.push(navTo('Review Queue', '/review-queue'));
    if (hasRole('admin')) leftButtons.push(navTo('Admin', '/admin/users'));
  }

  const rightButtons: NavButton[] = isLoggedIn
    ? [
        {
          text: 'Logout',
          variant: 'outline',
          onClick: () => {
            void logout().then(() => navigate('/'));
          },
        },
      ]
    : [
        { text: 'Login', variant: 'outline', onClick: () => navigate('/login') },
        {
          text: 'Register',
          variant: 'primary',
          onClick: () => navigate('/register'),
        },
      ];

  return (
    <div className="min-h-screen bg-white dark:bg-slate-900 text-gray-900 dark:text-gray-100 transition-colors">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-[60] focus:px-4 focus:py-2 focus:bg-sky-700 focus:text-white focus:rounded-md"
      >
        Skip to content
      </a>
      <Navbar
        title="Security of Systems - First Force"
        leftButtons={leftButtons}
        rightButtons={rightButtons}
        user={isLoggedIn && role ? { email: email ?? '', role } : null}
      />
      <main id="main-content" className="pt-16 px-4">
        <Outlet />
      </main>
    </div>
  );
};

const App = () => {
  return (
    <ThemeProvider>
      <BrowserRouter>
        <AuthProvider>
          <Routes>
            <Route element={<Layout />}>
              {/* Public */}
              <Route path="/" element={<HomePage />} />

              {/* Auth pages — only for logged-out users */}
              <Route element={<ProtectedRoute authRequired={false} />}>
                <Route path="/login" element={<LoginPage />} />
                <Route path="/register" element={<RegisterPage />} />
              </Route>

              {/* Photos / Devices — admin + doctor */}
              <Route element={<ProtectedRoute roles={['admin', 'doctor']} />}>
                <Route path="/photos" element={<PhotosPage />} />
                <Route path="/devices" element={<DevicesPage />} />
              </Route>

              {/* Statistics — admin + doctor + researcher */}
              <Route
                element={
                  <ProtectedRoute roles={['admin', 'doctor', 'researcher']} />
                }
              >
                <Route path="/statistics" element={<StatisticsPage />} />
              </Route>

              {/* Reports — any authenticated role (per-report RBAC server-side) */}
              <Route element={<ProtectedRoute />}>
                <Route path="/reports" element={<ReportsPage />} />
                <Route path="/reports/:name" element={<ReportDetailPage />} />
              </Route>

              {/* Review queue — admin + doctor */}
              <Route element={<ProtectedRoute roles={['admin', 'doctor']} />}>
                <Route path="/review-queue" element={<ReviewQueuePage />} />
              </Route>

              {/* Admin — admin only */}
              <Route element={<ProtectedRoute roles={['admin']} />}>
                <Route path="/admin/users" element={<AdminUsersPage />} />
              </Route>

              {/* Fallback */}
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  );
};

export default App;
