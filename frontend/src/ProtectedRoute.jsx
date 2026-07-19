import { Outlet, Navigate } from 'react-router-dom';
import { useAuth } from './AuthContext';

export const ProtectedRoute = () => {
  const { isLoggedIn } = useAuth();
  return isLoggedIn ? <Outlet /> : <Navigate to="/" replace />;
};

export const PublicOnlyRoute = () => {
  const { isLoggedIn } = useAuth();
  return isLoggedIn ? <Navigate to="/dashboard" replace /> : <Outlet />;
};

export const FallbackRoute = () => {
  const { isLoggedIn } = useAuth();
  return <Navigate to={isLoggedIn ? "/dashboard" : "/"} replace />;
};
