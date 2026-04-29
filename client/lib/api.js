const BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ||
  process.env.VITE_API_BASE_URL ||
  'http://localhost:8080/api/v1';

const TOKEN_KEY = 'sentinel_token';
const USER_KEY = 'sentinel_user';
export const FRONTEND_EDITABLE_TICKET_STATUSES = ['open', 'in_progress', 'resolved'];
const FRONTEND_EDITABLE_TICKET_STATUS_SET = new Set(FRONTEND_EDITABLE_TICKET_STATUSES);

// Helper to get stored token
export function getStoredToken() {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(TOKEN_KEY);
}

// Helper to get stored user
export function getStoredUser() {
  if (typeof window === 'undefined') return null;
  const user = localStorage.getItem(USER_KEY);
  return user ? JSON.parse(user) : null;
}

// Helper to store auth data
export function storeAuth(token, user) {
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(USER_KEY, JSON.stringify(user));
}

// Helper to clear auth data
export function clearAuth() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(USER_KEY);
}

// Main API request function
async function apiRequest(endpoint, options = {}) {
  const token = getStoredToken();
  
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  
  const response = await fetch(`${BASE_URL}${endpoint}`, {
    ...options,
    headers,
  });
  
  // Handle 401 - auto logout
  if (response.status === 401) {
    clearAuth();
    if (typeof window !== 'undefined') {
      window.location.href = '/auth';
    }
    throw new Error('Unauthorized');
  }
  
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || error.error || 'Request failed');
  }
  
  // Handle 204 No Content
  if (response.status === 204) {
    return null;
  }
  
  return response.json();
}

// Auth API
export async function login(email, password) {
  return apiRequest('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export async function register(email, password, fullName) {
  return apiRequest('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, password, full_name: fullName }),
  });
}

export async function bootstrapAdmin(email, password, fullName, bootstrapSecret) {
  return apiRequest('/auth/bootstrap-admin', {
    method: 'POST',
    body: JSON.stringify({ 
      email, 
      password, 
      full_name: fullName, 
      bootstrap_secret: bootstrapSecret 
    }),
  });
}

// Tickets API
export async function getTickets(params = {}) {
  const searchParams = new URLSearchParams();
  if (params.q) searchParams.set('q', params.q);
  if (params.status) searchParams.set('status', params.status);
  if (params.priority) searchParams.set('priority', params.priority);
  if (params.category_id) searchParams.set('category_id', params.category_id);
  if (params.assigned_agent_id) searchParams.set('assigned_agent_id', params.assigned_agent_id);
  if (params.is_problem !== undefined) searchParams.set('is_problem', params.is_problem);
  if (params.sla_breached !== undefined) searchParams.set('sla_breached', params.sla_breached);
  if (params.sort_by) searchParams.set('sort_by', params.sort_by);
  if (params.sort_dir) searchParams.set('sort_dir', params.sort_dir);
  if (params.page) searchParams.set('page', params.page);
  if (params.per_page) searchParams.set('per_page', params.per_page || 10);
  
  const query = searchParams.toString();
  return apiRequest(`/tickets${query ? `?${query}` : ''}`);
}

export async function getTicket(id) {
  return apiRequest(`/tickets/${id}`);
}

export async function createTicket(data) {
  return apiRequest('/tickets', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateTicket(ticketId, data) {
  return apiRequest(`/tickets/${ticketId}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
}

export async function assignTicket(ticketId, agentId) {
  return apiRequest(`/tickets/${ticketId}/assign`, {
    method: 'PATCH',
    body: JSON.stringify({ agent_id: agentId }),
  });
}

export async function updateTicketStatus(ticketId, status) {
  if (!FRONTEND_EDITABLE_TICKET_STATUS_SET.has(status)) {
    throw new Error('This status change is not available in the frontend.');
  }

  return apiRequest(`/tickets/${ticketId}/status`, {
    method: 'PATCH',
    body: JSON.stringify({ status }),
  });
}

export async function linkProblem(ticketId, parentId) {
  return apiRequest(`/tickets/${ticketId}/link-problem`, {
    method: 'PATCH',
    body: JSON.stringify({ parent_id: parentId }),
  });
}

// Comments API
export async function getComments(ticketId) {
  const response = await apiRequest(`/tickets/${ticketId}/comments`);
  // Normalize PascalCase to camelCase
  if (Array.isArray(response)) {
    return response.map(normalizeComment);
  }
  if (response?.data && Array.isArray(response.data)) {
    return response.data.map(normalizeComment);
  }
  return response;
}

function normalizeComment(c) {
  return {
    id: c.ID ?? c.id,
    ticketId: c.TicketID ?? c.ticket_id,
    authorId: c.AuthorID ?? c.author_id,
    body: c.Body ?? c.body,
    isInternal: c.IsInternal ?? c.is_internal ?? false,
    createdAt: c.CreatedAt ?? c.created_at,
    author: c.Author
      ? {
          id: c.Author.ID ?? c.Author.id,
          email: c.Author.Email ?? c.Author.email,
          fullName: c.Author.FullName ?? c.Author.full_name,
          role: c.Author.Role ?? c.Author.role,
        }
      : c.author,
  };
}

export async function createComment(ticketId, body, isInternal = false) {
  return apiRequest(`/tickets/${ticketId}/comments`, {
    method: 'POST',
    body: JSON.stringify({ body, is_internal: isInternal }),
  });
}

// Audit API
export async function getAuditLog(ticketId) {
  return apiRequest(`/tickets/${ticketId}/audit`);
}

// Dashboard API
export async function getDashboard(params = {}) {
  const searchParams = new URLSearchParams();
  if (params.q) searchParams.set('q', params.q);
  if (params.status) searchParams.set('status', params.status);
  if (params.priority) searchParams.set('priority', params.priority);
  if (params.category_id) searchParams.set('category_id', params.category_id);
  if (params.assigned_agent_id) searchParams.set('assigned_agent_id', params.assigned_agent_id);
  if (params.is_problem !== undefined) searchParams.set('is_problem', params.is_problem);
  if (params.sla_breached !== undefined) searchParams.set('sla_breached', params.sla_breached);

  const query = searchParams.toString();
  return apiRequest(`/dashboard${query ? `?${query}` : ''}`);
}

// Agents API
export async function getAgents() {
  return apiRequest('/agents');
}

export async function registerAgent(userId, department, maxTickets = 10, skills = []) {
  return apiRequest('/agents/register', {
    method: 'POST',
    body: JSON.stringify({ user_id: userId, department, max_tickets: maxTickets, skills }),
  });
}

export async function updateAgentCapacity(agentId, isAvailable, maxTickets, skills) {
  const payload = {};
  if (isAvailable !== undefined) payload.is_available = isAvailable;
  if (maxTickets !== undefined) payload.max_tickets = maxTickets;
  if (skills !== undefined) payload.skills = skills;

  return apiRequest(`/agents/${agentId}/capacity`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  });
}

// Users API
export async function getUsers(params = {}) {
  const searchParams = new URLSearchParams();
  if (params.role) searchParams.set('role', params.role);
  if (params.q) searchParams.set('q', params.q);
  
  const query = searchParams.toString();
  return apiRequest(`/users${query ? `?${query}` : ''}`);
}

export async function promoteAdmin(userId) {
  return apiRequest(`/users/${userId}/promote-admin`, {
    method: 'PATCH',
  });
}

// Lookups API
export async function getCategories() {
  return apiRequest('/lookups/categories');
}

export async function getPriorities() {
  return apiRequest('/lookups/priorities');
}

export async function getStatuses() {
  return apiRequest('/lookups/statuses');
}
