import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { TrendingUp, TrendingDown, type LucideIcon } from "lucide-react"

interface KPICardProps {
  title: string
  value: string
  badgeText?: string
  change?: string
  isPositive?: boolean
  trendText?: string
  description?: string
  icon?: LucideIcon
}

export function KPICard({
  title,
  value,
  badgeText,
  change,
  isPositive = true,
  trendText,
  description,
  icon: Icon,
}: KPICardProps) {
  const badgeDisplay = badgeText || change

  return (
    <Card className="rounded-xl border border-border/80 bg-card p-6 shadow-sm">
      <CardHeader className="p-0 flex flex-row items-center justify-between space-y-0 pb-3">
        <CardTitle className="text-xs font-normal text-muted-foreground">
          {title}
        </CardTitle>
        {badgeDisplay ? (
          <Badge
            variant="outline"
            className="text-[11px] font-medium px-2 py-0.5 rounded-full border-border/80 bg-muted/20 gap-1 text-foreground"
          >
            {isPositive ? (
              <TrendingUp className="h-3 w-3" />
            ) : (
              <TrendingDown className="h-3 w-3" />
            )}
            {badgeDisplay}
          </Badge>
        ) : (
          Icon && <Icon className="h-4 w-4 text-muted-foreground" />
        )}
      </CardHeader>
      <CardContent className="p-0 space-y-2">
        <div className="text-3xl font-bold tracking-tight text-foreground">
          {value}
        </div>
        <div className="space-y-0.5 text-xs">
          {trendText && (
            <div className="font-medium text-foreground flex items-center gap-1">
              <span>{trendText}</span>
              {isPositive ? (
                <TrendingUp className="h-3 w-3 inline text-muted-foreground" />
              ) : (
                <TrendingDown className="h-3 w-3 inline text-muted-foreground" />
              )}
            </div>
          )}
          {description && <p className="text-muted-foreground">{description}</p>}
        </div>
      </CardContent>
    </Card>
  )
}
