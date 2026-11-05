import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts"
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
  ChartLegend,
  ChartLegendContent,
} from "@/components/ui/chart"
import { mockUserGrowth } from "@/services/mockData"
import { formatCompactNumber } from "@/lib/formatters"
import { Badge } from "@/components/ui/badge"

const chartConfig = {
  returningUsers: {
    label: "Returning Users",
    color: "hsl(var(--chart-1))",
  },
  newUsers: {
    label: "New Signups",
    color: "hsl(var(--chart-2))",
  },
} satisfies ChartConfig

export function UserGrowthChart() {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle>User Growth</CardTitle>
          <CardDescription>
            New signups compared with monthly returning active users
          </CardDescription>
        </div>
        <Badge variant="secondary" className="font-normal text-xs">
          +32.4% QoQ
        </Badge>
      </CardHeader>
      <CardContent>
        <ChartContainer config={chartConfig} className="aspect-auto h-[300px] w-full">
          <AreaChart accessibilityLayer data={mockUserGrowth} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
            <defs>
              <linearGradient id="fillReturningUsers" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--color-returningUsers)" stopOpacity={0.4} />
                <stop offset="95%" stopColor="var(--color-returningUsers)" stopOpacity={0.0} />
              </linearGradient>
              <linearGradient id="fillNewUsers" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--color-newUsers)" stopOpacity={0.4} />
                <stop offset="95%" stopColor="var(--color-newUsers)" stopOpacity={0.0} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} />
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
              tickFormatter={(val) => formatCompactNumber(val)}
            />
            <ChartTooltip cursor={false} content={<ChartTooltipContent />} />
            <ChartLegend content={<ChartLegendContent />} />
            <Area
              type="natural"
              dataKey="returningUsers"
              fill="url(#fillReturningUsers)"
              fillOpacity={0.4}
              stroke="var(--color-returningUsers)"
            />
            <Area
              type="natural"
              dataKey="newUsers"
              fill="url(#fillNewUsers)"
              fillOpacity={0.4}
              stroke="var(--color-newUsers)"
            />
          </AreaChart>
        </ChartContainer>
      </CardContent>
    </Card>
  )
}
