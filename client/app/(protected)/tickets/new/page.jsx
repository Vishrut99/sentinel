'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { createTicket, getCategories } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Checkbox } from '@/components/ui/checkbox';
import { ErrorMessage } from '@/components/sentinel';
import { Spinner } from '@/components/ui/spinner';
import { ArrowLeft, ShieldCheck } from 'lucide-react';
import Link from 'next/link';

const requestGuidance = [
  'Summarize the user-facing impact in the title.',
  'Include steps to reproduce if the issue is repeatable.',
  'Optional routing hints like VPN or Account Access can speed up assignment.',
  'Add environment or device context when it matters.',
];

function parseRequiredSkills(value) {
  if (!value) {
    return [];
  }

  const seen = new Set();
  return value
    .split(/\n|,/)
    .map((part) => part.trim())
    .filter(Boolean)
    .filter((skill) => {
      const key = skill.toLowerCase().replace(/\s+/g, ' ');
      if (seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
}

export default function CreateTicketPage() {
  const router = useRouter();
  const { user, isAgentOrAdmin } = useAuth();

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [requiredSkillsInput, setRequiredSkillsInput] = useState('');
  const [priority, setPriority] = useState('medium');
  const [categoryId, setCategoryId] = useState('');
  const [isProblem, setIsProblem] = useState(false);
  const [parentId, setParentId] = useState('');

  const [categories, setCategories] = useState([]);
  const [loadingCategories, setLoadingCategories] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [validationErrors, setValidationErrors] = useState({});

  useEffect(() => {
    const fetchCategories = async () => {
      try {
        const data = await getCategories();
        const categoryList = Array.isArray(data) ? data : data?.data || [];
        setCategories(categoryList);
        setCategoryId((current) => current || categoryList[0]?.id || '');
      } catch (err) {
        console.error('Failed to load categories:', err);
      } finally {
        setLoadingCategories(false);
      }
    };

    fetchCategories();
  }, []);

  const validateForm = () => {
    const errors = {};

    if (!title.trim()) {
      errors.title = 'Title is required';
    } else if (title.trim().length < 5) {
      errors.title = 'Title must be at least 5 characters';
    }

    if (!categoryId) {
      errors.categoryId = 'Category is required';
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');

    if (!validateForm()) return;

    setSubmitting(true);
    try {
      const requiredSkills = parseRequiredSkills(requiredSkillsInput);
      const ticketData = {
        title: title.trim(),
        description: description.trim(),
        priority,
      };

      if (categoryId) {
        ticketData.category_id = categoryId;
      }
      if (requiredSkills.length > 0) {
        ticketData.required_skills = requiredSkills;
      }

      if (isAgentOrAdmin) {
        ticketData.is_problem = isProblem;
        if (parentId.trim()) {
          ticketData.parent_id = parentId.trim();
        }
      }

      const newTicket = await createTicket(ticketData);
      router.push(`/tickets/${newTicket.id}`);
    } catch (err) {
      setError(err.message || 'Failed to create ticket');
    } finally {
      setSubmitting(false);
    }
  };

  const backHref = user?.role === 'user' ? '/tickets' : '/manage/tickets';
  const requiredSkillPreview = parseRequiredSkills(requiredSkillsInput);

  return (
    <div className="page-shell">
      <header className="page-header">
        <div className="min-w-0">
          <span className="page-header__eyebrow">Ticket intake</span>
          <h1 className="page-header__title">Create a new ticket</h1>
          <p className="page-header__description">
            Capture the issue once, route it cleanly, and give the next responder enough context
            to act without chasing for basics.
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link href={backHref}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to queue
          </Link>
        </Button>
      </header>

      <div className="grid grid-equal grid-cols-1 xl:grid-cols-[1.15fr_0.85fr]">
        <section className="data-card data-card--lg">
          <div className="data-card__header">
            <div className="min-w-0">
              <h2 className="section-title">Ticket details</h2>
              <p className="section-subtitle">Fill in the essentials and Sentinel will take it from there.</p>
            </div>
          </div>
          <form onSubmit={handleSubmit} className="data-card__body space-y-6">
            <div className="space-y-2">
              <Label htmlFor="title" className="form-label">
                Title <span className="text-destructive">*</span>
              </Label>
              <Input
                id="title"
                placeholder="Brief summary of the issue"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                disabled={submitting}
                className={validationErrors.title ? 'border-destructive' : ''}
              />
              {validationErrors.title ? (
                <p className="text-sm text-destructive">{validationErrors.title}</p>
              ) : null}
            </div>

            <div className="space-y-2">
              <Label htmlFor="description" className="form-label">Description</Label>
              <Textarea
                id="description"
                placeholder="Describe the issue, user impact, steps to reproduce, and anything already tried."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                disabled={submitting}
                rows={6}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="requiredSkills" className="form-label">Routing hints</Label>
              <Textarea
                id="requiredSkills"
                placeholder="Optional skills such as VPN, Network, Account Access"
                value={requiredSkillsInput}
                onChange={(e) => setRequiredSkillsInput(e.target.value)}
                disabled={submitting}
                rows={3}
              />
              <p className="text-xs text-muted-foreground">
                Separate skills with commas or new lines. Sentinel can still infer skills from the issue,
                so this field is only for stronger routing hints.
              </p>
              {requiredSkillPreview.length > 0 ? (
                <div className="flex flex-wrap gap-2 pt-1">
                  {requiredSkillPreview.map((skill) => (
                    <span key={skill} className="chip chip--info">
                      {skill}
                    </span>
                  ))}
                </div>
              ) : null}
            </div>

            <div className="form-grid">
              <div className="space-y-2">
                <Label htmlFor="priority" className="form-label">Priority</Label>
                <Select value={priority} onValueChange={setPriority} disabled={submitting}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select priority" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="low">Low</SelectItem>
                    <SelectItem value="medium">Medium</SelectItem>
                    <SelectItem value="high">High</SelectItem>
                    <SelectItem value="critical">Critical</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="category" className="form-label">Category</Label>
                {loadingCategories ? (
                  <div className="flex h-9 items-center rounded-md border px-3">
                    <Spinner size="sm" />
                  </div>
                ) : (
                  <Select value={categoryId} onValueChange={setCategoryId} disabled={submitting}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select category" />
                    </SelectTrigger>
                    <SelectContent>
                      {categories.map((cat) => (
                        <SelectItem key={cat.id} value={cat.id}>
                          {cat.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
                {validationErrors.categoryId ? (
                  <p className="text-sm text-destructive">{validationErrors.categoryId}</p>
                ) : null}
              </div>
            </div>

            {isAgentOrAdmin ? (
              <div className="data-card">
                <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                  <ShieldCheck className="h-3.5 w-3.5 text-primary" />
                  Agent options
                </div>
                <div className="mt-4 space-y-5">
                  <div className="flex items-center space-x-3">
                    <Checkbox
                      id="isProblem"
                      checked={isProblem}
                      onCheckedChange={(checked) => setIsProblem(Boolean(checked))}
                      disabled={submitting}
                    />
                    <Label htmlFor="isProblem" className="font-normal">
                      Mark as problem ticket
                    </Label>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="parentId" className="form-label">Parent problem ID</Label>
                    <Input
                      id="parentId"
                      placeholder="UUID of parent problem ticket"
                      value={parentId}
                      onChange={(e) => setParentId(e.target.value)}
                      disabled={submitting}
                    />
                    <p className="text-xs text-muted-foreground">
                      Use this when the new ticket should roll up under an existing problem investigation.
                    </p>
                  </div>
                </div>
              </div>
            ) : null}

            <ErrorMessage message={error} />

            <div className="data-card__footer flex flex-col gap-3 sm:flex-row">
              <Button type="submit" disabled={submitting}>
                {submitting ? <Spinner size="sm" className="mr-2" /> : null}
                {submitting ? 'Creating...' : 'Create Ticket'}
              </Button>
              <Button type="button" variant="outline" asChild disabled={submitting}>
                <Link href={backHref}>Cancel</Link>
              </Button>
            </div>
          </form>
        </section>

        <aside className="flex flex-col gap-6">
          <section className="data-card">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Intake checklist</h2>
                <p className="section-subtitle">Small details here save time later in the queue.</p>
              </div>
            </div>
            <div className="data-card__body space-y-2">
              {requestGuidance.map((item) => (
                <p key={item} className="text-sm leading-6 text-muted-foreground">
                  {item}
                </p>
              ))}
            </div>
          </section>

          <section className="data-card">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Routing context</h2>
                <p className="section-subtitle">
                  {loadingCategories
                    ? 'Loading categories for this environment.'
                    : `${categories.length} categories are available for routing right now.`}
                </p>
              </div>
            </div>
            <div className="data-card__body space-y-4">
              <div>
                <p className="text-sm font-medium text-foreground">Priority defaults to medium</p>
                <p className="mt-1 text-sm leading-6 text-muted-foreground">
                  Raise priority only when impact, urgency, or business risk clearly warrants it.
                </p>
              </div>
              <hr className="hr-soft" />
              <div>
                <p className="text-sm font-medium text-foreground">Description drives triage quality</p>
                <p className="mt-1 text-sm leading-6 text-muted-foreground">
                  The clearer the symptoms and impact, the better the downstream AI hints and human routing decisions.
                </p>
              </div>
            </div>
          </section>
        </aside>
      </div>
    </div>
  );
}
