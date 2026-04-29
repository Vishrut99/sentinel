'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { FullPageSpinner } from '@/components/sentinel';

export default function HomePage() {
  const { isAuthenticated, user, loading } = useAuth() as {
    isAuthenticated: boolean;
    user: { role?: string } | null;
    loading: boolean;
  };
  const router = useRouter();

  useEffect(() => {
    if (!loading) {
      if (!isAuthenticated) {
        router.push('/auth');
      } else if (user?.role === 'user') {
        router.push('/tickets');
      } else {
        router.push('/dashboard');
      }
    }
  }, [isAuthenticated, user, loading, router]);

  return <FullPageSpinner />;
}
