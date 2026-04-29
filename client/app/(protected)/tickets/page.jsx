'use client';

import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import { useAuth } from '@/context/AuthContext';
import { getTickets } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import {
  StatusBadge,
  PriorityBadge,
  SLAWarning,
  AIInsightPreview,
  EmptyState,
  ErrorMessage,
  TicketPagination,
  TableSkeleton,
  CardSkeleton,
} from '@/components/sentinel';
import { Plus, Search, Ticket, TriangleAlert } from 'lucide-react';
import { format } from 'date-fns';
import { useIsMobile } from '@/hooks/use-mobile';

const DEFAULT_SORT = 'created_at:desc';

function RequiredSkillChips({ skills }) {
  if (!Array.isArray(skills) || skills.length === 0) {
    return null;
  }

  const visibleSkills = skills.slice(0, 3);
  const remainingCount = skills.length - visibleSkills.length;

  return (
    <div className="flex flex-wrap gap-1.5 pt-1">
      {visibleSkills.map((skill) => (
        <span key={skill} className="chip chip--neutral">
          {skill}
        </span>
      ))}
      {remainingCount > 0 ? (
        <span className="chip chip--neutral">+{remainingCount} more</span>
      ) : null}
    </div>
  );
}

export default function TicketsPage() {
  const { user } = useAuth();
  const isMobile = useIsMobile();
  const [tickets, setTickets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [ticketTotal, setTicketTotal] = useState(0);
  const [searchInput, setSearchInput] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [priorityFilter, setPriorityFilter] = useState('all');
  const [sortOption, setSortOption] = useState(DEFAULT_SORT);

  const fetchTickets = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [sortBy, sortDir] = sortOption.split(':');
      const params = {
        page,
        per_page: 10,
        sort_by: sortBy,
        sort_dir: sortDir,
      };

      if (searchQuery.trim()) params.q = searchQuery.trim();
      if (statusFilter && statusFilter !== 'all') params.status = statusFilter;
      if (priorityFilter && priorityFilter !== 'all') params.priority = priorityFilter;

      const response = await getTickets(params);
      setTickets(response.data || []);
      setTicketTotal(response.total || 0);
      setTotalPages(Math.ceil((response.total || 0) / (response.per_page || 10)));
    } catch (err) {
      setError(err.message || 'Failed to load tickets');
    } finally {
      setLoading(false);
    }
  }, [page, priorityFilter, searchQuery, sortOption, statusFilter]);

  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

  const applySearch = () => {
    setSearchQuery(searchInput.trim());
    setPage(1);
  };

  const resetFilters = () => {
    setSearchInput('');
    setSearchQuery('');
    setStatusFilter('all');
    setPriorityFilter('all');
    setSortOption(DEFAULT_SORT);
    setPage(1);
  };

  const activeFilterCount = [
    searchQuery.trim() !== '',
    statusFilter !== 'all',
    priorityFilter !== 'all',
    sortOption !== DEFAULT_SORT,
  ].filter(Boolean).length;

  const breachedVisibleCount = tickets.filter((ticket) => ticket.sla_breached).length;
  const resolvedVisibleCount = tickets.filter((ticket) =>
    ['resolved', 'closed'].includes(ticket.status),
  ).length;
  const openVisibleCount = tickets.filter((ticket) =>
    ['open', 'in_progress'].includes(ticket.status),
  ).length;

  const summaryCards = [
    {
      label: 'Filtered tickets',
      value: ticketTotal,
      description: 'Matching the current requester filters.',
    },
    {
      label: 'Active this page',
      value: openVisibleCount,
      description: 'Open or in-progress tickets on the current view.',
    },
    {
      label: 'SLA watch',
      value: breachedVisibleCount,
      description: 'Visible tickets currently flagged for SLA pressure.',
    },
    {
      label: 'Resolved visible',
      value: resolvedVisibleCount,
      description: 'Closed out or resolved tickets in this result set.',
    },
  ];

  return (
    <div className="page-shell">
      <header className="page-header">
        <div className="min-w-0">
          <span className="page-header__eyebrow">Requester workspace</span>
          <h1 className="page-header__title">
            Your support tickets, {user?.full_name?.split(' ')[0] || 'there'}
          </h1>
          <p className="page-header__description">
            Follow ticket progress, spot escalations early, and jump into a new request without
            losing the full context of what is already open.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="chip chip--neutral">
            <Ticket className="h-3.5 w-3.5" />
            {ticketTotal} ticket{ticketTotal === 1 ? '' : 's'} in scope
          </span>
          <Button asChild>
            <Link href="/tickets/new">
              <Plus className="mr-2 h-4 w-4" />
              New Ticket
            </Link>
          </Button>
        </div>
      </header>

      <div className="grid-equal grid-cols-1 sm:grid-cols-2 xl:grid-cols-4">
        {summaryCards.map((card) => (
          <div key={card.label} className="stat-card">
            <p className="stat-card__label">{card.label}</p>
            <p className="stat-card__value">{card.value}</p>
            <p className="stat-card__meta">{card.description}</p>
          </div>
        ))}
      </div>

      <ErrorMessage message={error} />

      <section className="data-card">
        <div className="data-card__header">
          <div className="min-w-0">
            <h2 className="section-title">Filter tickets</h2>
            <p className="section-subtitle">Refine your ticket list by status, priority, or sort order.</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="chip chip--neutral">
              {activeFilterCount} active filter{activeFilterCount === 1 ? '' : 's'}
            </span>
            <Button variant="outline" size="sm" onClick={resetFilters}>
              Reset
            </Button>
          </div>
        </div>
        <div className="data-card__body space-y-4">
          <div className="filter-bar">
            <div className="relative flex-1 min-w-0">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    applySearch();
                  }
                }}
                placeholder="Search by ticket, title, description, or skills"
                className="pl-9 w-full"
              />
            </div>
            <Button onClick={applySearch}>
              <Search className="mr-2 h-4 w-4" />
              Search
            </Button>
            {searchQuery ? (
              <Button
                variant="outline"
                onClick={() => {
                  setSearchInput('');
                  setSearchQuery('');
                  setPage(1);
                }}
              >
                Clear Search
              </Button>
            ) : null}
          </div>
          <div className="form-grid">
            <div className="space-y-1.5">
              <label className="form-label">Status</label>
              <Select
                value={statusFilter}
                onValueChange={(value) => {
                  setStatusFilter(value);
                  setPage(1);
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="All statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Statuses</SelectItem>
                  <SelectItem value="open">Open</SelectItem>
                  <SelectItem value="in_progress">In Progress</SelectItem>
                  <SelectItem value="resolved">Resolved</SelectItem>
                  <SelectItem value="closed">Closed</SelectItem>
                  <SelectItem value="cancelled">Cancelled</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="form-label">Priority</label>
              <Select
                value={priorityFilter}
                onValueChange={(value) => {
                  setPriorityFilter(value);
                  setPage(1);
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="All priorities" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Priorities</SelectItem>
                  <SelectItem value="critical">Critical</SelectItem>
                  <SelectItem value="high">High</SelectItem>
                  <SelectItem value="medium">Medium</SelectItem>
                  <SelectItem value="low">Low</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="form-label">Sort</label>
              <Select
                value={sortOption}
                onValueChange={(value) => {
                  setSortOption(value);
                  setPage(1);
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="Sort tickets" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ticket_number:asc">Ticket #</SelectItem>
                  <SelectItem value="title:asc">Title</SelectItem>
                  <SelectItem value="status:asc">Status</SelectItem>
                  <SelectItem value="priority:asc">Priority</SelectItem>
                  <SelectItem value="created_at:desc">Created</SelectItem>
                  <SelectItem value="updated_at:desc">Updated</SelectItem>
                  <SelectItem value="due_at:asc">Due At</SelectItem>
                  <SelectItem value="sla_breached:desc">SLA Breach</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
      </section>

      <section className="data-card">
        <div className="data-card__header">
          <div className="min-w-0">
            <h2 className="section-title">Ticket results</h2>
            <p className="section-subtitle">
              {searchQuery
                ? `${ticketTotal} ticket${ticketTotal === 1 ? '' : 's'} match "${searchQuery}" and your current filters.`
                : `${ticketTotal} ticket${ticketTotal === 1 ? '' : 's'} match your current filters.`}
            </p>
          </div>
          <span className="chip chip--warning">
            <TriangleAlert className="h-3.5 w-3.5" />
            {breachedVisibleCount} SLA flags on this page
          </span>
        </div>
        <div className="data-card__body space-y-4">
          {loading ? (
            isMobile ? <CardSkeleton count={3} /> : <TableSkeleton columns={5} rows={5} />
          ) : tickets.length === 0 ? (
            <EmptyState
              icon="tickets"
              title="No tickets found"
              description="You have no tickets in this view yet, or your current filters are hiding them."
              action={
                <Button asChild>
                  <Link href="/tickets/new">
                    <Plus className="mr-2 h-4 w-4" />
                    Create Ticket
                  </Link>
                </Button>
              }
            />
          ) : isMobile ? (
            <div className="grid-equal grid-cols-1 md:grid-cols-2">
              {tickets.map((ticket) => (
                <Link key={ticket.id} href={`/tickets/${ticket.id}`} className="block">
                  <div className="data-card data-card--interactive h-full">
                    <div className="data-card__header">
                      <div className="min-w-0">
                        <span className="font-mono text-xs text-muted-foreground">{ticket.ticket_number}</span>
                        <h3 className="mt-1 font-medium line-clamp-1">{ticket.title}</h3>
                      </div>
                      <SLAWarning breached={ticket.sla_breached} />
                    </div>
                    <div className="data-card__body space-y-3">
                      <AIInsightPreview insights={ticket.ai_insights} />
                      <RequiredSkillChips skills={ticket.required_skills} />
                      <div className="flex flex-wrap gap-2">
                        <StatusBadge status={ticket.status} />
                        <PriorityBadge priority={ticket.priority} />
                      </div>
                      <p className="text-sm text-muted-foreground">
                        {ticket.created_at ? format(new Date(ticket.created_at), 'MMM d, yyyy') : 'N/A'}
                      </p>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table className="data-table">
                <TableHeader>
                  <TableRow>
                    <TableHead>Ticket #</TableHead>
                    <TableHead>Title</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Priority</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tickets.map((ticket) => (
                    <TableRow key={ticket.id} className="cursor-pointer">
                      <TableCell>
                        <Link href={`/tickets/${ticket.id}`} className="flex items-center gap-2 font-medium min-w-0">
                          <span className="font-mono text-sm">{ticket.ticket_number}</span>
                          <SLAWarning breached={ticket.sla_breached} />
                        </Link>
                      </TableCell>
                      <TableCell>
                        <div className="space-y-1 min-w-0">
                          <Link href={`/tickets/${ticket.id}`} className="block line-clamp-1 hover:underline">
                            {ticket.title}
                          </Link>
                          <AIInsightPreview insights={ticket.ai_insights} />
                          <RequiredSkillChips skills={ticket.required_skills} />
                        </div>
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={ticket.status} />
                      </TableCell>
                      <TableCell>
                        <PriorityBadge priority={ticket.priority} />
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {ticket.created_at ? format(new Date(ticket.created_at), 'MMM d, yyyy') : 'N/A'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          <TicketPagination page={page} totalPages={totalPages} onPageChange={setPage} />
        </div>
      </section>
    </div>
  );
}
