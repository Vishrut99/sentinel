'use client';

import { Spinner } from '@/components/ui/spinner';

export function LoadingSpinner({ className = '', size = 'default' }) {
  return (
    <div className={`flex items-center justify-center p-8 ${className}`}>
      <Spinner size={size} />
    </div>
  );
}

export function FullPageSpinner() {
  return (
    <div className="flex items-center justify-center min-h-screen">
      <Spinner size="lg" />
    </div>
  );
}
