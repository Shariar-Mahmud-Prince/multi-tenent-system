import { KPICard } from "@/components/shared/kpi-card"
import { TotalVisitorsChart } from "@/features/dashboard/components/total-visitors-chart"
import { RevenueChart } from "@/features/dashboard/components/revenue-chart"
import { SubscriptionDonut } from "@/features/dashboard/components/subscription-donut"
import { RecentTransactionsTable } from "@/features/dashboard/components/recent-transactions-table"

export function DashboardPage() {
  return (
    <div className="space-y-6">
      {/* 4 Metric Cards Matching Screenshot */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KPICard
          title="Total Revenue"
          value="$1,250.00"
          badgeText="+12.5%"
          isPositive={true}
          trendText="Trending up this month"
          description="Visitors for the last 6 months"
        />
        <KPICard
          title="New Customers"
          value="1,234"
          badgeText="-20%"
          isPositive={false}
          trendText="Down 20% this period"
          description="Acquisition needs attention"
        />
        <KPICard
          title="Active Accounts"
          value="45,678"
          badgeText="+12.5%"
          isPositive={true}
          trendText="Strong user retention"
          description="Engagement exceed targets"
        />
        <KPICard
          title="Growth Rate"
          value="4.5%"
          badgeText="+4.5%"
          isPositive={true}
          trendText="Steady performance increase"
          description="Meets growth projections"
        />
      </div>

      {/* Main Chart: Total Visitors matching Screenshot */}
      <div>
        <TotalVisitorsChart />
      </div>

      {/* Secondary Charts: Revenue & Tier Breakdown */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        <div className="lg:col-span-4">
          <RevenueChart />
        </div>
        <div className="lg:col-span-3">
          <SubscriptionDonut />
        </div>
      </div>

      {/* Recent Transactions Table */}
      <div>
        <RecentTransactionsTable limit={6} />
      </div>
    </div>
  )
}
