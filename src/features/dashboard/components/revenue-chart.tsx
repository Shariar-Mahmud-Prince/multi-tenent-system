import * as React from "react"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
  type ChartConfig,
} from "@/components/ui/chart"
import { mockRevenueMonthly } from "@/services/mockData"
import { formatCompactNumber } from "@/lib/formatters"
import { Button } from "@/components/ui/button"

const chartConfig = {
  revenue: {
    label: "Revenue",
    color: "hsl(var(--foreground))",
  },
  expenses: {
    label: "Expenses",
    color: "hsl(var(--muted-foreground))",
  },
} satisfies ChartConfig

export function RevenueChart() {
  const [timeframe, setTimeframe] = React.useState<"12m" | "6m">("12m")

  const data = timeframe === "6m" ? mockRevenueMonthly.slice(-6) : mockRevenueMonthly

  return (
    <Card className="rounded-xl border border-border/80 bg-card p-6 shadow-sm space-y-4">
      <CardHeader className="p-0 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between space-y-0">
        <div>
          <CardTitle className="text-base font-semibold text-foreground">
            Revenue Overview
          </CardTitle>
          <CardDescription className="text-xs text-muted-foreground">
            Monthly gross revenue vs operating costs
          </CardDescription>
        </div>
        <div className="inline-flex rounded-lg border border-border p-0.5 bg-muted/20">
          <Button
            variant={timeframe === "6m" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setTimeframe("6m")}
            className="h-7 text-xs px-2.5 rounded-md font-medium"
          >
            Last 6m
          </Button>
          <Button
            variant={timeframe === "12m" ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setTimeframe("12m")}
            className="h-7 text-xs px-2.5 rounded-md font-medium"
          >
            All Year
          </Button>
        </div>
      </CardHeader>
      <CardContent className="p-0 pt-2">
        <ChartContainer config={chartConfig} className="aspect-auto h-[260px] w-full">
          <BarChart accessibilityLayer data={data} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
            <CartesianGrid vertical={false} stroke="hsl(var(--border) / 0.4)" />
            <XAxis
              dataKey="month"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
            />
            <YAxis
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              tickFormatter={(val) => `$${formatCompactNumber(val)}`}
            />
            <ChartTooltip
              cursor={{ fill: "hsl(var(--muted) / 0.3)" }}
              content={<ChartTooltipContent indicator="dashed" />}
            />
            <ChartLegend content={<ChartLegendContent />} />
            <Bar
              dataKey="revenue"
              fill="var(--color-revenue)"
              radius={[4, 4, 0, 0]}
              maxBarSize={30}
            />
            <Bar
              dataKey="expenses"
              fill="var(--color-expenses)"
              radius={[4, 4, 0, 0]}
              maxBarSize={30}
            />
          </BarChart>
        </ChartContainer>
      </CardContent>
    </Card>
  )
}
