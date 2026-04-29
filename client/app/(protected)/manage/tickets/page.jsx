'use client';

import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { getTickets, getCategories, getAgents } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Checkbox } from '@/components/ui/checkbox';
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
import {
  Filter,
  Plus,
  Search,
  ShieldAlert,
  Sparkles,
  Users,
} from 'lucide-react';
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

function AssignmentReason({ reason }) {
  if (typeof reason !== 'string' || !reason.trim()) {
    return null;
  }

  return <p className="line-clamp-2 text-xs leading-5 text-muted-foreground">{reason.trim()}</p>;
}

export default function ManageTicketsPage() {
  const { isUser, isAgent, isAdmin } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const isMobile = useIsMobile();

  const [tickets, setTickets] = useState([]);
  const [categories, setCategories] = useState([]);
  const [agents, setAgents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [ticketTotal, setTicketTotal] = useState(0);

  const [statusFilter, setStatusFilter] = useState(searchParams.get('status') || 'all');
  const [priorityFilter, setPriorityFilter] = useState(searchParams.get('priority') || 'all');
  const [categoryFilter, setCategoryFilter] = useState(searchParams.get('category_id') || 'all');
  const [searchInput, setSearchInput] = useState(searchParams.get('q') || '');
  const [searchQuery, setSearchQuery] = useState(searchParams.get('q') || '');
  const [assignedAgentFilter, setAssignedAgentFilter] = useState(
    searchParams.get('assigned_agent_id') || 'all',
  );
  const [slaBreachedFilter, setSlaBreachedFilter] = useState(
    searchParams.get('sla_breached') === 'true',
  );
  const [sortOption, setSortOption] = useState(searchParams.get('sort') || DEFAULT_SORT);

  useEffect(() => {
    if (isUser) {
      router.push('/tickets');
    }
  }, [isUser, router]);

  useEffect(() => {
    const loadFilters = async () => {
      try {
        const [categoriesData, agentsData] = await Promise.all([
          getCategories(),
          isAdmin ? getAgents().catch(() => []) : Promise.resolve([]),
        ]);
        setCategories(Array.isArray(categoriesData) ? categoriesData : categoriesData?.data || []);
        setAgents(Array.isArray(agentsData) ? agentsData : agentsData?.data || []);
      } catch (err) {
        console.error('Failed to load filters:', err);
      }
    };

    loadFilters();
  }, [isAdmin]);

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
      if (categoryFilter && categoryFilter !== 'all') params.category_id = categoryFilter;
      if (isAdmin && assignedAgentFilter && assignedAgentFilter !== 'all') {
        params.assigned_agent_id = assignedAgentFilter;
      }
      if (slaBreachedFilter) params.sla_breached = true;

      const response = await getTickets(params);
      setTickets(response.data || []);
      setTicketTotal(response.total || 0);
      setTotalPages(Math.ceil((response.total || 0) / (response.per_page || 10)));
    } catch (err) {
      setError(err.message || 'Failed to load tickets');
    } finally {
      setLoading(false);
    }
  }, [
    assignedAgentFilter,
    categoryFilter,
    isAdmin,
    page,
    priorityFilter,
    searchQuery,
    slaBreachedFilter,
    sortOption,
    statusFilter,
  ]);

  const applySearch = () => {
    setSearchQuery(searchInput.trim());
    setPage(1);
  };

  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

  const resetFilters = () => {
    setSearchInput('');
    setSearchQuery('');
    setStatusFilter('all');
    setPriorityFilter('all');
    setCategoryFilter('all');
    setAssignedAgentFilter('all');
    setSlaBreachedFilter(false);
    setSortOption(DEFAULT_SORT);
    setPage(1);
  };

  const activeFilterCount = [
    searchQuery.trim() !== '',
    statusFilter !== 'all',
    priorityFilter !== 'all',
    categoryFilter !== 'all',
    isAdmin && assignedAgentFilter !== 'all',
    slaBreachedFilter,
    sortOption !== DEFAULT_SORT,
  ].filter(Boolean).length;

  const breachedVisibleCount = tickets.filter((ticket) => ticket.sla_breached).length;
  const unassignedVisibleCount = tickets.filter((ticket) => !ticket.assigned_agent?.full_name).length;

  const pageTitle = isAgent ? 'Assigned Tickets' : 'All Tickets';
  const pageDescription = isAgent
    ? 'Track the tickets currently assigned to you and keep work moving before breaches build.'
    : 'Monitor the service desk queue, assignment health, and incident pressure across the board.';
  const emptyDescription = isAgent
    ? 'No assigned tickets match your current filters.'
    : 'No tickets match your current filters.';

  const summaryCards = [
    {
      label: 'Filtered tickets',
      value: ticketTotal,
      description: 'Visible within the current queue scope.',
    },
    {
      label: 'SLA watch',
      value: breachedVisibleCount,
      description: 'Flagged tickets on the current result page.',
    },
    {
      label: 'Unassigned visible',
      value: unassignedVisibleCount,
      description: isAdmin ? 'Tickets still waiting for ownership.' : 'Tickets not yet routed.',
    },
    {
      label: 'Categories loaded',
      value: categories.length,
      description: isAdmin ? `${agents.length} agents available in filters.` : 'Available incident categories.',
    },
  ];

  return (
    <div className="page-shell">
      <header className="page-header">
        <div className="page-header__eyebrow">
          <Sparkles className="h-3 w-3" />
          Operations queue
        </div>
        <h1 className="page-header__title">{pageTitle}</h1>
        <p className="page-header__description">{pageDescription}</p>
        <div className="flex flex-wrap gap-2.5">
          <Button asChild size="sm">
            <Link href="/tickets/new">
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              New Ticket
            </Link>
          </Button>
          <span className="chip chip--neutral">
            <Users className="h-3.5 w-3.5" />
            {isAdmin ? agents.length : ticketTotal} {isAdmin ? 'agents in scope' : 'tickets in scope'}
          </span>
        </div>
      </header>

      <div className="grid grid-equal grid-cols-2 md:grid-cols-4">
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
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 className="data-card__title">Queue filters</h2>
              <p className="data-card__subtitle">Refine the operational view by status, ownership, category, and breach risk.</p>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <span className="chip chip--neutral">
                <Filter className="h-3.5 w-3.5" />
                {activeFilterCount} active {activeFilterCount === 1 ? 'filter' : 'filters'}
              </span>
              <Button variant="outline" size="sm" onClick={resetFilters}>
                Reset
              </Button>
            </div>
          </div>
        </div>
        <div className="data-card__body">
          <div className="filter-bar mb-4">
            <div className="relative flex-1">
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
                className="pl-9"
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
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-6">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Status</label>
              <Select
                value={statusFilter}
                onValueChange={(value) => {
                  setStatusFilter(value);
                  setPage(1);
                }}
              >
                <SelectTrigger>
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
              <label className="text-sm font-medium">Priority</label>
              <Select
                value={priorityFilter}
                onValueChange={(value) => {
                  setPriorityFilter(value);
                  setPage(1);
                }}
              >
                <SelectTrigger>
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
              <label className="text-sm font-medium">Category</label>
              <Select
                value={categoryFilter}
                onValueChange={(value) => {
                  setCategoryFilter(value);
                  setPage(1);
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="All categories" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Categories</SelectItem>
                  {categories.map((cat) => (
                    <SelectItem key={cat.id} value={cat.id}>
                      {cat.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {isAdmin ? (
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Assigned</label>
                <Select
                  value={assignedAgentFilter}
                  onValueChange={(value) => {
                    setAssignedAgentFilter(value);
                    setPage(1);
                  }}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All assignees" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Assignees</SelectItem>
                    {agents.map((agent) => (
                      <SelectItem key={agent.id} value={agent.id}>
                        {agent.full_name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}

            <div className="space-y-1.5">
              <label className="text-sm font-medium">Sort</label>
              <Select
                value={sortOption}
                onValueChange={(value) => {
                  setSortOption(value);
                  setPage(1);
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Sort by" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ticket_number:asc">Ticket #</SelectItem>
                  <SelectItem value="title:asc">Title</SelectItem>
                  <SelectItem value="status:asc">Status</SelectItem>
                  <SelectItem value="priority:asc">Priority</SelectItem>
                  <SelectItem value="assigned_to:asc">Assigned To</SelectItem>
                  <SelectItem value="created_at:desc">Created</SelectItem>
                  <SelectItem value="updated_at:desc">Updated</SelectItem>
                  <SelectItem value="due_at:asc">Due At</SelectItem>
                  <SelectItem value="category:asc">Category</SelectItem>
                  <SelectItem value="sla_breached:desc">SLA Breach</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <label className="text-sm font-medium">Escalation</label>
              <div className="flex h-9 items-center gap-3 rounded-md border border-input bg-background px-3">
                <Checkbox
                  id="sla-breached"
                  checked={slaBreachedFilter}
                  onCheckedChange={(checked) => {
                    setSlaBreachedFilter(Boolean(checked));
                    setPage(1);
                  }}
                />
                <label
                  htmlFor="sla-breached"
                  className="cursor-pointer text-sm font-medium leading-none"
                >
                  SLA breached only
                </label>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="data-card">
        <div className="data-card__header">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="data-card__title">Queue results</h2>
              <p className="data-card__subtitle">
                {searchQuery
                  ? `${ticketTotal} ticket${ticketTotal === 1 ? '' : 's'} match "${searchQuery}" and the current operational filters.`
                  : `${ticketTotal} ticket${ticketTotal === 1 ? '' : 's'} match the current operational filters.`}
              </p>
            </div>
            <span className="chip chip--warning">
              <ShieldAlert className="h-3.5 w-3.5" />
              {breachedVisibleCount} SLA flags on this page
            </span>
          </div>
        </div>
        <div className="data-card__body space-y-6">
          {loading ? (
            isMobile ? <CardSkeleton count={3} /> : <TableSkeleton columns={6} rows={5} />
          ) : tickets.length === 0 ? (
            <EmptyState
              icon="tickets"
              title="No tickets found"
              description={emptyDescription}
              action={
                <Button variant="outline" onClick={resetFilters}>
                  Reset Filters
                </Button>
              }
            />
          ) : isMobile ? (
            <div className="space-y-4">
              {tickets.map((ticket) => (
                <Link key={ticket.id} href={`/tickets/${ticket.id}`} className="block">
                  <div className="data-card data-card--interactive">
                    <div className="data-card__body space-y-3">
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <span className="font-mono text-xs text-muted-foreground">{ticket.ticket_number}</span>
                          <h3 className="mt-1 font-medium line-clamp-1">{ticket.title}</h3>
                        </div>
                        <SLAWarning breached={ticket.sla_breached} />
                      </div>
                      <AIInsightPreview insights={ticket.ai_insights} />
                      <RequiredSkillChips skills={ticket.required_skills} />
                      <AssignmentReason reason={ticket.assignment_justification} />
                      <div className="flex flex-wrap gap-2">
                        <StatusBadge status={ticket.status} />
                        <PriorityBadge priority={ticket.priority} />
                      </div>
                      <div className="flex justify-between gap-3 text-sm text-muted-foreground">
                        <span className="line-clamp-1">
                          {ticket.assigned_agent?.full_name ||
                            ticket.assigned_agent?.user?.full_name ||
                            'Unassigned'}
                        </span>
                        <span>{ticket.created_at ? format(new Date(ticket.created_at), 'MMM d') : ''}</span>
                      </div>
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
                    <TableHead>Assigned To</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tickets.map((ticket) => (
                    <TableRow key={ticket.id} className="cursor-pointer">
                      <TableCell>
                        <Link href={`/tickets/${ticket.id}`} className="flex items-center gap-2 font-medium">
                          <span className="font-mono text-sm">{ticket.ticket_number}</span>
                          <SLAWarning breached={ticket.sla_breached} />
                        </Link>
                      </TableCell>
                      <TableCell>
                        <div className="space-y-1">
                          <Link href={`/tickets/${ticket.id}`} className="block line-clamp-1 hover:underline">
                            {ticket.title}
                          </Link>
                          <AIInsightPreview insights={ticket.ai_insights} />
                          <RequiredSkillChips skills={ticket.required_skills} />
                          <AssignmentReason reason={ticket.assignment_justification} />
                        </div>
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={ticket.status} />
                      </TableCell>
                      <TableCell>
                        <PriorityBadge priority={ticket.priority} />
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {ticket.assigned_agent?.full_name ||
                          ticket.assigned_agent?.user?.full_name ||
                          'Unassigned'}
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
