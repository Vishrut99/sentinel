'use client';

import { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { getStoredToken, getStoredUser, storeAuth, clearAuth } from '@/lib/api';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  // Initialize auth state from localStorage
  useEffect(() => {
    const storedToken = getStoredToken();
    const storedUser = getStoredUser();
    
    if (storedToken && storedUser) {
      setToken(storedToken);
      setUser(storedUser);
    }
    setLoading(false);
  }, []);

  const login = useCallback((newToken, newUser) => {
    storeAuth(newToken, newUser);
    setToken(newToken);
    setUser(newUser);
    
    // Route based on role
    if (newUser.role === 'user') {
      router.push('/tickets');
    } else {
      router.push('/dashboard');
    }
  }, [router]);

  const logout = useCallback(() => {
    clearAuth();
    setToken(null);
    setUser(null);
    router.push('/auth');
  }, [router]);

  const isAuthenticated = !!token && !!user;
  const isUser = user?.role === 'user';
  const isAgent = user?.role === 'agent';
  const isAdmin = user?.role === 'admin';
  const isAgentOrAdmin = isAgent || isAdmin;

  return (
    <AuthContext.Provider value={{
      user,
      token,
      loading,
      isAuthenticated,
      isUser,
      isAgent,
      isAdmin,
      isAgentOrAdmin,
      login,
      logout,
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
