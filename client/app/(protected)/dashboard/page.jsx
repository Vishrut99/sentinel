'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { getDashboard, getTickets, getCategories, getAgents } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { FullPageSpinner, ErrorMessage, StatusBadge, PriorityBadge, SLAWarning, AIInsightPreview } from '@/components/sentinel';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from 'recharts';
import {
  Ticket,
  Clock,
  CheckCircle,
  AlertTriangle,
  ListTodo,
  Settings,
  TrendingUp,
  ArrowRight,
  SlidersHorizontal,
} from 'lucide-react';

const PRIORITY_COLORS = {
  critical: '#ef4444',
  high: '#f97316',
  medium: '#eab308',
  low: '#6b7280',
};

const CHART_COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#06b6d4'];
const DEFAULT_FILTERS = {
  q: '',
  status: 'all',
  priority: 'all',
  categoryId: 'all',
  assignedAgentId: 'all',
  slaBreached: false,
  isProblem: false,
  sort: 'updated_at:desc',
};

export default function DashboardPage() {
  const { user, isUser, isAdmin, isAgent } = useAuth();
  const router = useRouter();
  const [stats, setStats] = useState(null);
  const [categories, setCategories] = useState([]);
  const [agents, setAgents] = useState([]);
  const [tickets, setTickets] = useState([]);
  const [loading, setLoading] = useState(true);
  const [explorerLoading, setExplorerLoading] = useState(true);
  const [error, setError] = useState('');
  const ticketListingLabel = isAgent ? 'Assigned Tickets' : 'All Tickets';
  const [filters, setFilters] = useState(DEFAULT_FILTERS);

  useEffect(() => {
    // Redirect users away from dashboard
    if (isUser) {
      router.push('/tickets');
      return;
    }

    const loadLookups = async () => {
      try {
        const [categoriesData, agentsData] = await Promise.all([
          getCategories().catch(() => []),
          isAdmin ? getAgents().catch(() => []) : Promise.resolve([]),
        ]);
        setCategories(Array.isArray(categoriesData) ? categoriesData : categoriesData?.data || []);
        setAgents(Array.isArray(agentsData) ? agentsData : agentsData?.data || []);
      } catch (err) {
        setError(err.message || 'Failed to load dashboard filters');
      }
    };

    loadLookups();
  }, [isAdmin, isUser, router]);

  useEffect(() => {
    if (isUser) {
      return;
    }

    const fetchDashboardData = async () => {
      const [sortBy, sortDir] = filters.sort.split(':');
      const params = {
        q: filters.q || undefined,
        status: filters.status !== 'all' ? filters.status : undefined,
        priority: filters.priority !== 'all' ? filters.priority : undefined,
        category_id: filters.categoryId !== 'all' ? filters.categoryId : undefined,
        assigned_agent_id: isAdmin && filters.assignedAgentId !== 'all' ? filters.assignedAgentId : undefined,
        sla_breached: filters.slaBreached ? true : undefined,
        is_problem: filters.isProblem ? true : undefined,
      };

      if (stats == null) {
        setLoading(true);
      }
      setExplorerLoading(true);
      try {
        const [dashboardResponse, ticketsResponse] = await Promise.all([
          getDashboard(params),
          getTickets({
            ...params,
            sort_by: sortBy,
            sort_dir: sortDir,
            page: 1,
            per_page: 8,
          }),
        ]);
        setStats(dashboardResponse);
        setTickets(Array.isArray(ticketsResponse?.data) ? ticketsResponse.data : []);
      } catch (err) {
        setError(err.message || 'Failed to load dashboard data');
      } finally {
        setLoading(false);
        setExplorerLoading(false);
      }
    };

    fetchDashboardData();
  }, [filters, isAdmin, isUser]);

  if (loading) {
    return <FullPageSpinner />;
  }

  // Format priority data for chart
  const priorityData = stats?.by_priority
    ? Object.entries(stats.by_priority).map(([name, value]) => ({
        name: name.charAt(0).toUpperCase() + name.slice(1),
        value,
        fill: PRIORITY_COLORS[name] || '#6b7280',
      }))
    : [];

  // Format category data for chart
  const categoryData = stats?.by_category
    ? Object.entries(stats.by_category).map(([name, value], index) => ({
        name,
        value,
        fill: CHART_COLORS[index % CHART_COLORS.length],
      }))
    : [];

  const updateFilter = (key, value) => {
    setFilters((current) => ({
      ...current,
      [key]: value,
    }));
  };

  const resetFilters = () => {
    setFilters(DEFAULT_FILTERS);
  };

  const spotlightItems = [
    {
      label: isAgent ? 'Assigned queue' : 'Open queue',
      value: stats?.open_tickets || 0,
      description: isAgent ? 'Tickets currently in your workflow.' : 'Tickets needing active attention.',
    },
    {
      label: 'Resolved today',
      value: stats?.resolved_today || 0,
      description: 'Closed out in the current reporting window.',
    },
    {
      label: 'SLA risk',
      value: stats?.sla_breached || 0,
      description: 'Breached tickets that need a response plan.',
    },
  ];

  const summaryCards = [
    {
      title: 'Total tickets',
      value: stats?.total_tickets || 0,
      description: 'Across the current dashboard scope',
      icon: Ticket,
    },
    {
      title: 'Open',
      value: stats?.open_tickets || 0,
      description: 'Waiting for progress or assignment',
      icon: Ticket,
    },
    {
      title: 'In progress',
      value: stats?.in_progress_tickets || 0,
      description: 'Currently being worked by the team',
      icon: Clock,
    },
    {
      title: 'Resolved today',
      value: stats?.resolved_today || 0,
      description: 'Closed during the active day view',
      icon: CheckCircle,
    },
    {
      title: 'SLA breached',
      value: stats?.sla_breached || 0,
      description: 'Requires immediate recovery attention',
      icon: AlertTriangle,
    },
    {
      title: 'Avg. resolve',
      value: stats?.avg_resolve_hours ? `${stats.avg_resolve_hours.toFixed(1)}h` : '0h',
      description: 'Average time to resolution',
      icon: TrendingUp,
    },
  ];

  const quickActions = [
    {
      href: '/manage/tickets?status=open',
      title: isAgent ? 'Review active queue' : 'View open tickets',
      description: `${stats?.open_tickets || 0} tickets need attention`,
      icon: Ticket,
    },
    {
      href: '/manage/tickets?sla_breached=true',
      title: 'Recover breached tickets',
      description: `${stats?.sla_breached || 0} tickets are past SLA`,
      icon: AlertTriangle,
    },
    {
      href: '/tickets/new',
      title: 'Create a ticket',
      description: 'Start a new support request fast',
      icon: ListTodo,
    },
  ];

  if (isAdmin) {
    quickActions.push({
      href: '/admin/settings',
      title: 'Open settings',
      description: 'Manage categories, users, and routing',
      icon: Settings,
    });
  }

  const activeFilterCount = [
    filters.q.trim() !== '',
    filters.status !== 'all',
    filters.priority !== 'all',
    filters.categoryId !== 'all',
    isAdmin && filters.assignedAgentId !== 'all',
    filters.slaBreached,
    filters.isProblem,
    filters.sort !== DEFAULT_FILTERS.sort,
  ].filter(Boolean).length;

  const dashboardDate = new Intl.DateTimeFormat('en-US', {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
  }).format(new Date());

  return (
    <div className="page-shell">
      <header className="page-header">
        <div className="min-w-0">
          <div className="page-header__eyebrow">Operations overview</div>
          <h1 className="page-header__title">Welcome back, {user?.full_name}</h1>
          <p className="page-header__description">
            {isAgent
              ? 'Stay on top of your assigned incidents, watch SLA pressure, and move the queue forward with better context.'
              : 'Monitor workload, resolution pace, and breach risk across the service desk from one command view.'}
          </p>
          <div className="mt-3 flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
            <span>{dashboardDate}</span>
            <span>•</span>
            <span>{categories.length} categories loaded</span>
            {isAdmin ? (
              <>
                <span>•</span>
                <span>{agents.length} agents visible</span>
              </>
            ) : null}
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button asChild>
            <Link href="/manage/tickets">
              <ListTodo className="h-4 w-4 mr-2" />
              {ticketListingLabel}
            </Link>
          </Button>
          {isAdmin ? (
            <Button variant="outline" asChild>
              <Link href="/admin/settings">
                <Settings className="h-4 w-4 mr-2" />
                Settings
              </Link>
            </Button>
          ) : null}
        </div>
      </header>

      <div className="grid grid-equal grid-cols-1 sm:grid-cols-3">
        {spotlightItems.map((item) => (
          <div key={item.label} className="stat-card">
            <p className="stat-card__label">{item.label}</p>
            <p className="stat-card__value">{item.value}</p>
            <p className="stat-card__meta">{item.description}</p>
          </div>
        ))}
      </div>

      <ErrorMessage message={error} />

      <div className="grid grid-equal grid-cols-2 md:grid-cols-3 xl:grid-cols-6">
        {summaryCards.map((card) => (
          <div key={card.title} className="stat-card">
            <div className="flex items-start justify-between gap-2">
              <p className="stat-card__label">{card.title}</p>
              <card.icon className="h-4 w-4 text-muted-foreground" />
            </div>
            <p className="stat-card__value">{card.value}</p>
            <p className="stat-card__meta">{card.description}</p>
          </div>
        ))}
      </div>

      <div className="row-equal">
        <div className="data-card data-card--lg">
          <div className="data-card__header">
            <div className="min-w-0">
              <div className="section-title">Tickets by Priority</div>
              <div className="section-subtitle">Distribution of tickets across priority levels</div>
            </div>
          </div>
          <div className="data-card__body">
            {priorityData.length === 0 ? (
              <p className="text-muted-foreground text-center py-8 text-sm">No data available</p>
            ) : (
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={priorityData} layout="vertical">
                  <XAxis type="number" />
                  <YAxis dataKey="name" type="category" width={80} />
                  <Tooltip
                    contentStyle={{
                      backgroundColor: 'var(--card)',
                      border: '1px solid var(--border)',
                      borderRadius: 'var(--radius)',
                      boxShadow: 'var(--shadow-md)',
                    }}
                  />
                  <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                    {priorityData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.fill} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>

        <div className="data-card data-card--lg">
          <div className="data-card__header">
            <div className="min-w-0">
              <div className="section-title">Tickets by Category</div>
              <div className="section-subtitle">Distribution of tickets across categories</div>
            </div>
          </div>
          <div className="data-card__body">
            {categoryData.length === 0 ? (
              <p className="text-muted-foreground text-center py-8 text-sm">No data available</p>
            ) : (
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={categoryData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={100}
                    paddingAngle={2}
                    dataKey="value"
                    label={({ name, percent }) => `${name} (${(percent * 100).toFixed(0)}%)`}
                    labelLine={false}
                  >
                    {categoryData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.fill} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{
                      backgroundColor: 'var(--card)',
                      border: '1px solid var(--border)',
                      borderRadius: 'var(--radius)',
                      boxShadow: 'var(--shadow-md)',
                    }}
                  />
                </PieChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>
      </div>

      <div className="data-card data-card--lg">
        <div className="data-card__header">
          <div className="min-w-0">
            <div className="section-title">Quick Actions</div>
            <div className="section-subtitle">Jump into the queue, recover risk, or move straight into intake.</div>
          </div>
        </div>
        <div className="data-card__body">
          <div className="grid grid-equal grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
            {quickActions.map((action) => (
              <Link
                key={action.href}
                href={action.href}
                className="data-card data-card--interactive"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-start gap-3 min-w-0">
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
                      <action.icon className="h-5 w-5" />
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-foreground">{action.title}</p>
                      <p className="mt-1 text-xs text-muted-foreground leading-5">{action.description}</p>
                    </div>
                  </div>
                  <ArrowRight className="mt-1 h-4 w-4 text-muted-foreground shrink-0" />
                </div>
              </Link>
            ))}
          </div>
        </div>
      </div>

      <div className="data-card data-card--lg">
        <div className="data-card__header flex-col lg:flex-row lg:items-center">
          <div className="min-w-0">
            <div className="section-title">Ticket Explorer</div>
            <div className="section-subtitle">Search, filter, and sort tickets directly from the dashboard.</div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="chip chip--neutral">
              <SlidersHorizontal className="h-3 w-3" />
              {activeFilterCount} active filter{activeFilterCount === 1 ? '' : 's'}
            </span>
            <Button variant="outline" size="sm" onClick={resetFilters}>
              Reset filters
            </Button>
          </div>
        </div>

        <div className="data-card__body space-y-6">
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <div className="space-y-2 xl:col-span-2 min-w-0">
              <label className="form-label">Search</label>
              <Input
                placeholder="Search by title or ticket number"
                value={filters.q}
                onChange={(e) => updateFilter('q', e.target.value)}
              />
            </div>
            <div className="space-y-2 min-w-0">
              <label className="form-label">Status</label>
              <Select value={filters.status} onValueChange={(value) => updateFilter('status', value)}>
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
            <div className="space-y-2 min-w-0">
              <label className="form-label">Priority</label>
              <Select value={filters.priority} onValueChange={(value) => updateFilter('priority', value)}>
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
            <div className="space-y-2 min-w-0">
              <label className="form-label">Category</label>
              <Select value={filters.categoryId} onValueChange={(value) => updateFilter('categoryId', value)}>
                <SelectTrigger>
                  <SelectValue placeholder="All categories" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Categories</SelectItem>
                  {categories.map((category) => (
                    <SelectItem key={category.id} value={category.id}>
                      {category.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {isAdmin && (
              <div className="space-y-2 min-w-0">
                <label className="form-label">Assigned Agent</label>
                <Select value={filters.assignedAgentId} onValueChange={(value) => updateFilter('assignedAgentId', value)}>
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
            )}
            <div className="space-y-2 min-w-0">
              <label className="form-label">Sort</label>
              <Select value={filters.sort} onValueChange={(value) => updateFilter('sort', value)}>
                <SelectTrigger>
                  <SelectValue placeholder="Sort tickets" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ticket_number:asc">Ticket #</SelectItem>
                  <SelectItem value="updated_at:desc">Recently Updated</SelectItem>
                  <SelectItem value="created_at:desc">Newest Created</SelectItem>
                  <SelectItem value="due_at:asc">Due Soonest</SelectItem>
                  <SelectItem value="priority:asc">Highest Priority</SelectItem>
                  <SelectItem value="status:asc">Status</SelectItem>
                  <SelectItem value="category:asc">Category</SelectItem>
                  <SelectItem value="assigned_to:asc">Assigned Agent</SelectItem>
                  <SelectItem value="sla_breached:desc">SLA Breach</SelectItem>
                  <SelectItem value="title:asc">Title A-Z</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-2 pt-8">
              <Checkbox
                id="dashboard-sla"
                checked={filters.slaBreached}
                onCheckedChange={(checked) => updateFilter('slaBreached', Boolean(checked))}
              />
              <label htmlFor="dashboard-sla" className="text-sm leading-none">
                SLA breached only
              </label>
            </div>
            <div className="flex items-center gap-2 pt-8">
              <Checkbox
                id="dashboard-problem"
                checked={filters.isProblem}
                onCheckedChange={(checked) => updateFilter('isProblem', Boolean(checked))}
              />
              <label htmlFor="dashboard-problem" className="text-sm leading-none">
                Problem tickets only
              </label>
            </div>
          </div>

          {explorerLoading ? (
            <p className="text-sm text-muted-foreground">Loading filtered tickets...</p>
          ) : tickets.length === 0 ? (
            <p className="text-sm text-muted-foreground">No tickets match the current dashboard filters.</p>
          ) : (
            <div className="grid grid-equal grid-cols-1 lg:grid-cols-2">
              {tickets.map((ticket) => (
                <Link
                  key={ticket.id}
                  href={`/tickets/${ticket.id}`}
                  className="data-card data-card--interactive"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-mono text-xs text-muted-foreground">{ticket.ticket_number}</p>
                      <h3 className="line-clamp-1 text-sm font-medium text-foreground">{ticket.title}</h3>
                    </div>
                    <SLAWarning breached={ticket.sla_breached} />
                  </div>
                  <div className="flex flex-wrap gap-2 mt-3">
                    <StatusBadge status={ticket.status} />
                    <PriorityBadge priority={ticket.priority} />
                  </div>
                  <p className="text-xs text-muted-foreground mt-3">
                    {ticket.assigned_agent?.full_name || 'Unassigned'} • {ticket.category}
                  </p>
                  <div className="mt-3">
                    <AIInsightPreview insights={ticket.ai_insights} />
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
