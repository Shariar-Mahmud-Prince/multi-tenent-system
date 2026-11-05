import * as React from "react"
import {
  CreditCard,
  Search,
  MoreHorizontal,
  Calendar,
  Sparkles,
  ArrowUpRight,
  Plus,
  RefreshCw,
} from "lucide-react"
import { PageHeader } from "@/components/shared/page-header"
import { KPICard } from "@/components/shared/kpi-card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { mockSubscriptions } from "@/services/mockData"
import { Subscription } from "@/types"
import { formatCurrency, formatDate } from "@/lib/formatters"
import { toast } from "sonner"

export function SubscriptionsPage() {
  const [subscriptions, setSubscriptions] = React.useState<Subscription[]>(mockSubscriptions)
  const [search, setSearch] = React.useState("")
  const [statusFilter, setStatusFilter] = React.useState<string>("all")

  const filtered = React.useMemo(() => {
    return subscriptions.filter((s) => {
      const matchesSearch =
        s.customerName.toLowerCase().includes(search.toLowerCase()) ||
        s.customerEmail.toLowerCase().includes(search.toLowerCase()) ||
        s.plan.toLowerCase().includes(search.toLowerCase())
      const matchesStatus = statusFilter === "all" || s.status === statusFilter
      return matchesSearch && matchesStatus
    })
  }, [subscriptions, search, statusFilter])

  const handleCancelSub = (id: string) => {
    setSubscriptions((prev) =>
      prev.map((s) => (s.id === id ? { ...s, status: "canceled" as const } : s))
    )
    toast.success(`Subscription ${id} scheduled for cancellation at period end`)
  }

  const handleRenew = (id: string) => {
    toast.success(`Sent immediate renewal invoice for ${id}`)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Subscriptions"
        description="Monitor automated recurring renewals, active contracts, and expansion MRR across plan tiers."
        badge="Stripe Billing Live"
      >
        <Button
          size="sm"
          onClick={() => toast.info("Opening custom enterprise contract generator...")}
          className="h-9 gap-1.5"
        >
          <Plus className="h-4 w-4" />
          <span>New Subscription</span>
        </Button>
      </PageHeader>

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KPICard
          title="Annual Run Rate (ARR)"
          value="$1.54M"
          change="+22.4%"
          isPositive={true}
          description="Projected 12m recurring"
          icon={CreditCard}
        />
        <KPICard
          title="Monthly Run Rate (MRR)"
          value="$128,430"
          change="+14.2%"
          isPositive={true}
          description="Net new MRR: +$8,420"
          icon={ArrowUpRight}
        />
        <KPICard
          title="Avg. Revenue Per User"
          value="$37.55"
          change="+4.2%"
          isPositive={true}
          description="Expansion in seat add-ons"
          icon={Sparkles}
        />
        <KPICard
          title="Gross Monthly Churn"
          value="0.84%"
          change="-0.3%"
          isPositive={true}
          description="Industry benchmark: 2.1%"
          icon={RefreshCw}
        />
      </div>

      {/* Search & Filter */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative w-full max-w-sm">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search subscriber, email, or plan..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 h-9"
          />
        </div>

        <div className="flex items-center gap-2">
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="h-9 w-36">
              <SelectValue placeholder="All" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="past_due">Past Due</SelectItem>
              <SelectItem value="trialing">Trialing</SelectItem>
              <SelectItem value="canceled">Canceled</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Subscriptions Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Subscriber</TableHead>
              <TableHead>Plan Tier</TableHead>
              <TableHead>Billing Interval</TableHead>
              <TableHead>Amount</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="hidden sm:table-cell">Next Renewal</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                  No subscriptions found.
                </TableCell>
              </TableRow>
            ) : (
              filtered.map((sub) => (
                <TableRow key={sub.id}>
                  <TableCell>
                    <div className="flex flex-col">
                      <span className="font-medium text-sm">
                        {sub.customerName}
                      </span>
                      <span className="text-xs text-muted-foreground">
                        {sub.customerEmail}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1.5">
                      <span className="font-medium text-sm">{sub.plan}</span>
                      <Badge variant="outline" className="font-normal text-[10px]">
                        {sub.seats} seats
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell className="text-sm capitalize text-muted-foreground">
                    {sub.billingCycle}
                  </TableCell>
                  <TableCell className="font-mono text-sm font-semibold">
                    {formatCurrency(sub.amount)}
                  </TableCell>
                  <TableCell>
                    {sub.status === "active" && (
                      <Badge variant="secondary">Active</Badge>
                    )}
                    {sub.status === "past_due" && (
                      <Badge variant="outline" className="border-destructive/50 text-destructive">Past Due</Badge>
                    )}
                    {sub.status === "trialing" && (
                      <Badge variant="outline">Trialing</Badge>
                    )}
                    {sub.status === "canceled" && (
                      <Badge variant="outline">Canceled</Badge>
                    )}
                  </TableCell>
                  <TableCell className="hidden sm:table-cell text-sm text-muted-foreground">
                    <span className="flex items-center gap-1.5">
                      <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
                      {formatDate(sub.renewalDate)}
                    </span>
                  </TableCell>
                  <TableCell className="text-right">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-44">
                        <DropdownMenuLabel>Actions</DropdownMenuLabel>
                        <DropdownMenuItem
                          onClick={() => handleRenew(sub.id)}
                          className="gap-2"
                        >
                          <RefreshCw className="h-4 w-4" />
                          <span>Trigger Renewal</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => toast.info(`Opening tier modifier for ${sub.customerName}`)}
                          className="gap-2"
                        >
                          <Sparkles className="h-4 w-4" />
                          <span>Change Plan</span>
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => handleCancelSub(sub.id)}
                          disabled={sub.status === "canceled"}
                          className="gap-2 text-destructive focus:text-destructive"
                        >
                          <span>Cancel Subscription</span>
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
