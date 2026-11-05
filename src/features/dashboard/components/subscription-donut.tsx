import { Pie, PieChart } from "recharts"
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

const chartData = [
  { tier: "enterprise", subscribers: 1420, fill: "var(--color-enterprise)" },
  { tier: "pro", subscribers: 1150, fill: "var(--color-pro)" },
  { tier: "growth", subscribers: 580, fill: "var(--color-growth)" },
  { tier: "starter", subscribers: 270, fill: "var(--color-starter)" },
]

const chartConfig = {
  subscribers: {
    label: "Subscribers",
  },
  enterprise: {
    label: "Enterprise",
    color: "hsl(var(--foreground))",
  },
  pro: {
    label: "Pro",
    color: "hsl(var(--muted-foreground))",
  },
  growth: {
    label: "Growth",
    color: "hsl(var(--chart-3))",
  },
  starter: {
    label: "Starter",
    color: "hsl(var(--chart-4))",
  },
} satisfies ChartConfig

export function SubscriptionDonut() {
  return (
    <Card className="rounded-xl border border-border/80 bg-card p-6 shadow-sm flex flex-col justify-between space-y-4">
      <CardHeader className="p-0 space-y-1">
        <CardTitle className="text-base font-semibold text-foreground">
          Tier Breakdown
        </CardTitle>
        <CardDescription className="text-xs text-muted-foreground">
          Subscription distribution across tiers
        </CardDescription>
      </CardHeader>
      <CardContent className="p-0 flex-1 flex items-center justify-center">
        <ChartContainer
          config={chartConfig}
          className="mx-auto aspect-square max-h-[240px] w-full"
        >
          <PieChart>
            <ChartTooltip
              cursor={false}
              content={<ChartTooltipContent hideLabel />}
            />
            <Pie
              data={chartData}
              dataKey="subscribers"
              nameKey="tier"
              innerRadius={65}
              strokeWidth={4}
            />
            <ChartLegend
              content={<ChartLegendContent nameKey="tier" />}
              className="-translate-y-2 flex-wrap gap-2 [&>*]:basis-1/4 [&>*]:justify-center text-xs"
            />
          </PieChart>
        </ChartContainer>
      </CardContent>
    </Card>
  )
}
