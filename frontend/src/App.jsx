import { Routes, Route } from 'react-router-dom';
import { AuthProvider } from './AuthContext';
import { FallbackRoute, ProtectedRoute, PublicOnlyRoute } from './ProtectedRoute';
import Signup from './Signup';
import Home from './Home';
import Dashboard from './Dashboard';
import { ToastContainer } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';

function App() {
  return (
    <AuthProvider>
      <ToastContainer position="top-right" theme="dark" />
      <Routes>
        <Route element={<PublicOnlyRoute />}>
          <Route path="/" element={<Home />} />
          <Route path="/signup" element={<Signup />} />
        </Route>

        <Route element={<ProtectedRoute />}>
          <Route path="/dashboard" element={<Dashboard />} />
        </Route>

        <Route path="*" element={<FallbackRoute />} />
      </Routes>
    </AuthProvider>
  );
}

export default App;
