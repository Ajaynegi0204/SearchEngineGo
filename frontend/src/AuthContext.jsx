import { createContext, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import axios from 'axios';
import { toast } from 'react-toastify';

const AuthContext = createContext(null);
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export function AuthProvider({ children }) {
  const navigate = useNavigate();
  const [user, setUser] = useState(() => {
    const storedUser = localStorage.getItem('user');
    return storedUser ? JSON.parse(storedUser) : null;
  });
  const [token, setToken] = useState(() => localStorage.getItem('access_token'));
  const [loading, setLoading] = useState(false);
  const tokenRef = useRef(token);

  const api = useMemo(() => axios.create({
    baseURL: API_BASE_URL,
    headers: { 'Content-Type': 'application/json' },
  }), []);

  useEffect(() => {
    tokenRef.current = token;
  }, [token]);

  useEffect(() => {
    const handleStorageChange = (event) => {
      if (event.key !== 'access_token' && event.key !== 'user') {
        return;
      }

      const nextToken = localStorage.getItem('access_token');
      const storedUser = localStorage.getItem('user');

      let nextUser = null;
      if (storedUser) {
        try {
          nextUser = JSON.parse(storedUser);
        } catch {
          nextUser = null;
        }
      }

      setToken(nextToken);
      tokenRef.current = nextToken;
      setUser(nextUser);

      if (!nextToken) {
        navigate('/', { replace: true });
      }
    };

    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, [navigate]);

  useEffect(() => {
    const requestInterceptor = api.interceptors.request.use((config) => {
      if (tokenRef.current) {
        config.headers.Authorization = `Bearer ${tokenRef.current}`;
      }
      return config;
    });

    const responseInterceptor = api.interceptors.response.use(
      (response) => response,
      async (error) => {
        const originalRequest = error.config;
        const refreshToken = localStorage.getItem('refresh_token');

        if (
          error.response?.status !== 401 ||
          originalRequest?._retry ||
          originalRequest?.url?.includes('/api/v1/auth/refresh') ||
          !refreshToken
        ) {
          return Promise.reject(error);
        }

        originalRequest._retry = true;

        try {
          const { data } = await axios.post(
            `${API_BASE_URL}/api/v1/auth/refresh`,
            { refresh_token: refreshToken },
          );

          saveTokens(data);
          originalRequest.headers.Authorization = `Bearer ${data.access_token}`;
          return api(originalRequest);
        } catch (refreshError) {
          clearAuth();
          return Promise.reject(refreshError);
        }
      },
    );

    return () => {
      api.interceptors.request.eject(requestInterceptor);
      api.interceptors.response.eject(responseInterceptor);
    };
  }, [api]);

  const saveTokens = (tokens) => {
    setToken(tokens.access_token);
    tokenRef.current = tokens.access_token;
    localStorage.setItem('access_token', tokens.access_token);
    localStorage.setItem('refresh_token', tokens.refresh_token);
  };

  const clearAuth = () => {
    setToken(null);
    setUser(null);
    tokenRef.current = null;
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');
  };

  const startSession = (authResponse) => {
    saveTokens(authResponse.tokens);
    setUser(authResponse.user);
    localStorage.setItem('user', JSON.stringify(authResponse.user));
    navigate('/dashboard');
  };

  const login = async (credentials) => {
    setLoading(true);
    try {
      const { data } = await api.post('/api/v1/auth/login', credentials);
      startSession(data);
      toast.success('Logged in successfully');
    } catch (error) {
      const message = error.response?.data?.error || 'Login failed';
      toast.error(message);
      throw error;
    } finally {
      setLoading(false);
    }
  };

  const logout = async () => {
    const refreshToken = localStorage.getItem('refresh_token');
    try {
      if (refreshToken) {
        await api.post('/api/v1/auth/logout', { refresh_token: refreshToken });
      }
    } finally {
      clearAuth();
      navigate('/');
    }
  };

  return (
    <AuthContext.Provider value={{ api, user, token, loading, isLoggedIn: Boolean(token), login, logout, startSession }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
