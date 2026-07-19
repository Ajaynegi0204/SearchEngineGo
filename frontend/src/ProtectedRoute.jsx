import { Outlet, Navigate } from 'react-router-dom';
import { useAuth } from './AuthContext';

export const ProtectedRoute = () => {
  const { user } = useAuth();
  return user ? <Outlet /> : <Navigate to="/" replace />;
};