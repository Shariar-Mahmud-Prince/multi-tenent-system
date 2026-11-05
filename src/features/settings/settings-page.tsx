import * as React from "react"
import {
  Key,
  CreditCard,
  Copy,
  Plus,
  Trash2,
  Smartphone,
  Laptop,
} from "lucide-react"
import { PageHeader } from "@/components/shared/page-header"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { toast } from "sonner"

export function SettingsPage() {
  // General form state
  const [workspaceName, setWorkspaceName] = React.useState("Nexus HQ")
  const [subdomain, setSubdomain] = React.useState("nexus-cloud")
  const [supportEmail, setSupportEmail] = React.useState("support@nexus.io")

  // API Keys state
  const [apiKeys, setApiKeys] = React.useState([
    {
      id: "key_1",
      name: "Production Backend Key",
      prefix: "sk_live_99a8b7...4f8",
      created: "2024-03-12",
      lastUsed: "2 mins ago",
    },
    {
      id: "key_2",
      name: "Staging CI/CD Runner",
      prefix: "sk_test_12c3d4...9e1",
      created: "2024-06-20",
      lastUsed: "1 day ago",
    },
  ])

  // Security toggles
  const [twoFactorRequired, setTwoFactorRequired] = React.useState(true)
  const [samlEnforced, setSamlEnforced] = React.useState(false)
  const [sessionTimeout, setSessionTimeout] = React.useState(true)

  // Notification toggles
  const [emailAlerts, setEmailAlerts] = React.useState(true)
  const [slackSync, setSlackSync] = React.useState(true)
  const [weeklyDigest, setWeeklyDigest] = React.useState(false)

  const handleSaveGeneral = (e: React.FormEvent) => {
    e.preventDefault()
    toast.success("Workspace profile updated successfully")
  }

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success("Copied to clipboard")
  }

  const handleCreateKey = () => {
    const newKey = {
      id: `key_${Date.now()}`,
      name: `API Key ${apiKeys.length + 1}`,
      prefix: `sk_live_${Math.random().toString(36).substring(2, 8)}...${Math.random().toString(36).substring(2, 5)}`,
      created: new Date().toISOString().split("T")[0],
      lastUsed: "Just now",
    }
    setApiKeys([...apiKeys, newKey])
    toast.success("Generated new production API secret key")
  }

  const handleRevokeKey = (id: string) => {
    setApiKeys(apiKeys.filter((k) => k.id !== id))
    toast.success("API key revoked immediately")
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Settings"
        description="Manage workspace identity, API credentials, enterprise authentication, and billing configuration."
      />

      <Tabs defaultValue="general" className="space-y-4">
        <TabsList className="grid w-full grid-cols-2 md:grid-cols-5 h-auto">
          <TabsTrigger value="general">General</TabsTrigger>
          <TabsTrigger value="apikeys">API Keys</TabsTrigger>
          <TabsTrigger value="security">Security</TabsTrigger>
          <TabsTrigger value="billing">Billing</TabsTrigger>
          <TabsTrigger value="notifications">Notifications</TabsTrigger>
        </TabsList>

        {/* 1. General Settings */}
        <TabsContent value="general" className="space-y-4">
          <Card>
            <form onSubmit={handleSaveGeneral}>
              <CardHeader>
                <CardTitle>Workspace Profile</CardTitle>
                <CardDescription>
                  This information is displayed across invoices and team invitations.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4 max-w-xl">
                <div className="space-y-2">
                  <Label htmlFor="ws-name">Workspace Display Name</Label>
                  <Input
                    id="ws-name"
                    value={workspaceName}
                    onChange={(e) => setWorkspaceName(e.target.value)}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="ws-subdomain">Custom Subdomain</Label>
                  <div className="flex items-center">
                    <Input
                      id="ws-subdomain"
                      value={subdomain}
                      onChange={(e) => setSubdomain(e.target.value)}
                      className="rounded-r-none border-r-0"
                    />
                    <span className="flex h-9 items-center px-3 border border-input bg-muted text-xs text-muted-foreground rounded-r-md">
                      .nexus.io
                    </span>
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="ws-email">Support & Billing Email</Label>
                  <Input
                    id="ws-email"
                    type="email"
                    value={supportEmail}
                    onChange={(e) => setSupportEmail(e.target.value)}
                  />
                </div>
              </CardContent>
              <CardFooter className="border-t pt-4 flex justify-end">
                <Button type="submit">
                  Save Changes
                </Button>
              </CardFooter>
            </form>
          </Card>
        </TabsContent>

        {/* 2. API Keys */}
        <TabsContent value="apikeys" className="space-y-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>Secret API Credentials</CardTitle>
                <CardDescription>
                  Secret keys allow server-side operations with full workspace authority. Keep them confidential.
                </CardDescription>
              </div>
              <Button size="sm" onClick={handleCreateKey} className="gap-1.5">
                <Plus className="h-4 w-4" />
                <span>Create Secret Key</span>
              </Button>
            </CardHeader>
            <CardContent className="space-y-3">
              {apiKeys.map((key) => (
                <div
                  key={key.id}
                  className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-4 rounded-lg border"
                >
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <Key className="h-4 w-4 text-muted-foreground" />
                      <span className="font-semibold text-sm">{key.name}</span>
                      <Badge variant="outline" className="text-xs">Active</Badge>
                    </div>
                    <div className="font-mono text-xs text-muted-foreground">{key.prefix}</div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground mr-2">Used {key.lastUsed}</span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleCopy(key.prefix)}
                    >
                      <Copy className="h-3.5 w-3.5 mr-1" />
                      <span>Copy</span>
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-destructive hover:text-destructive"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Revoke API Key</AlertDialogTitle>
                          <AlertDialogDescription>
                            Are you sure you want to revoke {key.name}? Any application or background job using this key will immediately be denied access.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => handleRevokeKey(key.id)}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                          >
                            Revoke Key
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        </TabsContent>

        {/* 3. Security */}
        <TabsContent value="security" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Authentication & Policy Enforcement</CardTitle>
              <CardDescription>
                Strengthen workspace integrity and session rules for all collaborators.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-sm">Enforce 2FA for all members</Label>
                  <p className="text-xs text-muted-foreground">
                    Require TOTP authenticator app verification upon login.
                  </p>
                </div>
                <Switch
                  checked={twoFactorRequired}
                  onCheckedChange={(c) => {
                    setTwoFactorRequired(c)
                    toast.success(`Two-factor enforcement ${c ? "enabled" : "disabled"}`)
                  }}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-sm">SAML 2.0 Single Sign-On</Label>
                  <p className="text-xs text-muted-foreground">
                    Route enterprise logins through corporate identity provider.
                  </p>
                </div>
                <Switch
                  checked={samlEnforced}
                  onCheckedChange={(c) => {
                    setSamlEnforced(c)
                    toast.info(`SAML SSO ${c ? "activated" : "deactivated"}`)
                  }}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-sm">Idle Session Invalidation</Label>
                  <p className="text-xs text-muted-foreground">
                    Automatically terminate dashboard sessions inactive after 30 minutes.
                  </p>
                </div>
                <Switch
                  checked={sessionTimeout}
                  onCheckedChange={(c) => {
                    setSessionTimeout(c)
                    toast.success(`Session timeout ${c ? "enabled" : "disabled"}`)
                  }}
                />
              </div>
            </CardContent>
          </Card>

          {/* Active Sessions */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>Active Sign-In Sessions</CardTitle>
                <CardDescription>
                  Devices currently authorized with an active session token.
                </CardDescription>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => toast.success("Terminated other sessions")}
              >
                Revoke Other Sessions
              </Button>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between p-3 rounded-lg border">
                <div className="flex items-center gap-3">
                  <Laptop className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <div className="text-sm font-medium">macOS · Chrome 132.0 (Current)</div>
                    <div className="text-xs text-muted-foreground">San Francisco, CA · IP: 192.168.1.104</div>
                  </div>
                </div>
                <Badge variant="secondary">This device</Badge>
              </div>

              <div className="flex items-center justify-between p-3 rounded-lg border">
                <div className="flex items-center gap-3">
                  <Smartphone className="h-5 w-5 text-muted-foreground" />
                  <div>
                    <div className="text-sm font-medium">iPhone 16 Pro · Nexus Mobile App</div>
                    <div className="text-xs text-muted-foreground">Austin, TX · IP: 172.56.21.90 · Active 4h ago</div>
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => toast.success("Revoked iPhone session")}
                  className="text-xs text-destructive hover:text-destructive"
                >
                  Revoke
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 4. Billing */}
        <TabsContent value="billing" className="space-y-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>Subscription Plan</CardTitle>
                <CardDescription>
                  Your workspace is on the Enterprise Plus tier.
                </CardDescription>
              </div>
              <Badge variant="default">
                Enterprise Plus
              </Badge>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="rounded-lg border p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div>
                  <div className="text-xl font-bold">$1,499.00 <span className="text-xs text-muted-foreground font-normal">/ month</span></div>
                  <p className="text-xs text-muted-foreground mt-0.5">Next billing date: December 4, 2026</p>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => toast.info("Opening tier selector...")}
                  >
                    Change Tier
                  </Button>
                  <Button
                    size="sm"
                    onClick={() => toast.success("Stripe customer billing portal launched")}
                  >
                    Customer Portal
                  </Button>
                </div>
              </div>

              <div className="rounded-lg border p-4 space-y-2">
                <div className="text-sm font-medium">Payment Method</div>
                <div className="flex items-center justify-between text-sm">
                  <div className="flex items-center gap-2">
                    <CreditCard className="h-4 w-4 text-muted-foreground" />
                    <span className="font-mono text-xs">Visa ending in 4242</span>
                    <span className="text-xs text-muted-foreground">Exp: 12/28</span>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => toast.info("Update card modal")}
                    className="text-xs"
                  >
                    Update
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 5. Notifications */}
        <TabsContent value="notifications" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Notification Preferences</CardTitle>
              <CardDescription>
                Control the channel and frequency of automated reports and operational alerts.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-sm">Immediate Transaction Alerts</Label>
                  <p className="text-xs text-muted-foreground">
                    Receive email notification whenever a transaction &gt; $1,000 occurs.
                  </p>
                </div>
                <Switch checked={emailAlerts} onCheckedChange={setEmailAlerts} />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-sm">Slack Incident Bridge</Label>
                  <p className="text-xs text-muted-foreground">
                    Post automated incident reports to configured Slack channels.
                  </p>
                </div>
                <Switch checked={slackSync} onCheckedChange={setSlackSync} />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label className="text-sm">Executive Weekly Digest</Label>
                  <p className="text-xs text-muted-foreground">
                    Delivered every Monday at 08:00 AM UTC.
                  </p>
                </div>
                <Switch checked={weeklyDigest} onCheckedChange={setWeeklyDigest} />
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
