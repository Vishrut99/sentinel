'use client';

import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty';
import { Inbox, Ticket, Users, FileText } from 'lucide-react';

const iconMap = {
  tickets: Ticket,
  users: Users,
  comments: FileText,
  default: Inbox,
};

export function EmptyState({ 
  title = 'No items found', 
  description = 'There are no items to display.',
  icon = 'default',
  action,
}) {
  const IconComponent = iconMap[icon] || iconMap.default;
  
  return (
    <Empty className="sentinel-panel rounded-[28px] border-white/75 bg-white/82 py-12 shadow-[0_24px_70px_-42px_rgba(15,23,42,0.42)]">
      <EmptyHeader>
        <EmptyMedia variant="icon" className="rounded-2xl bg-primary/10 text-primary">
          <IconComponent className="h-10 w-10 text-muted-foreground" />
        </EmptyMedia>
        <EmptyTitle className="font-display text-2xl">{title}</EmptyTitle>
        <EmptyDescription className="max-w-md">{description}</EmptyDescription>
      </EmptyHeader>
      {action && <EmptyContent>{action}</EmptyContent>}
    </Empty>
  );
}
