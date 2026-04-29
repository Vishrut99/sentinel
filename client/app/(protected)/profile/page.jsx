'use client';

import Link from 'next/link';
import { useEffect, useMemo, useState } from 'react';
import { format } from 'date-fns';
import { useAuth } from '@/context/AuthContext';
import { getDashboard, getTickets } from '@/lib/api';
import {
  ErrorMessage,
  FullPageSpinner,
  PriorityBadge,
  SLAWarning,
  StatusBadge,
} from '@/components/sentinel';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import {
  ArrowRight,
  BriefcaseBusiness,
  Mail,
  ShieldCheck,
  Ticket,
  UserRound,
  Workflow,
} from 'lucide-react';

function getRoleMeta(role) {
  if (role === 'admin') {
    return {
      label: 'Admin control',
      summary: 'You can oversee queue health, tune routing, and manage operational settings.',
      scopeLabel: 'Visible queue',
      activeLabel: 'Active queue',
      resolvedLabel: 'Resolved today',
      capabilities: [
        'Oversee the global incident queue and assignment health.',
        'Tune staffing, capacity, and skill-driven routing in settings.',
        'Link incidents to problems and keep the operating picture clean.',
      ],
      quickLinks: [
        { href: '/dashboard', label: 'Dashboard' },
        { href: '/manage/tickets', label: 'Operations Queue' },
        { href: '/admin/settings', label: 'Settings' },
      ],
      recentLabel: 'Recently updated tickets in your admin scope',
    };
  }

  if (role === 'agent') {
    return {
      label: 'Agent lane',
      summary: 'You are working the assigned queue and keeping active tickets moving.',
      scopeLabel: 'Assigned scope',
      activeLabel: 'Active on you',
      resolvedLabel: 'Resolved today',
      capabilities: [
        'Progress assigned work and update ticket status cleanly.',
        'Keep comments, ownership, and linked problem context up to date.',
        'Use the dashboard to spot SLA pressure before it grows.',
      ],
      quickLinks: [
        { href: '/dashboard', label: 'Dashboard' },
        { href: '/manage/tickets', label: 'Assigned Tickets' },
        { href: '/tickets/new', label: 'Create Ticket' },
      ],
      recentLabel: 'Recently updated tickets assigned to you',
    };
  }

  return {
    label: 'Requester lane',
    summary: 'You can track your requests, add new tickets, and keep the conversation moving.',
    scopeLabel: 'Your tickets',
    activeLabel: 'Active requests',
    resolvedLabel: 'Resolved or closed',
    capabilities: [
      'Create new requests with clearer routing context.',
      'Track status changes and recent updates across your tickets.',
      'Return to ticket workspaces quickly when follow-up is needed.',
    ],
    quickLinks: [
      { href: '/tickets', label: 'My Tickets' },
      { href: '/tickets/new', label: 'Create Ticket' },
    ],
    recentLabel: 'Your recently updated requests',
  };
}

function buildUserStats(response) {
  const items = Array.isArray(response?.data) ? response.data : [];

  return {
    total: response?.total || items.length,
    active: items.filter((ticket) => ['open', 'in_progress'].includes(ticket.status)).length,
    resolved: items.filter((ticket) => ['resolved', 'closed'].includes(ticket.status)).length,
    sla: items.filter((ticket) => ticket.sla_breached).length,
  };
}

export default function ProfilePage() {
  const { user, isAdmin, isAgent } = useAuth();
  const roleMeta = getRoleMeta(user?.role);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [summary, setSummary] = useState({
    total: 0,
    active: 0,
    resolved: 0,
    sla: 0,
  });
  const [recentTickets, setRecentTickets] = useState([]);

  const initials = user?.full_name
    ? user.full_name
        .split(' ')
        .slice(0, 2)
        .map((part) => part[0]?.toUpperCase())
        .join('')
    : 'SC';

  useEffect(() => {
    let ignore = false;

    const loadProfileOverview = async (isInitial) => {
      if (isInitial) setLoading(true);
      setError('');

      try {
        if (isAdmin || isAgent) {
          const [dashboardData, recentResponse] = await Promise.all([
            getDashboard(),
            getTickets({
              sort_by: 'updated_at',
              sort_dir: 'desc',
              page: 1,
              per_page: 6,
            }),
          ]);

          if (ignore) return;

          setSummary({
            total: dashboardData?.total_tickets || 0,
            active: (dashboardData?.open_tickets || 0) + (dashboardData?.in_progress_tickets || 0),
            resolved: dashboardData?.resolved_today || 0,
            sla: dashboardData?.sla_breached || 0,
          });
          setRecentTickets(Array.isArray(recentResponse?.data) ? recentResponse.data : []);
        } else {
          const ticketResponse = await getTickets({
            sort_by: 'updated_at',
            sort_dir: 'desc',
            page: 1,
            per_page: 100,
          });

          if (ignore) return;

          setSummary(buildUserStats(ticketResponse));
          setRecentTickets(Array.isArray(ticketResponse?.data) ? ticketResponse.data.slice(0, 6) : []);
        }
      } catch (err) {
        if (!ignore && isInitial) {
          setError(err.message || 'Failed to load profile overview');
        }
      } finally {
        if (!ignore && isInitial) setLoading(false);
      }
    };

    loadProfileOverview(true);
    const intervalId = window.setInterval(() => loadProfileOverview(false), 30000);

    return () => {
      ignore = true;
      window.clearInterval(intervalId);
    };
  }, [isAdmin, isAgent]);

  const summaryCards = useMemo(
    () => [
      {
        label: roleMeta.scopeLabel,
        value: summary.total,
        description: 'Tickets currently visible in your working scope.',
      },
      {
        label: roleMeta.activeLabel,
        value: summary.active,
        description: 'Open or in-progress work that still needs attention.',
      },
      {
        label: roleMeta.resolvedLabel,
        value: summary.resolved,
        description: isAdmin || isAgent ? 'Closed out in today’s active reporting window.' : 'Requests that have already reached completion.',
      },
      {
        label: 'SLA watch',
        value: summary.sla,
        description: 'Tickets currently flagged for breach pressure.',
      },
    ],
    [isAdmin, isAgent, roleMeta, summary],
  );

  if (loading) {
    return <FullPageSpinner />;
  }

  return (
    <div className="page-shell">
      <header className="page-header">
        <div className="min-w-0">
          <div className="page-header__eyebrow">Profile overview</div>
          <h1 className="page-header__title">{user?.full_name}</h1>
          <p className="page-header__description">{roleMeta.summary}</p>
        </div>
        <div className="flex items-center gap-3 min-w-0">
          <Avatar className="h-14 w-14">
            <AvatarFallback className="text-lg font-semibold text-primary">
              {initials}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Mail className="h-4 w-4 text-primary" />
              <span className="truncate">{user?.email}</span>
            </div>
            <div className="mt-2 flex flex-wrap gap-2">
              <span className="chip chip--info">{roleMeta.label}</span>
              <span className="chip chip--neutral">Logged in</span>
            </div>
          </div>
        </div>
      </header>

      <ErrorMessage message={error} />

      <div className="grid-equal grid-cols-2 md:grid-cols-4">
        {summaryCards.map((card) => (
          <div key={card.label} className="stat-card">
            <p className="stat-card__label">{card.label}</p>
            <p className="stat-card__value">{card.value}</p>
            <p className="stat-card__meta">{card.description}</p>
          </div>
        ))}
      </div>

      <div className="grid-equal grid-cols-1 xl:grid-cols-[0.85fr_1.15fr]">
        <div className="flex flex-col gap-6 min-w-0">
          <section className="data-card data-card--lg">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Account details</h2>
                <p className="section-subtitle">Core identity and workspace positioning.</p>
              </div>
            </div>
            <div className="data-card__body flex flex-col">
              <div className="flex items-center justify-between gap-3 py-2.5 border-b border-slate-100">
                <span className="flex items-center gap-2 text-xs text-muted-foreground">
                  <UserRound className="h-3.5 w-3.5" /> Full name
                </span>
                <span className="text-sm font-medium text-foreground truncate min-w-0">{user?.full_name}</span>
              </div>
              <div className="flex items-center justify-between gap-3 py-2.5 border-b border-slate-100">
                <span className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Mail className="h-3.5 w-3.5" /> Email
                </span>
                <span className="text-sm font-medium text-foreground truncate min-w-0">{user?.email}</span>
              </div>
              <div className="flex items-center justify-between gap-3 py-2.5">
                <span className="flex items-center gap-2 text-xs text-muted-foreground">
                  <ShieldCheck className="h-3.5 w-3.5" /> Role
                </span>
                <span className="text-sm font-medium text-foreground truncate min-w-0">{roleMeta.label}</span>
              </div>
            </div>
          </section>

          <section className="data-card data-card--lg">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">What you can do</h2>
                <p className="section-subtitle">Role-aware actions available to you.</p>
              </div>
            </div>
            <ul className="data-card__body flex flex-col gap-2 text-sm text-muted-foreground leading-6 list-disc pl-5">
              {roleMeta.capabilities.map((capability) => (
                <li key={capability}>{capability}</li>
              ))}
            </ul>
          </section>

          <section className="data-card data-card--lg">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Working style</h2>
                <p className="section-subtitle">How the workspace is framing your responsibilities.</p>
              </div>
            </div>
            <div className="data-card__body grid gap-3 sm:grid-cols-2">
              <div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <BriefcaseBusiness className="h-3.5 w-3.5" /> Session lane
                </div>
                <p className="mt-1 text-sm font-medium text-foreground">{roleMeta.label}</p>
              </div>
              <div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Workflow className="h-3.5 w-3.5" /> Primary scope
                </div>
                <p className="mt-1 text-sm font-medium text-foreground">{roleMeta.scopeLabel}</p>
              </div>
              <div className="sm:col-span-2">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Ticket className="h-3.5 w-3.5" /> Focus now
                </div>
                <p className="mt-1 text-sm leading-6 text-muted-foreground">{roleMeta.summary}</p>
              </div>
            </div>
          </section>
        </div>

        <div className="flex flex-col gap-6 min-w-0">
          <section className="data-card data-card--lg">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Quick links</h2>
                <p className="section-subtitle">Jump back into the views that matter most.</p>
              </div>
            </div>
            <div className="data-card__body grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
              {roleMeta.quickLinks.map((item) => (
                <Button
                  key={item.href}
                  asChild
                  variant="outline"
                  className="h-auto justify-between px-3 py-2.5 text-left"
                >
                  <Link href={item.href}>
                    <span className="text-sm font-medium truncate min-w-0">{item.label}</span>
                    <ArrowRight className="h-4 w-4 text-muted-foreground shrink-0" />
                  </Link>
                </Button>
              ))}
            </div>
          </section>

          <section className="data-card data-card--lg">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Recent ticket activity</h2>
                <p className="section-subtitle">{roleMeta.recentLabel}</p>
              </div>
              {/* <span className="chip chip--neutral">Auto-refresh · 30s</span> */}
            </div>
            <div className="data-card__body">
              {recentTickets.length === 0 ? (
                <p className="py-6 text-center text-sm text-muted-foreground">
                  No recent ticket activity yet.
                </p>
              ) : (
                <div className="flex flex-col divide-y divide-slate-100">
                  {recentTickets.map((ticket) => (
                    <Link
                      key={ticket.id}
                      href={`/tickets/${ticket.id}`}
                      className="group flex items-start justify-between gap-3 py-3 transition-colors hover:bg-slate-50/60 rounded -mx-2 px-2"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-mono text-xs text-muted-foreground">
                            {ticket.ticket_number}
                          </span>
                          <StatusBadge status={ticket.status} />
                          <PriorityBadge priority={ticket.priority} />
                          {ticket.is_problem ? <span className="chip chip--warning">Problem</span> : null}
                          <SLAWarning breached={ticket.sla_breached} />
                        </div>
                        <h3 className="mt-1.5 line-clamp-1 text-sm font-medium text-foreground">
                          {ticket.title}
                        </h3>
                        <p className="mt-1 text-xs text-muted-foreground">
                          Updated {format(new Date(ticket.updated_at || ticket.created_at), 'MMM d, yyyy h:mm a')}
                        </p>
                      </div>
                      <ArrowRight className="mt-1 h-4 w-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
                    </Link>
                  ))}
                </div>
              )}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
