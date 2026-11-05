import * as React from "react"
import {
  Activity,
  Globe,
  Users,
  Clock,
  RefreshCw,
} from "lucide-react"
import { PageHeader } from "@/components/shared/page-header"
import { KPICard } from "@/components/shared/kpi-card"
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
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  PieChart,
  Pie,
} from "recharts"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
  type ChartConfig,
} from "@/components/ui/chart"
import {
  mockAnalyticsTraffic,
  mockFunnelData,
} from "@/services/mockData"
import { toast } from "sonner"

const deviceData = [
  { device: "macos", visitors: 58, fill: "var(--color-macos)" },
  { device: "windows", visitors: 24, fill: "var(--color-windows)" },
  { device: "ios", visitors: 12, fill: "var(--color-ios)" },
  { device: "android", visitors: 6, fill: "var(--color-android)" },
]

const deviceChartConfig = {
  visitors: {
    label: "Visitors",
  },
  macos: {
    label: "macOS",
    color: "hsl(var(--chart-1))",
  },
  windows: {
    label: "Windows",
    color: "hsl(var(--chart-2))",
  },
  ios: {
    label: "iOS",
    color: "hsl(var(--chart-3))",
  },
  android: {
    label: "Android",
    color: "hsl(var(--chart-4))",
  },
} satisfies ChartConfig

export function AnalyticsPage() {
  const [isRefreshing, setIsRefreshing] = React.useState(false)

  const handleRefresh = () => {
    setIsRefreshing(true)
    setTimeout(() => {
      setIsRefreshing(false)
      toast.success("Analytics data synced with edge cluster")
    }, 600)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Analytics"
        description="Deep funnel conversion analytics, traffic attribution, and real-time behavioral insights."
        badge="Live Telemetry"
      >
        <Button
          variant="outline"
          size="sm"
          onClick={handleRefresh}
          disabled={isRefreshing}
          className="h-9 gap-1.5"
        >
          <RefreshCw className={`h-4 w-4 ${isRefreshing ? "animate-spin" : ""}`} />
          <span>Sync Telemetry</span>
        </Button>
      </PageHeader>

      {/* Analytics KPI Row */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KPICard
          title="Live Concurrent Users"
          value="482"
          change="+18.4%"
          isPositive={true}
          description="Active on app right now"
          icon={Activity}
        />
        <KPICard
          title="Avg. Session Duration"
          value="8m 42s"
          change="+4.1%"
          isPositive={true}
          description="vs. 8m 21s last week"
          icon={Clock}
        />
        <KPICard
          title="Bounce Rate"
          value="24.8%"
          change="-3.2%"
          isPositive={true}
          description="Lower is better"
          icon={Globe}
        />
        <KPICard
          title="Trial Conversion"
          value="18.4%"
          change="+2.8%"
          isPositive={true}
          description="Signups to active paid"
          icon={Users}
        />
      </div>

      {/* Funnel & Device Breakdown */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        {/* Conversion Funnel */}
        <Card className="lg:col-span-4">
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle>Acquisition & Conversion Funnel</CardTitle>
              <CardDescription>
                Visitor progression from landing to paid customer
              </CardDescription>
            </div>
            <Badge variant="secondary" className="font-normal text-xs">
              2.7% Overall CR
            </Badge>
          </CardHeader>
          <CardContent className="space-y-4">
            {mockFunnelData.map((stage, idx) => (
              <div key={stage.stage} className="space-y-1.5">
                <div className="flex items-center justify-between text-xs font-medium">
                  <div className="flex items-center gap-2">
                    <span className="flex h-5 w-5 items-center justify-center rounded-full bg-muted text-[10px] font-bold">
                      {idx + 1}
                    </span>
                    <span>{stage.stage}</span>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="font-mono text-muted-foreground">{stage.count.toLocaleString()}</span>
                    <span className="font-mono font-semibold w-12 text-right">{stage.rate}</span>
                  </div>
                </div>
                <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                  <div
                    className="h-full bg-primary rounded-full transition-all duration-300"
                    style={{
                      width: `${(stage.count / mockFunnelData[0].count) * 100}%`,
                    }}
                  />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Device breakdown */}
        <Card className="lg:col-span-3 flex flex-col">
          <CardHeader>
            <CardTitle>Client Platforms</CardTitle>
            <CardDescription>
              Operating system and device telemetry
            </CardDescription>
          </CardHeader>
          <CardContent className="flex-1 flex flex-col justify-center">
            <ChartContainer
              config={deviceChartConfig}
              className="mx-auto aspect-square max-h-[220px]"
            >
              <PieChart>
                <ChartTooltip
                  cursor={false}
                  content={<ChartTooltipContent hideLabel />}
                />
                <Pie
                  data={deviceData}
                  dataKey="visitors"
                  nameKey="device"
                  innerRadius={50}
                  strokeWidth={5}
                />
                <ChartLegend
                  content={<ChartLegendContent nameKey="device" />}
                  className="-translate-y-2 flex-wrap gap-2 [&>*]:basis-1/4 [&>*]:justify-center"
                />
              </PieChart>
            </ChartContainer>
          </CardContent>
        </Card>
      </div>

      {/* Traffic Sources Table */}
      <Card>
        <CardHeader>
          <CardTitle>Traffic Attribution & Performance</CardTitle>
          <CardDescription>
            Top acquisition channels ranked by visitor volume and qualified conversions
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Source Channel</TableHead>
                <TableHead>Unique Visitors</TableHead>
                <TableHead>Bounce Rate</TableHead>
                <TableHead className="text-right">Conversion Rate</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {mockAnalyticsTraffic.map((source) => (
                <TableRow key={source.source}>
                  <TableCell className="font-medium text-sm">
                    {source.source}
                  </TableCell>
                  <TableCell className="font-mono text-sm text-muted-foreground">
                    {source.visitors.toLocaleString()}
                  </TableCell>
                  <TableCell className="font-mono text-sm text-muted-foreground">
                    {source.bounceRate}
                  </TableCell>
                  <TableCell className="font-mono text-sm font-semibold text-right">
                    {source.conversion}
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
