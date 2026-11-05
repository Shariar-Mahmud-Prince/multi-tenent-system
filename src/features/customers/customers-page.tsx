import * as React from "react"
import {
  Building2,
  Search,
  MoreHorizontal,
  Mail,
  DollarSign,
  HeartPulse,
  TrendingUp,
  ExternalLink,
  Plus,
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Badge } from "@/components/ui/badge"
import { CustomerHealthBadge } from "@/components/shared/status-badge"
import { mockCustomers } from "@/services/mockData"
import { Customer } from "@/types"
import { formatCurrency } from "@/lib/formatters"
import { toast } from "sonner"

export function CustomersPage() {
  const [customers] = React.useState<Customer[]>(mockCustomers)
  const [search, setSearch] = React.useState("")
  const [tierFilter, setTierFilter] = React.useState<string>("all")
  const [selectedCustomer, setSelectedCustomer] = React.useState<Customer | null>(null)

  const filteredCustomers = React.useMemo(() => {
    return customers.filter((c) => {
      const matchesSearch =
        c.name.toLowerCase().includes(search.toLowerCase()) ||
        c.company.toLowerCase().includes(search.toLowerCase()) ||
        c.email.toLowerCase().includes(search.toLowerCase())
      const matchesTier = tierFilter === "all" || c.tier === tierFilter
      return matchesSearch && matchesTier
    })
  }, [customers, search, tierFilter])

  const totalMRR = customers.reduce((acc, c) => acc + c.mrr, 0)
  const totalLTV = customers.reduce((acc, c) => acc + c.totalSpent, 0)

  return (
    <div className="space-y-6">
      <PageHeader
        title="Customers"
        description="Enterprise client portfolios, MRR contributions, account health metrics, and contracts."
        badge={`${customers.length} Accounts`}
      >
        <Button
          size="sm"
          onClick={() => toast.info("Opening Add Organization...")}
          className="h-9 gap-1.5"
        >
          <Plus className="h-4 w-4" />
          <span>Add Account</span>
        </Button>
      </PageHeader>

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KPICard
          title="Portfolio MRR"
          value={formatCurrency(totalMRR)}
          change="+16.4%"
          isPositive={true}
          description="Recurring contract revenue"
          icon={DollarSign}
        />
        <KPICard
          title="Combined LTV"
          value={formatCurrency(totalLTV)}
          change="+28.2%"
          isPositive={true}
          description="Cumulative customer spend"
          icon={TrendingUp}
        />
        <KPICard
          title="Avg. Account Health"
          value="94.2%"
          change="+1.5%"
          isPositive={true}
          description="Composite engagement score"
          icon={HeartPulse}
        />
        <KPICard
          title="Active Accounts"
          value={customers.length.toString()}
          change="0 churned this Q"
          isPositive={true}
          description="100% retention rate"
          icon={Building2}
        />
      </div>

      {/* Filters & Search */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative w-full max-w-sm">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search accounts, companies, email..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 h-9"
          />
        </div>

        <div className="flex items-center gap-2">
          <Select value={tierFilter} onValueChange={setTierFilter}>
            <SelectTrigger className="h-9 w-36">
              <SelectValue placeholder="All Tiers" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Tiers</SelectItem>
              <SelectItem value="Enterprise">Enterprise</SelectItem>
              <SelectItem value="Pro">Pro</SelectItem>
              <SelectItem value="Growth">Growth</SelectItem>
              <SelectItem value="Starter">Starter</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Customers Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Organization</TableHead>
              <TableHead>Tier</TableHead>
              <TableHead>MRR</TableHead>
              <TableHead className="hidden sm:table-cell">Total LTV</TableHead>
              <TableHead>Health Score</TableHead>
              <TableHead className="hidden md:table-cell">Region</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filteredCustomers.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">
                  No customers found.
                </TableCell>
              </TableRow>
            ) : (
              filteredCustomers.map((c) => (
                <TableRow
                  key={c.id}
                  className="cursor-pointer"
                  onClick={() => setSelectedCustomer(c)}
                >
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <Avatar className="h-8 w-8">
                        <AvatarImage src={c.avatar} alt={c.name} />
                        <AvatarFallback>
                          {c.name.slice(0, 2).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex flex-col">
                        <span className="font-medium text-sm">
                          {c.name}
                        </span>
                        <span className="text-xs text-muted-foreground truncate">
                          {c.email}
                        </span>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="font-normal text-xs">
                      {c.tier}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-sm font-semibold">
                    {formatCurrency(c.mrr)}
                  </TableCell>
                  <TableCell className="hidden sm:table-cell font-mono text-sm text-muted-foreground">
                    {formatCurrency(c.totalSpent)}
                  </TableCell>
                  <TableCell>
                    <CustomerHealthBadge health={c.health} />
                  </TableCell>
                  <TableCell className="hidden md:table-cell text-sm text-muted-foreground">
                    {c.country}
                  </TableCell>
                  <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" className="w-44">
                        <DropdownMenuLabel>Actions</DropdownMenuLabel>
                        <DropdownMenuItem
                          onClick={() => setSelectedCustomer(c)}
                          className="gap-2"
                        >
                          <ExternalLink className="h-4 w-4" />
                          <span>View Profile</span>
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          onClick={() => toast.info(`Email composer opened for ${c.email}`)}
                          className="gap-2"
                        >
                          <Mail className="h-4 w-4" />
                          <span>Contact Primary</span>
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={() => toast.success(`Generated account audit package for ${c.name}`)}
                        >
                          <span>Export Dossier</span>
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

      {/* Customer Detail Dialog */}
      <Dialog open={!!selectedCustomer} onOpenChange={(open) => !open && setSelectedCustomer(null)}>
        {selectedCustomer && (
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <div className="flex items-center justify-between">
                <Badge variant="outline">{selectedCustomer.tier} Plan</Badge>
                <CustomerHealthBadge health={selectedCustomer.health} />
              </div>
              <DialogTitle className="text-xl mt-2">
                {selectedCustomer.name}
              </DialogTitle>
              <DialogDescription>
                {selectedCustomer.company} · Client since {selectedCustomer.joinedDate}
              </DialogDescription>
            </DialogHeader>

            <div className="my-2 grid grid-cols-2 gap-3">
              <div className="rounded-lg border p-3 space-y-1">
                <span className="text-xs text-muted-foreground">Monthly Recurring Revenue</span>
                <div className="text-lg font-bold font-mono text-primary">
                  {formatCurrency(selectedCustomer.mrr)}
                </div>
              </div>
              <div className="rounded-lg border p-3 space-y-1">
                <span className="text-xs text-muted-foreground">Total Lifetime Value</span>
                <div className="text-lg font-bold font-mono">
                  {formatCurrency(selectedCustomer.totalSpent)}
                </div>
              </div>
            </div>

            <div className="space-y-2 text-sm">
              <div className="flex justify-between py-1 border-b">
                <span className="text-muted-foreground">Primary Contact</span>
                <span className="font-mono text-xs">{selectedCustomer.email}</span>
              </div>
              <div className="flex justify-between py-1 border-b">
                <span className="text-muted-foreground">Region</span>
                <span>{selectedCustomer.country}</span>
              </div>
              <div className="flex justify-between py-1 border-b">
                <span className="text-muted-foreground">Status</span>
                <span className="capitalize font-medium">{selectedCustomer.status}</span>
              </div>
            </div>

            <DialogFooter className="gap-2 sm:gap-0">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSelectedCustomer(null)}
              >
                Close
              </Button>
              <Button
                size="sm"
                onClick={() => {
                  toast.success(`Exporting contract data for ${selectedCustomer.name}`)
                  setSelectedCustomer(null)
                }}
              >
                Download Statement
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  )
}
