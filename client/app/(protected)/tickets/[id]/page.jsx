'use client';

import { useState, useEffect, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '@/context/AuthContext';
import {
  getTicket,
  getComments,
  createComment,
  getAuditLog,
  getAgents,
  getCategories,
  getPriorities,
  getTickets,
  assignTicket,
  updateTicket,
  updateTicketStatus,
  linkProblem,
  FRONTEND_EDITABLE_TICKET_STATUSES,
} from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { Separator } from '@/components/ui/separator';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import {
  StatusBadge,
  PriorityBadge,
  SLAWarning,
  AIInsightPreview,
  ErrorMessage,
  FullPageSpinner,
} from '@/components/sentinel';
import { Spinner } from '@/components/ui/spinner';
import {
  ArrowLeft,
  User,
  Calendar,
  Tag,
  AlertTriangle,
  Lightbulb,
  UserPlus,
  CheckCircle,
  Link2,
  MessageSquare,
  History,
  ExternalLink,
} from 'lucide-react';
import { format } from 'date-fns';

function formatAuditAction(action) {
  return (action || 'activity')
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (char) => char.toUpperCase());
}

function renderAuditDetails(entry) {
  const next = entry?.new_value || {};
  const details = [];

  if (typeof next.assignment_justification === 'string' && next.assignment_justification.trim()) {
    details.push(next.assignment_justification.trim());
  }
  if (Array.isArray(next.required_skills) && next.required_skills.length > 0) {
    details.push(`Skills: ${next.required_skills.join(', ')}`);
  }
  if (typeof next.agent_name === 'string' && next.agent_name.trim()) {
    details.push(`Owner: ${next.agent_name.trim()}`);
  }

  return details;
}

export default function TicketDetailPage() {
  const { id } = useParams();
  const router = useRouter();
  const { isAdmin, isAgentOrAdmin } = useAuth();

  const [ticket, setTicket] = useState(null);
  const [comments, setComments] = useState([]);
  const [auditLog, setAuditLog] = useState([]);
  const [agents, setAgents] = useState([]);
  const [categories, setCategories] = useState([]);
  const [priorities, setPriorities] = useState([]);

  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');

  const [newComment, setNewComment] = useState('');
  const [isInternal, setIsInternal] = useState(false);
  const [submittingComment, setSubmittingComment] = useState(false);

  const [selectedAgentId, setSelectedAgentId] = useState('');
  const [assigning, setAssigning] = useState(false);
  const [assignDialogOpen, setAssignDialogOpen] = useState(false);

  const [selectedStatus, setSelectedStatus] = useState('');
  const [updatingStatus, setUpdatingStatus] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [savingTicket, setSavingTicket] = useState(false);
  const [editForm, setEditForm] = useState({
    title: '',
    description: '',
    priority: '',
    categoryId: '',
    changeReason: '',
  });

  const [linkProblemId, setLinkProblemId] = useState('');
  const [problemSearchQuery, setProblemSearchQuery] = useState('');
  const [problemResults, setProblemResults] = useState([]);
  const [searchingProblems, setSearchingProblems] = useState(false);
  const [linking, setLinking] = useState(false);
  const [linkDialogOpen, setLinkDialogOpen] = useState(false);

  const fetchTicketData = useCallback(async ({ silent = false } = {}) => {
    if (silent) {
      setRefreshing(true);
    } else {
      setLoading(true);
    }
    setError('');
    try {
      const [ticketData, commentsData, auditData] = await Promise.all([
        getTicket(id),
        getComments(id).catch(() => []),
        getAuditLog(id).catch(() => []),
      ]);

      setTicket(ticketData);
      setComments(Array.isArray(commentsData) ? commentsData : commentsData?.data || []);
      setAuditLog(Array.isArray(auditData) ? auditData : auditData?.data || []);

      if (isAdmin) {
        const agentsData = await getAgents().catch(() => []);
        setAgents(Array.isArray(agentsData) ? agentsData : agentsData?.data || []);
      }
    } catch (err) {
      setError(err.message || 'Failed to load ticket');
    } finally {
      if (silent) {
        setRefreshing(false);
      } else {
        setLoading(false);
      }
    }
  }, [id, isAdmin]);

  useEffect(() => {
    fetchTicketData();
  }, [fetchTicketData]);

  useEffect(() => {
    if (!isAgentOrAdmin) {
      return;
    }

    const loadLookups = async () => {
      const [categoriesData, prioritiesData] = await Promise.all([
        getCategories().catch(() => []),
        getPriorities().catch(() => []),
      ]);

      setCategories(Array.isArray(categoriesData) ? categoriesData : categoriesData?.data || []);
      setPriorities(Array.isArray(prioritiesData) ? prioritiesData : prioritiesData?.data || []);
    };

    loadLookups();
  }, [isAgentOrAdmin]);

  useEffect(() => {
    if (ticket?.status && FRONTEND_EDITABLE_TICKET_STATUSES.includes(ticket.status)) {
      setSelectedStatus(ticket.status);
      return;
    }

    setSelectedStatus('');
  }, [ticket?.status]);

  useEffect(() => {
    if (!ticket) {
      return;
    }

    setEditForm({
      title: ticket.title || '',
      description: ticket.description || '',
      priority: ticket.priority || '',
      categoryId: ticket.category_id || '',
      changeReason: '',
    });
  }, [ticket]);

  useEffect(() => {
    if (!linkDialogOpen) {
      setProblemResults([]);
      setSearchingProblems(false);
      setLinkProblemId('');
      setProblemSearchQuery('');
      return;
    }

    const query = problemSearchQuery.trim();
    if (query.length < 2) {
      setProblemResults([]);
      setLinkProblemId('');
      return;
    }

    const timeoutId = setTimeout(async () => {
      setSearchingProblems(true);
      try {
        const response = await getTickets({
          q: query,
          is_problem: true,
          per_page: 8,
          sort_by: 'updated_at',
          sort_dir: 'desc',
        });
        const results = Array.isArray(response?.data) ? response.data : [];
        setProblemResults(results.filter((candidate) => candidate.id !== id));
      } catch (err) {
        setError(err.message || 'Failed to search problem tickets');
      } finally {
        setSearchingProblems(false);
      }
    }, 300);

    return () => clearTimeout(timeoutId);
  }, [problemSearchQuery, linkDialogOpen, id]);

  const handleSubmitComment = async (e) => {
    e.preventDefault();
    if (!newComment.trim()) return;

    setSubmittingComment(true);
    try {
      await createComment(id, newComment.trim(), isInternal);
      setNewComment('');
      setIsInternal(false);
      // Refresh comments
      const commentsData = await getComments(id);
      setComments(Array.isArray(commentsData) ? commentsData : commentsData?.data || []);
    } catch (err) {
      setError(err.message || 'Failed to add comment');
    } finally {
      setSubmittingComment(false);
    }
  };

  const handleAssign = async () => {
    if (!selectedAgentId) return;

    setAssigning(true);
    try {
      await assignTicket(id, selectedAgentId);
      await fetchTicketData({ silent: true });
      setAssignDialogOpen(false);
      setSelectedAgentId('');
    } catch (err) {
      setError(err.message || 'Failed to assign ticket');
    } finally {
      setAssigning(false);
    }
  };

  const handleUpdateStatus = async () => {
    if (!selectedStatus) return;

    setUpdatingStatus(true);
    try {
      await updateTicketStatus(id, selectedStatus);
      await fetchTicketData({ silent: true });
    } catch (err) {
      setError(err.message || 'Failed to update ticket status');
    } finally {
      setUpdatingStatus(false);
    }
  };

  const handleEditFieldChange = (field, value) => {
    setEditForm((current) => ({
      ...current,
      [field]: value,
    }));
  };

  const handleSaveTicket = async () => {
    if (!ticket) return;

    const payload = {};
    const trimmedTitle = editForm.title.trim();
    const trimmedDescription = editForm.description.trim();
    const trimmedReason = editForm.changeReason.trim();

    if (trimmedTitle !== ticket.title) {
      payload.title = trimmedTitle;
    }
    if (trimmedDescription !== (ticket.description || '')) {
      payload.description = trimmedDescription;
    }
    if (editForm.priority !== ticket.priority) {
      payload.priority = editForm.priority;
    }
    if (editForm.categoryId !== ticket.category_id) {
      payload.category_id = editForm.categoryId;
    }
    if (trimmedReason) {
      payload.change_reason = trimmedReason;
    }

    if (Object.keys(payload).length === 0) {
      return;
    }

    setSavingTicket(true);
    try {
      await updateTicket(id, payload);
      await fetchTicketData({ silent: true });
      setEditDialogOpen(false);
    } catch (err) {
      setError(err.message || 'Failed to update ticket');
    } finally {
      setSavingTicket(false);
    }
  };

  const handleLinkProblem = async () => {
    if (!linkProblemId.trim()) return;

    setLinking(true);
    try {
      await linkProblem(id, linkProblemId.trim());
      await fetchTicketData({ silent: true });
      setLinkDialogOpen(false);
      setLinkProblemId('');
      setProblemSearchQuery('');
      setProblemResults([]);
    } catch (err) {
      setError(err.message || 'Failed to link problem');
    } finally {
      setLinking(false);
    }
  };

  if (loading) {
    return <FullPageSpinner />;
  }

  if (error && !ticket) {
    return (
      <div className="page-shell">
        <ErrorMessage title="Error Loading Ticket" message={error} />
        <Button variant="outline" onClick={() => router.back()}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          Go Back
        </Button>
      </div>
    );
  }

  if (!ticket) return null;

  const ticketCategory =
    typeof ticket.category === 'string' ? ticket.category : ticket.category?.name;
  const canEditStatus = isAgentOrAdmin && FRONTEND_EDITABLE_TICKET_STATUSES.includes(ticket.status);
  const classificationChanged =
    editForm.priority !== ticket.priority || editForm.categoryId !== ticket.category_id;
  const hasTicketEdits =
    editForm.title.trim() !== (ticket.title || '') ||
    editForm.description.trim() !== (ticket.description || '') ||
    editForm.priority !== ticket.priority ||
    editForm.categoryId !== ticket.category_id;
  const statusOptions = [
    { value: 'open', label: 'Open' },
    { value: 'in_progress', label: 'In Progress' },
    { value: 'resolved', label: 'Resolved' },
  ];
  const createdByName = ticket.created_by?.full_name || 'Unknown';
  const assignedToName =
    ticket.assigned_agent?.full_name || ticket.assigned_agent?.user?.full_name || 'Unassigned';
  const requiredSkills = Array.isArray(ticket.required_skills) ? ticket.required_skills : [];
  const assignmentJustification = ticket.assignment_justification || '';
  const backHref = isAgentOrAdmin ? '/manage/tickets' : '/tickets';
  const linkProblemLabel = ticket.parent_id ? 'Change Problem Link' : 'Link to Problem';
  const detailRows = [
    {
      label: 'Created by',
      value: createdByName,
      icon: User,
    },
    {
      label: 'Assigned to',
      value: assignedToName,
      icon: UserPlus,
    },
    {
      label: 'Category',
      value: ticketCategory || 'Uncategorized',
      icon: Tag,
    },
    {
      label: 'Created',
      value: ticket.created_at ? format(new Date(ticket.created_at), 'MMM d, yyyy h:mm a') : 'N/A',
      icon: Calendar,
    },
  ];

  if (ticket.due_at) {
    detailRows.push({
      label: 'Due',
      value: format(new Date(ticket.due_at), 'MMM d, yyyy h:mm a'),
      icon: Calendar,
    });
  }

  if (ticket.resolved_at) {
    detailRows.push({
      label: 'Resolved',
      value: format(new Date(ticket.resolved_at), 'MMM d, yyyy h:mm a'),
      icon: CheckCircle,
    });
  }

  return (
    <div className="page-shell">
      <header className="page-header">
        <div className="min-w-0">
          <span className="page-header__eyebrow">Ticket workspace</span>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <span className="chip chip--neutral">{ticket.ticket_number}</span>
            <StatusBadge status={ticket.status} />
            <PriorityBadge priority={ticket.priority} />
            <SLAWarning breached={ticket.sla_breached} />
          </div>
          <h1 className="page-header__title">{ticket.title}</h1>
          <p className="page-header__description">
            Created by {createdByName} and currently owned by {assignedToName}. Use this workspace
            to manage updates, collaboration, and linked problem context in one place.
          </p>
          <AIInsightPreview insights={ticket.ai_insights} className="mt-4 max-w-2xl" />
          {requiredSkills.length > 0 && (
            <div className="mt-4 flex flex-wrap gap-2">
              {requiredSkills.map((skill) => (
                <span key={skill} className="chip chip--neutral">
                  {skill}
                </span>
              ))}
            </div>
          )}
        </div>
        <Button variant="outline" asChild>
          <Link href={backHref}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to queue
          </Link>
        </Button>
      </header>

      <ErrorMessage message={error} />
      {refreshing ? (
        <div className="inline-flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner size="sm" />
          Refreshing workspace...
        </div>
      ) : null}

      {isAgentOrAdmin && (
        <section className="data-card">
          <div className="data-card__header">
            <div className="min-w-0">
              <h2 className="section-title">Agent actions</h2>
              <p className="section-subtitle">Update ownership, classification, status, or problem links from this control bar.</p>
            </div>
          </div>
          <div className="data-card__body">
            <div className="flex flex-wrap gap-3">
              {isAdmin && (
                <Dialog open={assignDialogOpen} onOpenChange={setAssignDialogOpen}>
                  <DialogTrigger asChild>
                    <Button variant="outline" disabled={refreshing}>
                      <UserPlus className="h-4 w-4 mr-2" />
                      Assign
                    </Button>
                  </DialogTrigger>
                  <DialogContent>
                    <DialogHeader>
                      <DialogTitle>Assign Ticket</DialogTitle>
                      <DialogDescription>
                        Select an agent to assign this ticket to.
                        {requiredSkills.length > 0 ? ` Required skills: ${requiredSkills.join(', ')}.` : ''}
                      </DialogDescription>
                    </DialogHeader>
                    <div className="py-4">
                      <Select value={selectedAgentId} onValueChange={setSelectedAgentId}>
                        <SelectTrigger>
                          <SelectValue placeholder="Select agent" />
                        </SelectTrigger>
                        <SelectContent>
                          {agents.map((agent) => (
                            <SelectItem key={agent.id} value={agent.id}>
                              {agent.user?.full_name || agent.full_name || 'Unknown Agent'}
                              {agent.department && ` - ${agent.department}`}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <DialogFooter>
                      <Button variant="outline" onClick={() => setAssignDialogOpen(false)}>
                        Cancel
                      </Button>
                      <Button
                        onClick={handleAssign}
                        disabled={!selectedAgentId || assigning || refreshing}
                      >
                        {assigning ? <Spinner size="sm" className="mr-2" /> : null}
                        Assign
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              )}

              <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
                <DialogTrigger asChild>
                  <Button variant="outline" disabled={refreshing}>
                    Edit Ticket
                  </Button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-2xl">
                  <DialogHeader>
                    <DialogTitle>Edit Ticket</DialogTitle>
                    <DialogDescription>
                      Agents and admins can update ticket details. Changing priority or category requires a reason and creates an internal comment.
                    </DialogDescription>
                  </DialogHeader>
                  <div className="grid gap-4 py-4">
                    <div className="grid gap-2">
                      <Label htmlFor="edit-title" className="form-label">Title</Label>
                      <Input
                        id="edit-title"
                        value={editForm.title}
                        onChange={(e) => handleEditFieldChange('title', e.target.value)}
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label htmlFor="edit-description" className="form-label">Description</Label>
                      <Textarea
                        id="edit-description"
                        rows={4}
                        value={editForm.description}
                        onChange={(e) => handleEditFieldChange('description', e.target.value)}
                      />
                    </div>
                    <div className="form-grid">
                      <div className="grid gap-2">
                        <Label className="form-label">Priority</Label>
                        <Select value={editForm.priority} onValueChange={(value) => handleEditFieldChange('priority', value)}>
                          <SelectTrigger>
                            <SelectValue placeholder="Select priority" />
                          </SelectTrigger>
                          <SelectContent>
                            {priorities.map((priority) => (
                              <SelectItem key={priority.id || priority.name} value={priority.name}>
                                {priority.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="grid gap-2">
                        <Label className="form-label">Category</Label>
                        <Select value={editForm.categoryId} onValueChange={(value) => handleEditFieldChange('categoryId', value)}>
                          <SelectTrigger>
                            <SelectValue placeholder="Select category" />
                          </SelectTrigger>
                          <SelectContent>
                            {categories.map((category) => (
                              <SelectItem key={category.id} value={category.id}>
                                {category.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                    <div className="grid gap-2">
                      <Label htmlFor="change-reason" className="form-label">
                        Change Reason {classificationChanged ? '(Required)' : '(Optional)'}
                      </Label>
                      <Textarea
                        id="change-reason"
                        rows={3}
                        placeholder="Explain why you changed the category or priority"
                        value={editForm.changeReason}
                        onChange={(e) => handleEditFieldChange('changeReason', e.target.value)}
                      />
                    </div>
                  </div>
                  <DialogFooter>
                    <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
                      Cancel
                    </Button>
                    <Button
                      onClick={handleSaveTicket}
                      disabled={
                        !hasTicketEdits ||
                        (classificationChanged && !editForm.changeReason.trim()) ||
                        savingTicket ||
                        refreshing
                      }
                    >
                      {savingTicket ? <Spinner size="sm" className="mr-2" /> : null}
                      Save Changes
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>

              {canEditStatus && (
                <div className="flex flex-wrap items-center gap-3">
                  <Select value={selectedStatus} onValueChange={setSelectedStatus}>
                    <SelectTrigger className="w-[200px]">
                      <SelectValue placeholder="Select status" />
                    </SelectTrigger>
                    <SelectContent>
                      {statusOptions.map((statusOption) => (
                        <SelectItem key={statusOption.value} value={statusOption.value}>
                          {statusOption.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <Button
                    variant="outline"
                    onClick={handleUpdateStatus}
                    disabled={!selectedStatus || selectedStatus === ticket.status || updatingStatus || refreshing}
                  >
                    {updatingStatus ? <Spinner size="sm" className="mr-2" /> : <CheckCircle className="h-4 w-4 mr-2" />}
                    Update Status
                  </Button>
                </div>
              )}

              {!ticket.is_problem ? (
                <Dialog open={linkDialogOpen} onOpenChange={setLinkDialogOpen}>
                  <DialogTrigger asChild>
                    <Button variant="outline" disabled={refreshing}>
                      <Link2 className="h-4 w-4 mr-2" />
                      {linkProblemLabel}
                    </Button>
                  </DialogTrigger>
                  <DialogContent>
                    <DialogHeader>
                      <DialogTitle>{linkProblemLabel}</DialogTitle>
                      <DialogDescription>
                        Search problem tickets by title, ticket number, or related keywords and link the correct parent ticket.
                      </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                      <Input
                        placeholder="Search problem ticket title or keywords"
                        value={problemSearchQuery}
                        onChange={(e) => {
                          setProblemSearchQuery(e.target.value);
                          setLinkProblemId('');
                        }}
                      />
                      <div className="max-h-60 space-y-2 overflow-y-auto">
                        {searchingProblems ? (
                          <div className="flex items-center text-sm text-muted-foreground">
                            <Spinner size="sm" className="mr-2" />
                            Searching problem tickets...
                          </div>
                        ) : problemSearchQuery.trim().length < 2 ? (
                          <p className="text-sm text-muted-foreground">Type at least 2 characters to search by title or keywords.</p>
                        ) : problemResults.length === 0 ? (
                          <p className="text-sm text-muted-foreground">No matching problem tickets found yet.</p>
                        ) : (
                          problemResults.map((problemTicket) => (
                            <button
                              key={problemTicket.id}
                              type="button"
                              onClick={() => setLinkProblemId(problemTicket.id)}
                              className={`w-full rounded-md border p-3 text-left transition-colors ${
                                linkProblemId === problemTicket.id
                                  ? 'border-primary bg-primary/5'
                                  : 'hover:bg-muted/50'
                              }`}
                            >
                              <div className="flex items-center justify-between gap-2 min-w-0">
                                <span className="font-medium min-w-0">{problemTicket.title}</span>
                                <span className="font-mono text-xs text-muted-foreground">{problemTicket.ticket_number}</span>
                              </div>
                              <AIInsightPreview insights={problemTicket.ai_insights} className="mt-2" />
                            </button>
                          ))
                        )}
                      </div>
                    </div>
                    <DialogFooter>
                      <Button variant="outline" onClick={() => setLinkDialogOpen(false)}>
                        Cancel
                      </Button>
                      <Button
                        onClick={handleLinkProblem}
                        disabled={!linkProblemId.trim() || linking || refreshing}
                      >
                        {linking ? <Spinner size="sm" className="mr-2" /> : null}
                        Save Link
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              ) : null}
            </div>
          </div>
        </section>
      )}

      <div className="grid grid-equal grid-cols-1 lg:grid-cols-[1fr_320px]">
        <div className="flex flex-col gap-6 min-w-0">
          <section className="data-card">
            <div className="data-card__header">
              <h2 className="section-title">Description</h2>
            </div>
            <div className="data-card__body">
              <p className="whitespace-pre-wrap text-sm text-muted-foreground">
                {ticket.description || 'No description provided.'}
              </p>
            </div>
          </section>

          {ticket.ai_insights && (
            <section className="data-card">
              <div className="data-card__header">
                <div className="flex items-center gap-2 min-w-0">
                  <Lightbulb className="h-5 w-5 text-yellow-500" />
                  <h2 className="section-title">AI Insights</h2>
                </div>
              </div>
              <div className="data-card__body">
                {ticket.ai_insights.triage_status === 'failed' ? (
                  <div className="space-y-2 rounded-md border border-amber-200 bg-amber-50 p-4 text-amber-700">
                    <div className="flex items-center gap-2">
                      <AlertTriangle className="h-4 w-4" />
                      <span>AI triage unavailable</span>
                    </div>
                    {ticket.ai_insights.reason && (
                      <p className="text-sm">{ticket.ai_insights.reason}</p>
                    )}
                  </div>
                ) : (
                  <div className="space-y-3">
                    {ticket.ai_insights.suggested_priority && (
                      <div>
                        <span className="text-sm font-medium">Suggested Priority: </span>
                        <PriorityBadge priority={ticket.ai_insights.suggested_priority} />
                      </div>
                    )}
                    {ticket.ai_insights.suggested_category && (
                      <div>
                        <span className="text-sm font-medium">Suggested Category: </span>
                        <span className="text-sm text-muted-foreground">{ticket.ai_insights.suggested_category}</span>
                      </div>
                    )}
                    {(ticket.ai_insights.summary || ticket.ai_insights.one_line_summary) && (
                      <div>
                        <span className="text-sm font-medium">Summary: </span>
                        <p className="text-sm text-muted-foreground mt-1">
                          {ticket.ai_insights.summary || ticket.ai_insights.one_line_summary}
                        </p>
                      </div>
                    )}
                    {requiredSkills.length > 0 && (
                      <div>
                        <span className="text-sm font-medium">Required Skills: </span>
                        <div className="mt-2 flex flex-wrap gap-2">
                          {requiredSkills.map((skill) => (
                            <span key={skill} className="chip chip--neutral">
                              {skill}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </section>
          )}

          <section className="data-card">
            <div className="data-card__header">
              <div className="flex items-center gap-2 min-w-0">
                <MessageSquare className="h-5 w-5" />
                <h2 className="section-title">Comments</h2>
              </div>
            </div>
            <div className="data-card__body--scroll space-y-4">
              {comments.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-4">No comments yet</p>
              ) : (
                <div className="space-y-3">
                  {comments.map((comment) => (
                    <div
                      key={comment.id}
                      className={`rounded-md border p-4 ${
                        comment.isInternal
                          ? 'border-yellow-200 bg-yellow-50'
                          : 'border-border bg-card'
                      }`}
                    >
                      <div className="flex items-center justify-between mb-2 min-w-0">
                        <span className="font-medium text-sm min-w-0">
                          {comment.author?.fullName || comment.author?.full_name || 'Unknown'}
                        </span>
                        <div className="flex items-center gap-2">
                          {comment.isInternal && (
                            <span className="chip chip--warning">Internal</span>
                          )}
                          <span className="text-xs text-muted-foreground">
                            {comment.createdAt ? format(new Date(comment.createdAt), 'MMM d, yyyy h:mm a') : ''}
                          </span>
                        </div>
                      </div>
                      <p className="text-sm whitespace-pre-wrap">{comment.body}</p>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <form onSubmit={handleSubmitComment} className="data-card__footer space-y-4">
              <div className="space-y-2">
                <Label htmlFor="comment" className="form-label">Add Comment</Label>
                <Textarea
                  id="comment"
                  placeholder="Write your comment..."
                  value={newComment}
                  onChange={(e) => setNewComment(e.target.value)}
                  disabled={submittingComment}
                  rows={3}
                />
              </div>

              {isAgentOrAdmin && (
                <div className="flex items-center space-x-2">
                  <Switch
                    id="internal"
                    checked={isInternal}
                    onCheckedChange={setIsInternal}
                    disabled={submittingComment}
                  />
                  <Label htmlFor="internal" className="font-normal">
                    Internal comment (visible only to agents and admins)
                  </Label>
                </div>
              )}

              <Button type="submit" disabled={!newComment.trim() || submittingComment}>
                {submittingComment ? <Spinner size="sm" className="mr-2" /> : null}
                Add Comment
              </Button>
            </form>
          </section>

          <section className="data-card">
            <div className="data-card__header">
              <div className="flex items-center gap-2 min-w-0">
                <History className="h-5 w-5" />
                <h2 className="section-title">Activity Timeline</h2>
              </div>
            </div>
            <div className="data-card__body">
              {auditLog.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-4">No activity recorded</p>
              ) : (
                <div className="relative pl-6 space-y-6">
                  <div className="absolute left-2 top-2 bottom-2 w-px bg-border" />
                  {auditLog.map((entry, index) => (
                    <div key={entry.id || index} className="relative">
                      <div className="absolute -left-4 top-2 h-3 w-3 rounded-full border-2 border-background bg-primary" />
                      <div className="min-w-0">
                        <p className="text-sm font-medium">{formatAuditAction(entry.action)}</p>
                        <div className="flex items-center gap-2 text-xs text-muted-foreground">
                          <span>{entry.actor?.full_name || entry.actor_name || 'System'}</span>
                          <span>•</span>
                          <span>
                            {entry.created_at ? format(new Date(entry.created_at), 'MMM d, yyyy h:mm a') : ''}
                          </span>
                        </div>
                        {renderAuditDetails(entry).length > 0 && (
                          <div className="mt-2 space-y-1">
                            {renderAuditDetails(entry).map((detail) => (
                              <p key={detail} className="text-sm leading-6 text-muted-foreground">
                                {detail}
                              </p>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </section>
        </div>

        <aside className="flex flex-col gap-6 min-w-0">
          <section className="data-card">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Details</h2>
                <p className="section-subtitle">Ownership, timing, and relationship context for this ticket.</p>
              </div>
            </div>
            <div className="data-card__body space-y-4">
              {detailRows.map((row) => (
                <div key={row.label} className="flex items-center gap-3 min-w-0">
                  <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary/10 text-primary shrink-0">
                    <row.icon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-medium">{row.label}</p>
                    <p className="text-sm text-muted-foreground">{row.value}</p>
                  </div>
                </div>
              ))}

              {assignmentJustification && (
                <>
                  <hr className="hr-soft" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium">Assignment Justification</p>
                    <p className="mt-2 text-sm leading-6 text-muted-foreground">{assignmentJustification}</p>
                  </div>
                </>
              )}

              {requiredSkills.length > 0 && (
                <>
                  <hr className="hr-soft" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium">Required Skills</p>
                    <div className="mt-3 flex flex-wrap gap-2">
                      {requiredSkills.map((skill) => (
                        <span key={skill} className="chip chip--neutral">
                          {skill}
                        </span>
                      ))}
                    </div>
                  </div>
                </>
              )}

              {ticket.is_problem && (
                <div className="flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700">
                  <AlertTriangle className="h-4 w-4" />
                  <span className="font-medium">Problem ticket</span>
                </div>
              )}

              {ticket.parent_id && (
                <>
                  <hr className="hr-soft" />
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary/10 text-primary shrink-0">
                      <Link2 className="h-4 w-4" />
                    </div>
                    <div className="min-w-0">
                      <p className="text-sm font-medium">Parent Problem</p>
                      <Link
                        href={`/tickets/${ticket.parent_id}`}
                        className="mt-1 inline-flex items-center gap-1 text-sm text-primary hover:underline"
                      >
                        View parent ticket
                        <ExternalLink className="h-3 w-3" />
                      </Link>
                    </div>
                  </div>
                </>
              )}
            </div>
          </section>
        </aside>
      </div>
    </div>
  );
}
