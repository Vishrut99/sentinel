'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useEffect, useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from '@/components/ui/sidebar';
import { Button } from '@/components/ui/button';
import {
  LayoutDashboard,
  Ticket,
  Plus,
  UserRound,
  Settings,
  LogOut,
  Shield,
  List,
  Clock,
  CheckCircle2,
  CircleDot,
} from 'lucide-react';
import { getTickets } from '@/lib/api';
import { formatDistanceToNow } from 'date-fns';

const STATUS_ICON = {
  open: CircleDot,
  in_progress: Clock,
  resolved: CheckCircle2,
  closed: CheckCircle2,
};

const STATUS_COLOR = {
  open: 'text-amber-400',
  in_progress: 'text-blue-400',
  resolved: 'text-emerald-400',
  closed: 'text-slate-400',
};

export function AppSidebar() {
  const { user, isUser, isAgent, isAdmin, logout } = useAuth();
  const pathname = usePathname();
  const [recentTickets, setRecentTickets] = useState([]);

  useEffect(() => {
    if (!isAdmin && !isAgent) return;
    getTickets({ sort_by: 'updated_at', sort_dir: 'desc', per_page: 5 })
      .then((res) => {
        const list = Array.isArray(res) ? res : res?.data ?? [];
        setRecentTickets(list.slice(0, 5));
      })
      .catch(() => {});
  }, [isAdmin, isAgent]);

  const userNavItems = [
    { href: '/tickets', label: 'My Tickets', icon: Ticket },
    { href: '/tickets/new', label: 'New Ticket', icon: Plus },
    { href: '/profile', label: 'Profile', icon: UserRound },
  ];

  const agentNavItems = [
    { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/manage/tickets', label: 'Assigned Tickets', icon: List },
    { href: '/profile', label: 'Profile', icon: UserRound },
  ];

  const adminNavItems = [
    { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/manage/tickets', label: 'All Tickets', icon: List },
    { href: '/profile', label: 'Profile', icon: UserRound },
    { href: '/admin/settings', label: 'Settings', icon: Settings },
  ];

  const navItems = isAdmin ? adminNavItems : isAgent ? agentNavItems : userNavItems;
  const initials = user?.full_name
    ? user.full_name
        .split(' ')
        .slice(0, 2)
        .map((part) => part[0]?.toUpperCase())
        .join('')
    : 'SC';

  const isActivePath = (href) => pathname === href || pathname.startsWith(`${href}/`);

  return (
    <Sidebar variant="inset" collapsible="offcanvas" className="border-none">
      <SidebarHeader className="px-3 pt-3">
        <div className="rounded-[28px] border border-white/10 bg-white/5 p-3 shadow-[0_18px_40px_-32px_rgba(15,23,42,0.85)]">
          <Link href={isUser ? '/tickets' : '/dashboard'} className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-white/15 text-white shadow-sm">
              <Shield className="h-5 w-5" />
            </div>
            <div className="min-w-0 group-data-[collapsible=icon]:hidden">
              <p className="text-[10px] uppercase tracking-[0.28em] text-sidebar-foreground/50">ITSM</p>
              <p className="font-display text-base font-semibold text-sidebar-foreground">Command Center</p>
              <p className="text-xs text-sidebar-foreground/60">AI-assisted incident ops</p>
            </div>
          </Link>
        </div>
      </SidebarHeader>
      
      <SidebarContent className="px-3 pb-3">
        <SidebarGroup className="p-0">
          <SidebarGroupLabel className="px-3 text-[10px] uppercase tracking-[0.28em] text-sidebar-foreground/45">
            Workspace
          </SidebarGroupLabel>
          <SidebarGroupContent className="px-1">
            <SidebarMenu>
              {navItems.map((item) => (
                <SidebarMenuItem key={item.href}>
                  <SidebarMenuButton
                    asChild
                    isActive={isActivePath(item.href)}
                    size="lg"
                    className="rounded-2xl px-3 text-sidebar-foreground/80 data-[active=true]:bg-white/10 data-[active=true]:text-white"
                  >
                    <Link href={item.href}>
                      <item.icon className="h-4 w-4" />
                      <span>{item.label}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {(isAdmin || isAgent) && recentTickets.length > 0 && (
          <SidebarGroup className="mt-2 p-0">
            <SidebarGroupLabel className="px-3 text-[10px] uppercase tracking-[0.28em] text-sidebar-foreground/45">
              Recent Activity
            </SidebarGroupLabel>
            <SidebarGroupContent className="px-1">
              <div className="mx-1 overflow-hidden rounded-[20px] border border-white/10 bg-white/5">
                {recentTickets.map((ticket, i) => {
                  const StatusIcon = STATUS_ICON[ticket.status] ?? CircleDot;
                  const iconColor = STATUS_COLOR[ticket.status] ?? 'text-slate-400';
                  const updatedAt = ticket.updated_at ?? ticket.UpdatedAt;
                  return (
                    <Link
                      key={ticket.id ?? ticket.ID}
                      href={`/manage/tickets/${ticket.id ?? ticket.ID}`}
                      className={`flex items-start gap-2.5 px-3 py-2.5 transition-colors hover:bg-white/8 ${i !== 0 ? 'border-t border-white/8' : ''}`}
                    >
                      <StatusIcon className={`mt-0.5 h-3.5 w-3.5 shrink-0 ${iconColor}`} />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-[12px] font-medium leading-tight text-sidebar-foreground/90">
                          {ticket.title ?? ticket.Title}
                        </p>
                        <p className="mt-0.5 text-[10px] text-sidebar-foreground/45">
                          {updatedAt ? formatDistanceToNow(new Date(updatedAt), { addSuffix: true }) : '—'}
                        </p>
                      </div>
                    </Link>
                  );
                })}
              </div>
            </SidebarGroupContent>
          </SidebarGroup>
        )}
      </SidebarContent>

      <SidebarFooter className="p-3 pt-0">
        <div className="rounded-[28px] border border-white/10 bg-white/5 p-4 shadow-[0_18px_40px_-34px_rgba(15,23,42,0.8)]">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-white/10 text-sm font-semibold text-white">
              {initials}
            </div>
            <div className="min-w-0 flex-1 text-sm group-data-[collapsible=icon]:hidden">
              <p className="truncate font-medium text-sidebar-foreground">{user?.full_name}</p>
              <p className="truncate text-xs text-sidebar-foreground/58">{user?.email}</p>
              <p className="mt-1 inline-flex rounded-full border border-white/10 bg-white/10 px-2 py-1 text-[10px] uppercase tracking-[0.22em] text-sidebar-foreground/68">
                {user?.role}
              </p>
            </div>
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={logout}
            className="mt-4 w-full rounded-xl border-white/10 bg-white/5 text-sidebar-foreground hover:bg-white/10 hover:text-white"
          >
            <LogOut className="h-4 w-4 mr-2" />
            Logout
          </Button>
        </div>
      </SidebarFooter>

      <SidebarRail />
    </Sidebar>
  );
}
