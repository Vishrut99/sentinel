'use client';

import { Button } from '@/components/ui/button';
import { ChevronLeft, ChevronRight } from 'lucide-react';

export function TicketPagination({ page, totalPages, onPageChange }) {
  if (totalPages <= 1) return null;

  return (
    <div className="flex flex-col items-center justify-center gap-3 py-4 sm:flex-row">
      <Button
        variant="outline"
        size="sm"
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        className="rounded-full border-white/70 bg-white/82 px-4 shadow-[0_18px_40px_-34px_rgba(15,23,42,0.35)]"
      >
        <ChevronLeft className="h-4 w-4" />
        Previous
      </Button>
      
      <span className="rounded-full border border-white/70 bg-white/82 px-4 py-2 text-sm text-muted-foreground shadow-[0_18px_40px_-34px_rgba(15,23,42,0.35)]">
        Page {page} of {totalPages}
      </span>
      
      <Button
        variant="outline"
        size="sm"
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        className="rounded-full border-white/70 bg-white/82 px-4 shadow-[0_18px_40px_-34px_rgba(15,23,42,0.35)]"
      >
        Next
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  );
}
