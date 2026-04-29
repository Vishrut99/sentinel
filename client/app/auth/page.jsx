'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/context/AuthContext';
import { login as apiLogin, register as apiRegister, bootstrapAdmin } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ErrorMessage } from '@/components/sentinel';
import { Spinner } from '@/components/ui/spinner';
import { CheckCircle2, ChevronDown, Shield, Headphones, UserRound } from 'lucide-react';

const DEMO_ACCOUNTS = [
  { role: 'admin', label: 'Admin', email: 'olivia.brooks@northstar-demo.com', icon: Shield },
  { role: 'agent', label: 'Agent', email: 'maya.chen@northstar-demo.com', icon: Headphones },
  { role: 'user', label: 'User', email: 'daniel.kim@northstar-demo.com', icon: UserRound },
];
const DEMO_PASSWORD = 'DemoPass123!';

export default function AuthPage() {
  const { isAuthenticated, user, login } = useAuth();
  const router = useRouter();
  const [activeTab, setActiveTab] = useState('login');
  const [showBootstrap, setShowBootstrap] = useState(false);

  const [loginEmail, setLoginEmail] = useState('');
  const [loginPassword, setLoginPassword] = useState('');
  const [loginLoading, setLoginLoading] = useState(false);
  const [loginError, setLoginError] = useState('');

  const [registerEmail, setRegisterEmail] = useState('');
  const [registerPassword, setRegisterPassword] = useState('');
  const [registerFullName, setRegisterFullName] = useState('');
  const [registerLoading, setRegisterLoading] = useState(false);
  const [registerError, setRegisterError] = useState('');
  const [registerSuccess, setRegisterSuccess] = useState(false);

  const [bootstrapEmail, setBootstrapEmail] = useState('');
  const [bootstrapPassword, setBootstrapPassword] = useState('');
  const [bootstrapFullName, setBootstrapFullName] = useState('');
  const [bootstrapSecret, setBootstrapSecret] = useState('');
  const [bootstrapLoading, setBootstrapLoading] = useState(false);
  const [bootstrapError, setBootstrapError] = useState('');

  useEffect(() => {
    if (isAuthenticated && user) {
      if (user.role === 'user') {
        router.push('/tickets');
      } else {
        router.push('/dashboard');
      }
    }
  }, [isAuthenticated, user, router]);

  const handleLogin = async (e) => {
    e.preventDefault();
    setLoginError('');

    if (!loginEmail || !loginPassword) {
      setLoginError('Please enter email and password');
      return;
    }

    setLoginLoading(true);
    try {
      const response = await apiLogin(loginEmail, loginPassword);
      login(response.token, response.user);
    } catch (err) {
      setLoginError(err.message || 'Login failed');
    } finally {
      setLoginLoading(false);
    }
  };

  const handleDemoLogin = async (email) => {
    setLoginError('');
    setLoginEmail(email);
    setLoginPassword(DEMO_PASSWORD);
    setLoginLoading(true);
    try {
      const response = await apiLogin(email, DEMO_PASSWORD);
      login(response.token, response.user);
    } catch (err) {
      setLoginError(err.message || 'Login failed');
    } finally {
      setLoginLoading(false);
    }
  };

  const handleRegister = async (e) => {
    e.preventDefault();
    setRegisterError('');
    setRegisterSuccess(false);

    if (!registerEmail || !registerPassword || !registerFullName) {
      setRegisterError('Please fill in all fields');
      return;
    }

    setRegisterLoading(true);
    try {
      await apiRegister(registerEmail, registerPassword, registerFullName);
      setRegisterSuccess(true);
      setRegisterEmail('');
      setRegisterPassword('');
      setRegisterFullName('');
      setTimeout(() => {
        setActiveTab('login');
        setRegisterSuccess(false);
      }, 2000);
    } catch (err) {
      setRegisterError(err.message || 'Registration failed');
    } finally {
      setRegisterLoading(false);
    }
  };

  const handleBootstrap = async (e) => {
    e.preventDefault();
    setBootstrapError('');

    if (!bootstrapEmail || !bootstrapPassword || !bootstrapFullName || !bootstrapSecret) {
      setBootstrapError('Please fill in all fields');
      return;
    }

    setBootstrapLoading(true);
    try {
      const response = await bootstrapAdmin(
        bootstrapEmail,
        bootstrapPassword,
        bootstrapFullName,
        bootstrapSecret,
      );
      login(response.token, response.user);
    } catch (err) {
      setBootstrapError(err.message || 'Bootstrap failed');
    } finally {
      setBootstrapLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-10">
      <div className="w-full max-w-[460px]">
        <div className="mb-6 text-center">
          <span className="page-header__eyebrow">Secure access</span>
          <h1 className="page-header__title">Sign in to ITSM</h1>
          <p className="mt-1.5 text-sm text-muted-foreground">
            Jump into your queue or create a new requester account.
          </p>
        </div>

        <div className="data-card data-card--lg">
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="mb-5 grid w-full grid-cols-2">
              <TabsTrigger value="login">Login</TabsTrigger>
              <TabsTrigger value="register">Register</TabsTrigger>
            </TabsList>

            <TabsContent value="login" className="mt-0">
              <div className="mb-5">
                <div className="mb-2 flex items-center justify-between">
                  <span className="form-label">Quick demo login</span>
                  <span className="chip chip--warning">Dev only</span>
                </div>
                <div className="grid grid-cols-3 gap-2">
                  {DEMO_ACCOUNTS.map((acc) => (
                    <Button
                      key={acc.role}
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={loginLoading}
                      onClick={() => handleDemoLogin(acc.email)}
                      className="flex-col gap-1 h-auto py-2.5"
                    >
                      <acc.icon className="h-4 w-4" />
                      <span className="text-xs font-medium">{acc.label}</span>
                    </Button>
                  ))}
                </div>
                <div className="my-4 flex items-center gap-3">
                  <div className="h-px flex-1 bg-border" />
                  <span className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">or sign in manually</span>
                  <div className="h-px flex-1 bg-border" />
                </div>
              </div>
              <form onSubmit={handleLogin} className="space-y-4">
                <div className="space-y-1.5">
                  <Label htmlFor="login-email" className="form-label">Email</Label>
                  <Input
                    id="login-email"
                    type="email"
                    placeholder="you@example.com"
                    value={loginEmail}
                    onChange={(e) => setLoginEmail(e.target.value)}
                    disabled={loginLoading}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="login-password" className="form-label">Password</Label>
                  <Input
                    id="login-password"
                    type="password"
                    placeholder="Enter your password"
                    value={loginPassword}
                    onChange={(e) => setLoginPassword(e.target.value)}
                    disabled={loginLoading}
                  />
                </div>
                <ErrorMessage message={loginError} />
                <Button type="submit" className="w-full" disabled={loginLoading}>
                  {loginLoading ? <Spinner className="mr-2" /> : null}
                  {loginLoading ? 'Signing in...' : 'Sign In'}
                </Button>
              </form>
            </TabsContent>

            <TabsContent value="register" className="mt-0">
              <form onSubmit={handleRegister} className="space-y-4">
                <div className="space-y-1.5">
                  <Label htmlFor="register-name" className="form-label">Full Name</Label>
                  <Input
                    id="register-name"
                    type="text"
                    placeholder="Enter your full name"
                    value={registerFullName}
                    onChange={(e) => setRegisterFullName(e.target.value)}
                    disabled={registerLoading}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="register-email" className="form-label">Email</Label>
                  <Input
                    id="register-email"
                    type="email"
                    placeholder="you@example.com"
                    value={registerEmail}
                    onChange={(e) => setRegisterEmail(e.target.value)}
                    disabled={registerLoading}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="register-password" className="form-label">Password</Label>
                  <Input
                    id="register-password"
                    type="password"
                    placeholder="Create a password"
                    value={registerPassword}
                    onChange={(e) => setRegisterPassword(e.target.value)}
                    disabled={registerLoading}
                  />
                </div>
                <ErrorMessage message={registerError} />
                {registerSuccess ? (
                  <div className="flex items-start gap-2 rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800">
                    <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" />
                    Registration successful. Redirecting to the login tab...
                  </div>
                ) : null}
                <Button type="submit" className="w-full" disabled={registerLoading}>
                  {registerLoading ? <Spinner className="mr-2" /> : null}
                  {registerLoading ? 'Creating account...' : 'Create Account'}
                </Button>
              </form>
            </TabsContent>
          </Tabs>
        </div>

        <Collapsible open={showBootstrap} onOpenChange={setShowBootstrap} className="mt-5">
          <CollapsibleTrigger asChild>
            <button className="flex w-full items-center justify-center gap-2 text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground transition-colors hover:text-foreground">
              Bootstrap admin access
              <ChevronDown
                className={`h-3.5 w-3.5 transition-transform ${showBootstrap ? 'rotate-180' : ''}`}
              />
            </button>
          </CollapsibleTrigger>

          <CollapsibleContent className="mt-4">
            <div className="data-card data-card--lg">
              <div className="data-card__header">
                <div>
                  <h2 className="data-card__title">Bootstrap the first admin</h2>
                  <p className="data-card__subtitle">
                    Use the bootstrap secret to create the first privileged account.
                  </p>
                </div>
              </div>
              <div className="data-card__body">
                <form onSubmit={handleBootstrap} className="space-y-4">
                  <div className="space-y-1.5">
                    <Label htmlFor="bootstrap-name" className="form-label">Full Name</Label>
                    <Input
                      id="bootstrap-name"
                      type="text"
                      placeholder="Admin name"
                      value={bootstrapFullName}
                      onChange={(e) => setBootstrapFullName(e.target.value)}
                      disabled={bootstrapLoading}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="bootstrap-email" className="form-label">Email</Label>
                    <Input
                      id="bootstrap-email"
                      type="email"
                      placeholder="admin@example.com"
                      value={bootstrapEmail}
                      onChange={(e) => setBootstrapEmail(e.target.value)}
                      disabled={bootstrapLoading}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="bootstrap-password" className="form-label">Password</Label>
                    <Input
                      id="bootstrap-password"
                      type="password"
                      placeholder="Create a secure password"
                      value={bootstrapPassword}
                      onChange={(e) => setBootstrapPassword(e.target.value)}
                      disabled={bootstrapLoading}
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="bootstrap-secret" className="form-label">Bootstrap Secret</Label>
                    <Input
                      id="bootstrap-secret"
                      type="password"
                      placeholder="Enter the bootstrap secret"
                      value={bootstrapSecret}
                      onChange={(e) => setBootstrapSecret(e.target.value)}
                      disabled={bootstrapLoading}
                    />
                  </div>
                  <ErrorMessage message={bootstrapError} />
                  <Button type="submit" className="w-full" disabled={bootstrapLoading}>
                    {bootstrapLoading ? <Spinner className="mr-2" /> : null}
                    {bootstrapLoading ? 'Bootstrapping...' : 'Create Admin Account'}
                  </Button>
                </form>
              </div>
            </div>
          </CollapsibleContent>
        </Collapsible>
      </div>
    </div>
  );
}
