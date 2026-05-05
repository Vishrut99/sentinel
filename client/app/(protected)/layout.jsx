'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { usePathname } from 'next/navigation';
import { SidebarProvider, SidebarTrigger, SidebarInset } from '@/components/ui/sidebar';
import { AppSidebar, ProtectedRoute } from '@/components/sentinel';
import { Separator } from '@/components/ui/separator';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  CalendarClock,
  ChevronDown,
  LayoutDashboard,
  List,
  LogOut,
  Plus,
  Settings,
  Ticket,
  UserRound,
} from 'lucide-react';
import { useAuth } from '@/context/AuthContext';
import { format } from 'date-fns';

function getPageMeta(pathname) {
  if (pathname.startsWith('/dashboard')) {
    return {
      eyebrow: 'Operations overview',
      title: 'Dashboard',
      description: 'Track service health, incoming pressure, and routing quality from a single command surface.',
    };
  }

  if (pathname.startsWith('/manage/tickets')) {
    return {
      eyebrow: 'Operations queue',
      title: 'Ticket management',
      description: 'Move quickly through assignment, escalations, and backlog pressure without losing context.',
    };
  }

  if (pathname.startsWith('/tickets/new')) {
    return {
      eyebrow: 'Ticket intake',
      title: 'Create a ticket',
      description: 'Capture the issue cleanly so triage, ownership, and downstream resolution start stronger.',
    };
  }

  if (pathname.startsWith('/tickets/')) {
    return {
      eyebrow: 'Ticket workspace',
      title: 'Ticket details',
      description: 'Work the ticket, update ownership, and keep linked problem context visible while you act.',
    };
  }

  if (pathname.startsWith('/tickets')) {
    return {
      eyebrow: 'Requester workspace',
      title: 'My tickets',
      description: 'Follow ticket progress, scan outcomes, and jump into new requests without losing the thread.',
    };
  }

  if (pathname.startsWith('/profile')) {
    return {
      eyebrow: 'Personal workspace',
      title: 'Profile overview',
      description: 'Review your role, current scope, recent tickets, and the shortcuts that matter most in this session.',
    };
  }

  if (pathname.startsWith('/admin/settings')) {
    return {
      eyebrow: 'Administration',
      title: 'Settings',
      description: 'Adjust team capacity, routing controls, and system-level defaults for the workspace.',
    };
  }

  return {
    eyebrow: 'ITSM workspace',
    title: 'Console',
    description: 'Stay oriented, move between core workflows, and keep the command surface close at hand.',
  };
}

function getRoleMeta(role) {
  if (role === 'admin') {
    return {
      label: 'Admin control',
      summary: 'Global queue oversight and service controls are available in this session.',
    };
  }

  if (role === 'agent') {
    return {
      label: 'Agent lane',
      summary: 'Assigned work, active follow-through, and SLA pressure are your current focus.',
    };
  }

  return {
    label: 'Requester lane',
    summary: 'Track requests, add new tickets, and keep communication flowing from one place.',
  };
}

export default function ProtectedLayout({ children }) {
  const pathname = usePathname();
  const { user, isAdmin, isAgent, isUser, logout } = useAuth();
  const pageMeta = getPageMeta(pathname);
  const roleMeta = getRoleMeta(user?.role);
  const [now, setNow] = useState(() => new Date());
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const initials = user?.full_name
    ? user.full_name
        .split(' ')
        .slice(0, 2)
        .map((part) => part[0]?.toUpperCase())
        .join('')
    : 'SC';
  const homeHref = isUser ? '/tickets' : '/dashboard';
  const queueHref = isUser ? '/tickets' : '/manage/tickets';
  const queueLabel = isUser ? 'My Tickets' : isAgent ? 'Assigned Queue' : 'Operations Queue';
  const profileLinks = [
    { href: '/profile', label: 'Profile', icon: UserRound },
    { href: homeHref, label: isUser ? 'Workspace Home' : 'Dashboard', icon: isUser ? Ticket : LayoutDashboard },
    { href: queueHref, label: queueLabel, icon: isUser ? Ticket : List },
    ...(isAdmin ? [{ href: '/admin/settings', label: 'Settings', icon: Settings }] : []),
  ];

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setNow(new Date());
    }, 60_000);

    return () => window.clearInterval(intervalId);
  }, []);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(max-width: 1280px)');

    const syncSidebar = () => {
      setSidebarOpen(!mediaQuery.matches);
    };

    syncSidebar();
    mediaQuery.addEventListener('change', syncSidebar);

    return () => mediaQuery.removeEventListener('change', syncSidebar);
  }, []);

  return (
    <ProtectedRoute>
      <SidebarProvider open={sidebarOpen} onOpenChange={setSidebarOpen}>
        <AppSidebar />
        <SidebarInset className="min-w-0 overflow-x-hidden bg-transparent">
          <header className="sticky top-0 z-20 overflow-x-hidden px-4 py-3 backdrop-blur-xl sm:px-6 lg:px-8">
            <div className="mx-auto w-full max-w-7xl rounded-xl border border-slate-200 bg-white/95 p-3 shadow-sm">
              <div className="flex min-w-0 flex-col gap-3 2xl:flex-row 2xl:items-start 2xl:justify-between">
                <div className="flex min-w-0 flex-1 items-center gap-3">
                  <SidebarTrigger className="-ml-1 shrink-0 rounded-xl border border-slate-200/80 bg-white text-foreground shadow-sm hover:bg-slate-50" />
                  <Separator orientation="vertical" className="hidden h-10 sm:block" />
                  <div className="min-w-0">
                    <p className="text-[10px] uppercase tracking-[0.26em] text-muted-foreground">
                      {pageMeta.eyebrow}
                    </p>
                    <h1 className="font-display text-base font-semibold text-foreground sm:text-lg">
                      {pageMeta.title}
                    </h1>
                  </div>
                </div>

                <div className="flex shrink-0 items-center gap-2">
                  <Button
                    asChild
                    size="sm"
                    className="rounded-lg bg-slate-900 px-3 text-white hover:bg-slate-800"
                  >
                    <Link href="/tickets/new">
                      <Plus className="mr-1.5 h-3.5 w-3.5" />
                      New Ticket
                    </Link>
                  </Button>

                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <button
                        type="button"
                        className="flex items-center gap-2 rounded-xl border border-slate-200/80 bg-white px-2.5 py-1.5 text-left shadow-sm transition-colors hover:bg-slate-50"
                      >
                        <Avatar className="h-7 w-7 border border-primary/10">
                          <AvatarFallback className="bg-primary/10 text-xs font-semibold text-primary">
                            {initials}
                          </AvatarFallback>
                        </Avatar>
                        <div className="hidden min-w-0 sm:block">
                          <p className="truncate text-xs font-medium text-foreground">{user?.full_name}</p>
                          <p className="truncate text-[10px] capitalize text-muted-foreground">{roleMeta.label}</p>
                        </div>
                        <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                      </button>
                    </DropdownMenuTrigger>
                      <DropdownMenuContent
                        align="end"
                        className="w-64 rounded-2xl border border-slate-200/80 bg-white p-2 shadow-md"
                      >
                        <DropdownMenuLabel className="rounded-xl px-3 py-2.5">
                          <div className="flex items-center gap-3">
                            <Avatar className="h-9 w-9 border border-primary/10">
                              <AvatarFallback className="bg-primary/10 text-xs font-semibold text-primary">
                                {initials}
                              </AvatarFallback>
                            </Avatar>
                            <div className="min-w-0">
                              <p className="truncate text-sm font-medium text-foreground">{user?.full_name}</p>
                              <p className="truncate text-xs text-muted-foreground">{user?.email}</p>
                              <span className="mt-1.5 inline-flex rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-[0.18em] text-muted-foreground">
                                {roleMeta.label}
                              </span>
                            </div>
                          </div>
                        </DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        {profileLinks.map((item) => (
                          <DropdownMenuItem key={`${item.href}-${item.label}`} asChild className="rounded-2xl px-3 py-2">
                            <Link href={item.href}>
                              <item.icon className="h-4 w-4" />
                              {item.label}
                            </Link>
                          </DropdownMenuItem>
                        ))}
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          variant="destructive"
                          onSelect={logout}
                          className="rounded-2xl px-3 py-2"
                        >
                          <LogOut className="h-4 w-4" />
                          Logout
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>

              {/* Compact status strip */}
              <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-slate-100 pt-3 text-xs text-muted-foreground">
                <span className="font-medium text-foreground/80">{pageMeta.eyebrow}</span>
                <span className="hidden sm:inline">·</span>
                <span className="hidden sm:inline">{roleMeta.summary}</span>
                <span className="ml-auto flex items-center gap-1.5 shrink-0">
                  <CalendarClock className="h-3 w-3" />
                  {format(now, 'EEE, MMM d')} · {format(now, 'h:mm a')}
                </span>
                <span className="shrink-0 rounded-md border border-slate-200 bg-slate-50 px-2 py-0.5 text-[10px] font-medium uppercase tracking-[0.18em] text-muted-foreground">
                  {roleMeta.label}
                </span>
              </div>
            </div>
          </header>
          <main className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
            <div className="mx-auto w-full max-w-7xl">
              {children}
            </div>
          </main>
        </SidebarInset>
      </SidebarProvider>
    </ProtectedRoute>
  );
}
