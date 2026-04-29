'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { 
  getUsers, 
  getAgents, 
  promoteAdmin, 
  registerAgent, 
  updateAgentCapacity,
  bootstrapAdmin as apiBootstrapAdmin,
} from '@/lib/api';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { ErrorMessage, LoadingSpinner, EmptyState, TableSkeleton } from '@/components/sentinel';
import { Spinner } from '@/components/ui/spinner';
import { Badge } from '@/components/ui/badge';
import { Users, UserCog, Shield, Search, UserPlus } from 'lucide-react';

function formatSkillEditor(skills = []) {
  if (!Array.isArray(skills) || skills.length === 0) {
    return '';
  }

  return skills
    .map((skill) => `${skill.name || ''}:${skill.score ?? 70}`)
    .join(', ');
}

function parseSkillEditor(value) {
  const tokens = value
    .split(/\n|,/)
    .map((part) => part.trim())
    .filter(Boolean);

  const skills = [];
  for (const token of tokens) {
    const [rawName, rawScore] = token.split(':').map((part) => part.trim());
    if (!rawName) {
      return { skills: [], error: 'Each skill entry needs a name.' };
    }

    const parsedScore = rawScore === undefined || rawScore === '' ? 70 : Number(rawScore);
    if (!Number.isFinite(parsedScore) || parsedScore < 0 || parsedScore > 100) {
      return { skills: [], error: `Skill score for "${rawName}" must be between 0 and 100.` };
    }

    skills.push({ name: rawName, score: Math.round(parsedScore) });
  }

  return { skills, error: '' };
}

export default function AdminSettingsPage() {
  const { user, isAdmin, login, loading } = useAuth();
  const router = useRouter();
  
  // Users tab state
  const [users, setUsers] = useState([]);
  const [usersLoading, setUsersLoading] = useState(true);
  const [usersError, setUsersError] = useState('');
  const [userSearch, setUserSearch] = useState('');
  const [userRoleFilter, setUserRoleFilter] = useState('all');
  const [promotingUserId, setPromotingUserId] = useState(null);

  // Agents tab state
  const [agents, setAgents] = useState([]);
  const [agentsLoading, setAgentsLoading] = useState(true);
  const [agentsError, setAgentsError] = useState('');
  const [editingAgent, setEditingAgent] = useState(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editMaxTickets, setEditMaxTickets] = useState(10);
  const [editIsAvailable, setEditIsAvailable] = useState(true);
  const [editSkills, setEditSkills] = useState('');
  const [updatingAgent, setUpdatingAgent] = useState(false);

  // Register Agent tab state
  const [regularUsers, setRegularUsers] = useState([]);
  const [regularUsersLoading, setRegularUsersLoading] = useState(true);
  const [selectedUserId, setSelectedUserId] = useState('');
  const [newAgentDepartment, setNewAgentDepartment] = useState('');
  const [newAgentMaxTickets, setNewAgentMaxTickets] = useState(10);
  const [newAgentSkills, setNewAgentSkills] = useState('');
  const [registeringAgent, setRegisteringAgent] = useState(false);
  const [registerAgentError, setRegisterAgentError] = useState('');
  const [registerAgentSuccess, setRegisterAgentSuccess] = useState(false);

  // Bootstrap Admin tab state
  const [bootstrapEmail, setBootstrapEmail] = useState('');
  const [bootstrapPassword, setBootstrapPassword] = useState('');
  const [bootstrapFullName, setBootstrapFullName] = useState('');
  const [bootstrapSecret, setBootstrapSecret] = useState('');
  const [bootstrapping, setBootstrapping] = useState(false);
  const [bootstrapError, setBootstrapError] = useState('');

  // Redirect non-admins
  useEffect(() => {
    if (!loading && !isAdmin) {
      router.push('/dashboard');
    }
  }, [isAdmin, loading, router]);

  // Fetch users
  useEffect(() => {
    const fetchUsers = async () => {
      setUsersLoading(true);
      setUsersError('');
      try {
        const params = {};
        if (userRoleFilter && userRoleFilter !== 'all') params.role = userRoleFilter;
        if (userSearch) params.q = userSearch;
        
        const data = await getUsers(params);
        setUsers(Array.isArray(data) ? data : data?.data || []);
      } catch (err) {
        setUsersError(err.message || 'Failed to load users');
      } finally {
        setUsersLoading(false);
      }
    };
    fetchUsers();
  }, [userRoleFilter, userSearch]);

  // Fetch agents
  useEffect(() => {
    const fetchAgents = async () => {
      setAgentsLoading(true);
      setAgentsError('');
      try {
        const data = await getAgents();
        setAgents(Array.isArray(data) ? data : data?.data || []);
      } catch (err) {
        setAgentsError(err.message || 'Failed to load agents');
      } finally {
        setAgentsLoading(false);
      }
    };
    fetchAgents();
  }, []);

  // Fetch regular users for agent registration
  useEffect(() => {
    const fetchRegularUsers = async () => {
      setRegularUsersLoading(true);
      try {
        const data = await getUsers({ role: 'user' });
        setRegularUsers(Array.isArray(data) ? data : data?.data || []);
      } catch (err) {
        console.error('Failed to load users:', err);
      } finally {
        setRegularUsersLoading(false);
      }
    };
    fetchRegularUsers();
  }, []);

  const handlePromoteAdmin = async (userId) => {
    setPromotingUserId(userId);
    try {
      await promoteAdmin(userId);
      // Refresh users
      const params = {};
      if (userRoleFilter && userRoleFilter !== 'all') params.role = userRoleFilter;
      if (userSearch) params.q = userSearch;
      const data = await getUsers(params);
      setUsers(Array.isArray(data) ? data : data?.data || []);
    } catch (err) {
      setUsersError(err.message || 'Failed to promote user');
    } finally {
      setPromotingUserId(null);
    }
  };

  const handleEditAgent = (agent) => {
    setEditingAgent(agent);
    setEditMaxTickets(agent.max_tickets || 10);
    setEditIsAvailable(agent.is_available ?? true);
    setEditSkills(formatSkillEditor(agent.skills));
    setEditDialogOpen(true);
  };

  const handleUpdateAgent = async () => {
    if (!editingAgent) return;
    if (editMaxTickets < 1) {
      setAgentsError('Max tickets must be at least 1');
      return;
    }

    const { skills, error: skillError } = parseSkillEditor(editSkills);
    if (skillError) {
      setAgentsError(skillError);
      return;
    }
    
    setUpdatingAgent(true);
    try {
      await updateAgentCapacity(editingAgent.id, editIsAvailable, editMaxTickets, skills);
      // Refresh agents
      const data = await getAgents();
      setAgents(Array.isArray(data) ? data : data?.data || []);
      setEditDialogOpen(false);
      setEditingAgent(null);
    } catch (err) {
      setAgentsError(err.message || 'Failed to update agent');
    } finally {
      setUpdatingAgent(false);
    }
  };

  const handleRegisterAgent = async (e) => {
    e.preventDefault();
    setRegisterAgentError('');
    setRegisterAgentSuccess(false);
    
    if (!selectedUserId) {
      setRegisterAgentError('Please select a user');
      return;
    }
    if (newAgentDepartment.trim().length < 2) {
      setRegisterAgentError('Department must be at least 2 characters');
      return;
    }
    if (newAgentMaxTickets < 1) {
      setRegisterAgentError('Max tickets must be at least 1');
      return;
    }

    const { skills, error: skillError } = parseSkillEditor(newAgentSkills);
    if (skillError) {
      setRegisterAgentError(skillError);
      return;
    }

    setRegisteringAgent(true);
    try {
      await registerAgent(selectedUserId, newAgentDepartment.trim(), newAgentMaxTickets, skills);
      setRegisterAgentSuccess(true);
      setSelectedUserId('');
      setNewAgentDepartment('');
      setNewAgentMaxTickets(10);
      setNewAgentSkills('');
      
      // Refresh agents and users
      const [agentsData, usersData] = await Promise.all([
        getAgents(),
        getUsers({ role: 'user' }),
      ]);
      setAgents(Array.isArray(agentsData) ? agentsData : agentsData?.data || []);
      setRegularUsers(Array.isArray(usersData) ? usersData : usersData?.data || []);
    } catch (err) {
      setRegisterAgentError(err.message || 'Failed to register agent');
    } finally {
      setRegisteringAgent(false);
    }
  };

  const handleBootstrapAdmin = async (e) => {
    e.preventDefault();
    setBootstrapError('');
    
    if (!bootstrapEmail || !bootstrapPassword || !bootstrapFullName || !bootstrapSecret) {
      setBootstrapError('Please fill in all fields');
      return;
    }

    setBootstrapping(true);
    try {
      const response = await apiBootstrapAdmin(bootstrapEmail, bootstrapPassword, bootstrapFullName, bootstrapSecret);
      login(response.token, response.user);
    } catch (err) {
      setBootstrapError(err.message || 'Bootstrap failed');
    } finally {
      setBootstrapping(false);
    }
  };

  const getRoleBadgeVariant = (role) => {
    switch (role) {
      case 'admin': return 'default';
      case 'agent': return 'secondary';
      default: return 'outline';
    }
  };

  return (
    <div className="page-shell">
      <header className="page-header">
        <div className="min-w-0">
          <div className="page-header__eyebrow">Administration</div>
          <h1 className="page-header__title">Admin settings</h1>
          <p className="page-header__description">Manage users, agents, and system settings.</p>
        </div>
      </header>

      <Tabs defaultValue="users" className="flex flex-col gap-6">
        <TabsList className="grid w-full grid-cols-4 max-w-2xl">
          <TabsTrigger value="users" className="flex items-center gap-2">
            <Users className="h-4 w-4" />
            <span className="hidden sm:inline">Users</span>
          </TabsTrigger>
          <TabsTrigger value="register-agent" className="flex items-center gap-2">
            <UserPlus className="h-4 w-4" />
            <span className="hidden sm:inline">Register Agent</span>
          </TabsTrigger>
          <TabsTrigger value="agents" className="flex items-center gap-2">
            <UserCog className="h-4 w-4" />
            <span className="hidden sm:inline">Agents</span>
          </TabsTrigger>
          <TabsTrigger value="bootstrap" className="flex items-center gap-2">
            <Shield className="h-4 w-4" />
            <span className="hidden sm:inline">Bootstrap</span>
          </TabsTrigger>
        </TabsList>

        {/* Users Tab */}
        <TabsContent value="users">
          <section className="data-card data-card--lg">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">User management</h2>
                <p className="section-subtitle">View and manage user accounts.</p>
              </div>
            </div>
            <div className="data-card__body flex flex-col gap-4">
              {/* Filters */}
              <div className="filter-bar">
                <div className="relative flex-1 min-w-0">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search users..."
                    value={userSearch}
                    onChange={(e) => setUserSearch(e.target.value)}
                    className="pl-9"
                  />
                </div>
                <Select value={userRoleFilter} onValueChange={setUserRoleFilter}>
                  <SelectTrigger className="w-full sm:w-40">
                    <SelectValue placeholder="All roles" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Roles</SelectItem>
                    <SelectItem value="user">Users</SelectItem>
                    <SelectItem value="agent">Agents</SelectItem>
                    <SelectItem value="admin">Admins</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <ErrorMessage message={usersError} />

              {usersLoading ? (
                <TableSkeleton columns={4} rows={5} />
              ) : users.length === 0 ? (
                <EmptyState
                  icon="users"
                  title="No users found"
                  description="No users match your search criteria."
                />
              ) : (
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Name</TableHead>
                        <TableHead>Email</TableHead>
                        <TableHead>Role</TableHead>
                        <TableHead>Actions</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {users.map((u) => (
                        <TableRow key={u.id}>
                          <TableCell className="font-medium">{u.full_name}</TableCell>
                          <TableCell className="text-muted-foreground">{u.email}</TableCell>
                          <TableCell>
                            <Badge variant={getRoleBadgeVariant(u.role)}>{u.role}</Badge>
                          </TableCell>
                          <TableCell>
                            {u.role === 'user' && (
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => handlePromoteAdmin(u.id)}
                                disabled={promotingUserId === u.id}
                              >
                                {promotingUserId === u.id ? (
                                  <Spinner size="sm" className="mr-2" />
                                ) : null}
                                Promote to Admin
                              </Button>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </div>
          </section>
        </TabsContent>

        {/* Register Agent Tab */}
        <TabsContent value="register-agent">
          <section className="data-card data-card--lg">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Register new agent</h2>
                <p className="section-subtitle">Convert a regular user into a support agent.</p>
              </div>
            </div>
            <div className="data-card__body">
              <form onSubmit={handleRegisterAgent} className="flex flex-col gap-4">
                <div className="form-grid">
                  <div className="flex flex-col gap-2 min-w-0">
                    <Label htmlFor="select-user" className="form-label">Select user</Label>
                    {regularUsersLoading ? (
                      <Spinner size="sm" />
                    ) : (
                      <Select value={selectedUserId} onValueChange={setSelectedUserId}>
                        <SelectTrigger>
                          <SelectValue placeholder="Select a user to register as agent" />
                        </SelectTrigger>
                        <SelectContent>
                          {regularUsers.map((u) => (
                            <SelectItem key={u.id} value={u.id}>
                              {u.full_name} ({u.email})
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    )}
                  </div>

                  <div className="flex flex-col gap-2 min-w-0">
                    <Label htmlFor="department" className="form-label">Department</Label>
                    <Input
                      id="department"
                      placeholder="e.g., Technical Support"
                      value={newAgentDepartment}
                      onChange={(e) => setNewAgentDepartment(e.target.value)}
                    />
                  </div>

                  <div className="flex flex-col gap-2 min-w-0">
                    <Label htmlFor="max-tickets" className="form-label">Max tickets</Label>
                    <Input
                      id="max-tickets"
                      type="number"
                      min={1}
                      value={newAgentMaxTickets}
                      onChange={(e) => setNewAgentMaxTickets(parseInt(e.target.value) || 10)}
                    />
                    <p className="text-xs text-muted-foreground">
                      Maximum number of tickets this agent can handle at once.
                    </p>
                  </div>
                </div>

                <div className="flex flex-col gap-2 min-w-0">
                  <Label htmlFor="agent-skills" className="form-label">Skills</Label>
                  <Textarea
                    id="agent-skills"
                    rows={3}
                    placeholder="VPN:80, Network:72, Account Access:76"
                    value={newAgentSkills}
                    onChange={(e) => setNewAgentSkills(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    Enter comma-separated skills as `name:score`. Scores default to 70 if omitted.
                  </p>
                </div>

                <ErrorMessage message={registerAgentError} />

                {registerAgentSuccess && (
                  <div className="chip chip--success">Agent registered successfully!</div>
                )}

                <div>
                  <Button type="submit" disabled={registeringAgent || !selectedUserId}>
                    {registeringAgent ? <Spinner size="sm" className="mr-2" /> : null}
                    Register agent
                  </Button>
                </div>
              </form>
            </div>
          </section>
        </TabsContent>

        {/* Agents Tab */}
        <TabsContent value="agents">
          <section className="data-card data-card--lg">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Agent management</h2>
                <p className="section-subtitle">Manage agent capacity and availability.</p>
              </div>
            </div>
            <div className="data-card__body flex flex-col gap-4">
              <ErrorMessage message={agentsError} />

              {agentsLoading ? (
                <TableSkeleton columns={6} rows={5} />
              ) : agents.length === 0 ? (
                <EmptyState
                  icon="users"
                  title="No agents found"
                  description="No agents have been registered yet."
                />
              ) : (
                <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Department</TableHead>
                      <TableHead>Skills</TableHead>
                      <TableHead>Load</TableHead>
                      <TableHead>Available</TableHead>
                      <TableHead>Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {agents.map((agent) => (
                      <TableRow key={agent.id}>
                        <TableCell className="font-medium">
                          {agent.user?.full_name || agent.full_name || 'Unknown'}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {agent.department || 'N/A'}
                        </TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-1">
                            {(agent.skills || []).length === 0 ? (
                              <span className="text-xs text-muted-foreground">No skills yet</span>
                            ) : (
                              (agent.skills || []).slice(0, 3).map((skill) => (
                                <Badge key={`${agent.id}-${skill.name}`} variant="outline">
                                  {skill.name} {skill.score}
                                </Badge>
                              ))
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {agent.active_tickets ?? 0} / {agent.max_tickets || 10}
                        </TableCell>
                        <TableCell>
                          <Badge variant={agent.is_available ? 'default' : 'secondary'}>
                            {agent.is_available ? 'Available' : 'Unavailable'}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Button variant="outline" size="sm" onClick={() => handleEditAgent(agent)}>
                            Edit
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                </div>
              )}

              {/* Edit Agent Dialog */}
              <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Edit Agent Profile</DialogTitle>
                    <DialogDescription>
                      Update availability, ticket capacity, and skill profile for {editingAgent?.user?.full_name || editingAgent?.full_name}
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 py-4">
                    <div className="flex items-center justify-between">
                      <Label htmlFor="edit-available">Available</Label>
                      <Switch
                        id="edit-available"
                        checked={editIsAvailable}
                        onCheckedChange={setEditIsAvailable}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-max-tickets">Max Tickets</Label>
                      <Input
                        id="edit-max-tickets"
                        type="number"
                        min={1}
                        value={editMaxTickets}
                        onChange={(e) => setEditMaxTickets(parseInt(e.target.value) || 10)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="edit-skills">Skills</Label>
                      <Textarea
                        id="edit-skills"
                        rows={4}
                        value={editSkills}
                        onChange={(e) => setEditSkills(e.target.value)}
                        placeholder="VPN:80, Network:72, Account Access:76"
                      />
                      <p className="text-xs text-muted-foreground">
                        Enter comma-separated skills as `name:score`. Scores default to 70 if omitted.
                      </p>
                    </div>
                  </div>
                  <DialogFooter>
                    <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
                      Cancel
                    </Button>
                    <Button onClick={handleUpdateAgent} disabled={updatingAgent}>
                      {updatingAgent ? <Spinner size="sm" className="mr-2" /> : null}
                      Save Changes
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
          </section>
        </TabsContent>

        {/* Bootstrap Admin Tab */}
        <TabsContent value="bootstrap">
          <section className="data-card data-card--lg">
            <div className="data-card__header">
              <div className="min-w-0">
                <h2 className="section-title">Bootstrap admin</h2>
                <p className="section-subtitle">Create a new admin account with the bootstrap secret.</p>
              </div>
            </div>
            <div className="data-card__body">
              <form onSubmit={handleBootstrapAdmin} className="flex flex-col gap-4">
                <div className="form-grid">
                  <div className="flex flex-col gap-2 min-w-0">
                    <Label htmlFor="bs-name" className="form-label">Full name</Label>
                    <Input
                      id="bs-name"
                      placeholder="Admin name"
                      value={bootstrapFullName}
                      onChange={(e) => setBootstrapFullName(e.target.value)}
                      disabled={bootstrapping}
                    />
                  </div>
                  <div className="flex flex-col gap-2 min-w-0">
                    <Label htmlFor="bs-email" className="form-label">Email</Label>
                    <Input
                      id="bs-email"
                      type="email"
                      placeholder="Admin email"
                      value={bootstrapEmail}
                      onChange={(e) => setBootstrapEmail(e.target.value)}
                      disabled={bootstrapping}
                    />
                  </div>
                  <div className="flex flex-col gap-2 min-w-0">
                    <Label htmlFor="bs-password" className="form-label">Password</Label>
                    <Input
                      id="bs-password"
                      type="password"
                      placeholder="Admin password"
                      value={bootstrapPassword}
                      onChange={(e) => setBootstrapPassword(e.target.value)}
                      disabled={bootstrapping}
                    />
                  </div>
                  <div className="flex flex-col gap-2 min-w-0">
                    <Label htmlFor="bs-secret" className="form-label">Bootstrap secret</Label>
                    <Input
                      id="bs-secret"
                      type="password"
                      placeholder="Enter bootstrap secret"
                      value={bootstrapSecret}
                      onChange={(e) => setBootstrapSecret(e.target.value)}
                      disabled={bootstrapping}
                    />
                  </div>
                </div>

                <ErrorMessage message={bootstrapError} />

                <div>
                  <Button type="submit" disabled={bootstrapping}>
                    {bootstrapping ? <Spinner size="sm" className="mr-2" /> : null}
                    Create admin
                  </Button>
                </div>
              </form>
            </div>
          </section>
        </TabsContent>
      </Tabs>
    </div>
  );
}
