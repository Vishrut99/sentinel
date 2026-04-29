'use client';

import { AlertTriangle, Lightbulb } from 'lucide-react';

export function AIInsightPreview({ insights, className = '' }) {
  if (!insights) {
    return null;
  }

  if (insights.triage_status === 'failed') {
    const reason = insights.reason ? `AI triage unavailable: ${insights.reason}` : 'AI triage unavailable';

    return (
      <div className={`flex items-start gap-2 text-xs text-amber-700 ${className}`.trim()}>
        <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
        <span className="line-clamp-2">{reason}</span>
      </div>
    );
  }

  const parts = [];
  if (insights.summary || insights.one_line_summary) parts.push(insights.summary || insights.one_line_summary);
  if (insights.suggested_priority) parts.push(`Suggested priority: ${insights.suggested_priority}`);
  if (insights.suggested_category) parts.push(`Suggested category: ${insights.suggested_category}`);
  if (Array.isArray(insights.required_skills) && insights.required_skills.length > 0) {
    parts.push(`Skills: ${insights.required_skills.join(', ')}`);
  }

  const preview = parts.join(' • ') || 'AI triage available for this ticket.';

  return (
    <div className={`flex items-start gap-2 text-xs text-muted-foreground ${className}`.trim()}>
      <Lightbulb className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span className="line-clamp-2">{preview}</span>
    </div>
  );
}
