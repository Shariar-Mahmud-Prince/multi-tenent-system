import * as React from "react"
import {
  MoreHorizontal,
  ArrowUpDown,
  Search,
  Copy,
  Receipt,
  RotateCcw,
  ChevronLeft,
  ChevronRight,
} from "lucide-react"
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
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card"
import { TransactionStatusBadge } from "@/components/shared/status-badge"
import { mockTransactions } from "@/services/mockData"
import { Transaction, TransactionStatus } from "@/types"
import { formatCurrency, formatDate } from "@/lib/formatters"
import { toast } from "sonner"

export function RecentTransactionsTable({ limit }: { limit?: number }) {
  const [data, setData] = React.useState<Transaction[]>(mockTransactions)
  const [search, setSearch] = React.useState("")
  const [statusFilter, setStatusFilter] = React.useState<string>("all")
  const [sortField, setSortField] = React.useState<"date" | "amount">("date")
  const [sortDirection, setSortDirection] = React.useState<"asc" | "desc">("desc")
  const [currentPage, setCurrentPage] = React.useState(1)
  const [selectedTx, setSelectedTx] = React.useState<Transaction | null>(null)
  const pageSize = limit || 6

  // Filter
  const filteredData = React.useMemo(() => {
    return data.filter((item) => {
      const matchesSearch =
        item.customerName.toLowerCase().includes(search.toLowerCase()) ||
        item.customerEmail.toLowerCase().includes(search.toLowerCase()) ||
        item.id.toLowerCase().includes(search.toLowerCase())
      const matchesStatus =
        statusFilter === "all" || item.status === statusFilter
      return matchesSearch && matchesStatus
    })
  }, [data, search, statusFilter])

  // Sort
  const sortedData = React.useMemo(() => {
    return [...filteredData].sort((a, b) => {
      if (sortField === "amount") {
        return sortDirection === "asc"
          ? a.amount - b.amount
          : b.amount - a.amount
      } else {
        return sortDirection === "asc"
          ? new Date(a.date).getTime() - new Date(b.date).getTime()
          : new Date(b.date).getTime() - new Date(a.date).getTime()
      }
    })
  }, [filteredData, sortField, sortDirection])

  // Paginate
  const totalPages = Math.ceil(sortedData.length / pageSize) || 1
  const paginatedData = React.useMemo(() => {
    const start = (currentPage - 1) * pageSize
    return sortedData.slice(start, start + pageSize)
  }, [sortedData, currentPage, pageSize])

  const toggleSort = (field: "date" | "amount") => {
    if (sortField === field) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"))
    } else {
      setSortField(field)
      setSortDirection("desc")
    }
  }

  const handleCopyId = (id: string) => {
    navigator.clipboard.writeText(id)
    toast.success(`Copied ${id} to clipboard`)
  }

  const handleRefund = (id: string) => {
    setData((prev) =>
      prev.map((t) => (t.id === id ? { ...t, status: "refunded" as TransactionStatus } : t))
    )
    toast.success(`Refund initiated for transaction ${id}`)
  }

  return (
    <Card className="rounded-xl border border-border/80 bg-card p-6 shadow-sm space-y-4">
      <CardHeader className="p-0 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between space-y-0">
        <div>
          <CardTitle className="text-base font-semibold text-foreground">
            Recent Transactions
          </CardTitle>
          <CardDescription className="text-xs text-muted-foreground">
            Audit-grade payment logs across connected payment gateways
          </CardDescription>
        </div>

        <div className="flex flex-col sm:flex-row sm:items-center gap-2">
          <div className="relative w-full sm:w-56">
            <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
            <Input
              placeholder="Search customer, ID..."
              value={search}
              onChange={(e) => {
                setSearch(e.target.value)
                setCurrentPage(1)
              }}
              className="pl-8 h-8 text-xs"
            />
          </div>

          <Select
            value={statusFilter}
            onValueChange={(val) => {
              setStatusFilter(val)
              setCurrentPage(1)
            }}
          >
            <SelectTrigger className="h-8 w-32 text-xs">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="completed">Completed</SelectItem>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="failed">Failed</SelectItem>
              <SelectItem value="refunded">Refunded</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </CardHeader>

      <CardContent className="p-0 space-y-4">
        {/* Table */}
        <div className="rounded-lg border border-border/80 overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="text-xs">Customer</TableHead>
                <TableHead className="hidden md:table-cell text-xs">Plan</TableHead>
                <TableHead className="text-xs">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => toggleSort("amount")}
                    className="-ml-3 h-7 text-xs"
                  >
                    <span>Amount</span>
                    <ArrowUpDown className="ml-1 h-3 w-3" />
                  </Button>
                </TableHead>
                <TableHead className="text-xs">Status</TableHead>
                <TableHead className="hidden sm:table-cell text-xs">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => toggleSort("date")}
                    className="-ml-3 h-7 text-xs"
                  >
                    <span>Date</span>
                    <ArrowUpDown className="ml-1 h-3 w-3" />
                  </Button>
                </TableHead>
                <TableHead className="text-right text-xs">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedData.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="h-24 text-center text-muted-foreground text-xs">
                    No transactions found.
                  </TableCell>
                </TableRow>
              ) : (
                paginatedData.map((tx) => (
                  <TableRow
                    key={tx.id}
                    className="cursor-pointer hover:bg-muted/40 transition-colors"
                    onClick={() => setSelectedTx(tx)}
                  >
                    <TableCell className="py-2.5">
                      <div className="flex items-center gap-2.5">
                        <Avatar className="h-7 w-7 border border-border">
                          <AvatarImage src={tx.customerAvatar} alt={tx.customerName} />
                          <AvatarFallback className="text-[10px]">
                            {tx.customerName.slice(0, 2).toUpperCase()}
                          </AvatarFallback>
                        </Avatar>
                        <div className="flex flex-col">
                          <span className="font-medium text-xs text-foreground leading-tight">
                            {tx.customerName}
                          </span>
                          <span className="text-[11px] text-muted-foreground">
                            {tx.customerEmail}
                          </span>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell className="hidden md:table-cell text-xs text-muted-foreground">
                      <Badge variant="outline" className="font-normal text-[11px]">
                        {tx.plan}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs font-semibold">
                      {formatCurrency(tx.amount, tx.currency)}
                    </TableCell>
                    <TableCell>
                      <TransactionStatusBadge status={tx.status} />
                    </TableCell>
                    <TableCell className="hidden sm:table-cell text-xs text-muted-foreground">
                      {formatDate(tx.date, "MMM dd, yyyy")}
                    </TableCell>
                    <TableCell className="text-right" onClick={(e) => e.stopPropagation()}>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                          >
                            <MoreHorizontal className="h-3.5 w-3.5" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end" className="w-44">
                          <DropdownMenuLabel>Actions</DropdownMenuLabel>
                          <DropdownMenuItem
                            onClick={() => setSelectedTx(tx)}
                            className="gap-2 cursor-pointer text-xs"
                          >
                            <Receipt className="h-3.5 w-3.5" />
                            <span>View Receipt</span>
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => handleCopyId(tx.id)}
                            className="gap-2 cursor-pointer text-xs"
                          >
                            <Copy className="h-3.5 w-3.5" />
                            <span>Copy ID</span>
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            onClick={() => handleRefund(tx.id)}
                            disabled={tx.status === "refunded"}
                            className="gap-2 cursor-pointer text-xs text-destructive focus:text-destructive"
                          >
                            <RotateCcw className="h-3.5 w-3.5" />
                            <span>{tx.status === "refunded" ? "Refunded" : "Issue Refund"}</span>
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

        {/* Pagination Footer */}
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <div>
            Showing {paginatedData.length} of {sortedData.length} transactions
          </div>
          <div className="flex items-center space-x-1.5">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={currentPage <= 1}
              className="h-7 text-xs px-2"
            >
              <ChevronLeft className="h-3.5 w-3.5 mr-1" />
              Previous
            </Button>
            <span className="text-xs font-medium px-2">
              Page {currentPage} of {totalPages}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              disabled={currentPage >= totalPages}
              className="h-7 text-xs px-2"
            >
              Next
              <ChevronRight className="h-3.5 w-3.5 ml-1" />
            </Button>
          </div>
        </div>
      </CardContent>

      {/* Receipt Modal Dialog */}
      <Dialog open={!!selectedTx} onOpenChange={(open) => !open && setSelectedTx(null)}>
        {selectedTx && (
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <div className="flex items-center justify-between">
                <span className="text-xs font-mono text-muted-foreground">
                  {selectedTx.id}
                </span>
                <TransactionStatusBadge status={selectedTx.status} />
              </div>
              <DialogTitle className="text-lg">Transaction Receipt</DialogTitle>
              <DialogDescription>
                Processed via {selectedTx.paymentMethod}
              </DialogDescription>
            </DialogHeader>

            <div className="my-2 rounded-lg border p-4 space-y-3 text-sm">
              <div className="flex justify-between items-center">
                <span className="text-muted-foreground text-xs">Customer</span>
                <span className="font-medium">{selectedTx.customerName}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-muted-foreground text-xs">Billing Email</span>
                <span className="font-mono text-xs">{selectedTx.customerEmail}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-muted-foreground text-xs">Plan</span>
                <span className="font-medium">{selectedTx.plan}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-muted-foreground text-xs">Date</span>
                <span className="text-xs">{formatDate(selectedTx.date, "PPpp")}</span>
              </div>
              <div className="border-t pt-2 flex justify-between items-center">
                <span className="font-semibold text-sm">Total Paid</span>
                <span className="font-mono text-base font-bold text-primary">
                  {formatCurrency(selectedTx.amount, selectedTx.currency)}
                </span>
              </div>
            </div>

            <DialogFooter className="gap-2 sm:gap-0">
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleCopyId(selectedTx.id)}
              >
                <Copy className="h-4 w-4 mr-2" />
                Copy ID
              </Button>
              <Button
                size="sm"
                onClick={() => {
                  toast.success(`PDF receipt generated for ${selectedTx.id}`)
                  setSelectedTx(null)
                }}
              >
                Download PDF
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </Card>
  )
}
