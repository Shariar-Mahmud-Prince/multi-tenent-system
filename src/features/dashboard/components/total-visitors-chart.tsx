import * as React from "react"
import { Area, AreaChart, CartesianGrid, XAxis } from "recharts"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { Button } from "@/components/ui/button"

const chartData7Days = [
  { date: "Jun 24", visitors: 220, secondary: 110 },
  { date: "Jun 25", visitors: 90, secondary: 50 },
  { date: "Jun 26", visitors: 140, secondary: 75 },
  { date: "Jun 27", visitors: 520, secondary: 290 },
  { date: "Jun 28", visitors: 260, secondary: 140 },
  { date: "Jun 29", visitors: 95, secondary: 55 },
  { date: "Jun 30", visitors: 480, secondary: 260 },
]

const chartData30Days = [
  { date: "Jun 24", visitors: 210, secondary: 105 },
  { date: "Jun 25", visitors: 110, secondary: 60 },
  { date: "Jun 26", visitors: 160, secondary: 85 },
  { date: "Jun 27", visitors: 510, secondary: 280 },
  { date: "Jun 28", visitors: 240, secondary: 130 },
  { date: "Jun 29", visitors: 100, secondary: 60 },
  { date: "Jun 30", visitors: 490, secondary: 270 },
]

const chartData3Months = [
  { date: "Jun 24", visitors: 190, secondary: 95 },
  { date: "Jun 25", visitors: 100, secondary: 55 },
  { date: "Jun 26", visitors: 150, secondary: 80 },
  { date: "Jun 27", visitors: 490, secondary: 270 },
  { date: "Jun 28", visitors: 230, secondary: 120 },
  { date: "Jun 29", visitors: 90, secondary: 50 },
  { date: "Jun 30", visitors: 470, secondary: 250 },
]

const chartConfig = {
  visitors: {
    label: "Total Visitors",
    color: "hsl(var(--foreground))",
  },
  secondary: {
    label: "Direct Visitors",
    color: "hsl(var(--muted-foreground))",
  },
} satisfies ChartConfig

export function TotalVisitorsChart() {
  const [timeRange, setTimeRange] = React.useState<"3m" | "30d" | "7d">("30d")

  const data =
    timeRange === "7d"
      ? chartData7Days
      : timeRange === "30d"
      ? chartData30Days
      : chartData3Months

  return (
    <Card className="rounded-xl border border-border/80 bg-card p-6 shadow-sm space-y-4">
      <CardHeader className="p-0 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between space-y-0">
        <div className="space-y-1">
          <CardTitle className="text-base font-semibold text-foreground">
            Total Visitors
          </CardTitle>
          <CardDescription className="text-xs text-muted-foreground">
            Total for the last {timeRange === "3m" ? "3 months" : timeRange === "30d" ? "30 days" : "7 days"}
          </CardDescription>
        </div>

        {/* Segmented control matching screenshot */}
        <div className="inline-flex rounded-lg border border-border p-0.5 bg-muted/20">
          <Button
            variant={timeRange === "3m" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setTimeRange("3m")}
            className="h-7 px-3 text-xs rounded-md font-medium"
          >
            Last 3 months
          </Button>
          <Button
            variant={timeRange === "30d" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setTimeRange("30d")}
            className="h-7 px-3 text-xs rounded-md font-medium"
          >
            Last 30 days
          </Button>
          <Button
            variant={timeRange === "7d" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setTimeRange("7d")}
            className="h-7 px-3 text-xs rounded-md font-medium"
          >
            Last 7 days
          </Button>
        </div>
      </CardHeader>

      <CardContent className="p-0 pt-2">
        <ChartContainer
          config={chartConfig}
          className="aspect-auto h-[260px] w-full"
        >
          <AreaChart
            accessibilityLayer
            data={data}
            margin={{ top: 10, right: 10, left: 10, bottom: 0 }}
          >
            <defs>
              <linearGradient id="fillVisitors" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="hsl(var(--foreground))" stopOpacity={0.35} />
                <stop offset="95%" stopColor="hsl(var(--foreground))" stopOpacity={0.0} />
              </linearGradient>
              <linearGradient id="fillSecondary" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="hsl(var(--muted-foreground))" stopOpacity={0.25} />
                <stop offset="95%" stopColor="hsl(var(--muted-foreground))" stopOpacity={0.0} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} stroke="hsl(var(--border) / 0.4)" />
            <XAxis
              dataKey="date"
              tickLine={false}
              axisLine={false}
              tickMargin={12}
              className="text-xs text-muted-foreground"
            />
            <ChartTooltip
              cursor={{ stroke: "hsl(var(--border))", strokeWidth: 1 }}
              content={<ChartTooltipContent />}
            />
            <Area
              type="natural"
              dataKey="secondary"
              stroke="hsl(var(--muted-foreground))"
              strokeWidth={1.5}
              fill="url(#fillSecondary)"
              fillOpacity={0.4}
            />
            <Area
              type="natural"
              dataKey="visitors"
              stroke="hsl(var(--foreground))"
              strokeWidth={1.8}
              fill="url(#fillVisitors)"
              fillOpacity={0.6}
            />
          </AreaChart>
        </ChartContainer>
      </CardContent>
    </Card>
  )
}

