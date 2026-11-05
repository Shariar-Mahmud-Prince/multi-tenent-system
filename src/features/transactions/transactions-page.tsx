import {
  Download,
  Percent,
  CheckCircle2,
  RotateCcw,
} from "lucide-react"
import { PageHeader } from "@/components/shared/page-header"
import { KPICard } from "@/components/shared/kpi-card"
import { RecentTransactionsTable } from "@/features/dashboard/components/recent-transactions-table"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"

export function TransactionsPage() {
  const handleExportCSV = () => {
    toast.success("Transaction ledger export (CSV) ready for download")
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Transactions"
        description="Comprehensive audit-grade financial ledger detailing all incoming charges, gateways, and settlements."
        badge="PCI-DSS Compliant"
      >
        <Button
          size="sm"
          onClick={handleExportCSV}
          className="h-9 gap-1.5"
        >
          <Download className="h-4 w-4" />
          <span>Export Ledger CSV</span>
        </Button>
      </PageHeader>

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KPICard
          title="Settled Gross Volume"
          value="$128,430.00"
          badgeText="+14.2%"
          isPositive={true}
          trendText="Processed past 30 days"
          description="Average settlement window: 24h"
        />
        <KPICard
          title="Gateway Processing Fees"
          value="$3,724.47"
          badgeText="2.9%"
          isPositive={true}
          trendText="Stripe & Wire network fees"
          description="Interchange cost optimized"
          icon={Percent}
        />
        <KPICard
          title="Settlement Success Rate"
          value="96.8%"
          badgeText="+0.4%"
          isPositive={true}
          trendText="First-attempt authorization"
          description="Card decline rate: 3.2%"
          icon={CheckCircle2}
        />
        <KPICard
          title="Refunded Volume"
          value="$299.00"
          badgeText="0.23%"
          isPositive={false}
          trendText="1 transaction refunded"
          description="Dispute loss: $0.00"
          icon={RotateCcw}
        />
      </div>

      {/* Full Transaction Table */}
      <div>
        <RecentTransactionsTable limit={10} />
      </div>
    </div>
  )
}
