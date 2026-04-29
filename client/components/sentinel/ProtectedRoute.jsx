'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { FullPageSpinner } from './LoadingSpinner';

export function ProtectedRoute({ children, allowedRoles = [] }) {
  const { isAuthenticated, user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading) {
      if (!isAuthenticated) {
        router.push('/auth');
        return;
      }

      // Check role if allowedRoles is specified
      if (allowedRoles.length > 0 && !allowedRoles.includes(user?.role)) {
        // Redirect based on role
        if (user?.role === 'user') {
          router.push('/tickets');
        } else {
          router.push('/dashboard');
        }
      }
    }
  }, [isAuthenticated, user, loading, router, allowedRoles]);

  if (loading) {
    return <FullPageSpinner />;
  }

  if (!isAuthenticated) {
    return <FullPageSpinner />;
  }

  if (allowedRoles.length > 0 && !allowedRoles.includes(user?.role)) {
    return <FullPageSpinner />;
  }

  return children;
}
