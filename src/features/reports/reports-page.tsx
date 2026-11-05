import * as React from "react"
import {
  FileSpreadsheet,
  Download,
  ShieldCheck,
  CheckCircle2,
  AlertTriangle,
} from "lucide-react"
import { PageHeader } from "@/components/shared/page-header"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { mockAuditLogs } from "@/services/mockData"
import { formatDate } from "@/lib/formatters"
import { toast } from "sonner"

export function ReportsPage() {
  const [reportType, setReportType] = React.useState("pnl")
  const [timeframe, setTimeframe] = React.useState("ytd")
  const [format, setFormat] = React.useState("pdf")
  const [isGenerating, setIsGenerating] = React.useState(false)

  const handleGenerate = () => {
    setIsGenerating(true)
    setTimeout(() => {
      setIsGenerating(false)
      toast.success(`Generated ${reportType.toUpperCase()} statement (${format.toUpperCase()})`)
    }, 800)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Reports"
        description="Official income accounting, sales tax filings, scheduled reconciliations, and compliance audit logs."
        badge="SOC2 Type II Certified"
      />

      {/* Generator Card */}
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <FileSpreadsheet className="h-5 w-5 text-muted-foreground" />
            <CardTitle>Statement & Data Export Generator</CardTitle>
          </div>
          <CardDescription>
            Configure parameters to compile certified balance records for accounting or tax authorities.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-4 items-end">
            <div className="space-y-2">
              <Label>Report Type</Label>
              <Select value={reportType} onValueChange={setReportType}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="pnl">Profit & Loss (P&L)</SelectItem>
                  <SelectItem value="mrr">MRR Expansion & Churn</SelectItem>
                  <SelectItem value="tax">Sales Tax & VAT</SelectItem>
                  <SelectItem value="audit">Security Audit Trail</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Period</Label>
              <Select value={timeframe} onValueChange={setTimeframe}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ytd">Year-to-Date (2026)</SelectItem>
                  <SelectItem value="q3">Q3 2026</SelectItem>
                  <SelectItem value="oct">October 2026</SelectItem>
                  <SelectItem value="annual">FY 2025 Annual</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>File Format</Label>
              <Select value={format} onValueChange={setFormat}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="pdf">PDF Document</SelectItem>
                  <SelectItem value="csv">CSV Spreadsheet</SelectItem>
                  <SelectItem value="json">Raw JSON</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <Button
              onClick={handleGenerate}
              disabled={isGenerating}
              className="gap-2"
            >
              <Download className={`h-4 w-4 ${isGenerating ? "animate-bounce" : ""}`} />
              <span>{isGenerating ? "Compiling..." : "Generate Statement"}</span>
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Financial Statement Highlights */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="uppercase font-medium text-xs">Net Revenue (YTD)</CardDescription>
            <CardTitle className="text-2xl font-bold font-mono">$1,180,430.00</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">+24.8% vs FY 2025</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="uppercase font-medium text-xs">Gateway & Interchange</CardDescription>
            <CardTitle className="text-2xl font-bold font-mono">$34,232.47</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">2.9% average processing</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="uppercase font-medium text-xs">Operating Margin</CardDescription>
            <CardTitle className="text-2xl font-bold font-mono">67.4%</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">Top quartile SaaS</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="uppercase font-medium text-xs">Estimated Tax Reserve</CardDescription>
            <CardTitle className="text-2xl font-bold font-mono">$118,043.00</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">Set aside in treasury</p>
          </CardContent>
        </Card>
      </div>

      {/* Audit Log Table */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>Security & Configuration Audit Log</CardTitle>
            <CardDescription>
              Immutable record of team actions, role elevations, and payment events
            </CardDescription>
          </div>
          <Badge variant="outline" className="gap-1 font-normal text-xs">
            <ShieldCheck className="h-3.5 w-3.5" />
            Verified
          </Badge>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Timestamp</TableHead>
                <TableHead>Actor</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Scope</TableHead>
                <TableHead className="hidden sm:table-cell">IP Address</TableHead>
                <TableHead className="text-right">Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {mockAuditLogs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {formatDate(log.timestamp, "MMM dd · HH:mm:ss")}
                  </TableCell>
                  <TableCell className="text-sm font-medium">
                    {log.user}
                  </TableCell>
                  <TableCell className="text-sm">
                    {log.action}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    <Badge variant="outline" className="font-mono text-[11px]">
                      {log.entity}
                    </Badge>
                  </TableCell>
                  <TableCell className="hidden sm:table-cell font-mono text-xs text-muted-foreground">
                    {log.ipAddress}
                  </TableCell>
                  <TableCell className="text-right">
                    {log.status === "success" ? (
                      <Badge variant="secondary" className="gap-1 font-normal">
                        <CheckCircle2 className="h-3 w-3" /> Success
                      </Badge>
                    ) : (
                      <Badge variant="outline" className="gap-1 font-normal border-destructive/50 text-destructive">
                        <AlertTriangle className="h-3 w-3" /> Warning
                      </Badge>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
