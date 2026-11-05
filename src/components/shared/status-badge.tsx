import { Badge } from "@/components/ui/badge"
import { TransactionStatus, UserStatus, CustomerHealth } from "@/types"

export function TransactionStatusBadge({ status }: { status: TransactionStatus }) {
  switch (status) {
    case "completed":
      return <Badge variant="secondary">Completed</Badge>
    case "pending":
      return <Badge variant="outline">Pending</Badge>
    case "failed":
      return <Badge variant="destructive">Failed</Badge>
    case "refunded":
      return <Badge variant="outline">Refunded</Badge>
  }
}

export function UserStatusBadge({ status }: { status: UserStatus }) {
  switch (status) {
    case "active":
      return <Badge variant="secondary">Active</Badge>
    case "invited":
      return <Badge variant="outline">Invited</Badge>
    case "suspended":
      return <Badge variant="destructive">Suspended</Badge>
  }
}

export function CustomerHealthBadge({ health }: { health: CustomerHealth }) {
  switch (health) {
    case "Excellent":
      return <Badge variant="secondary">Excellent</Badge>
    case "Good":
      return <Badge variant="outline">Good</Badge>
    case "At Risk":
      return <Badge variant="outline" className="border-destructive/50 text-destructive">At Risk</Badge>
    case "Critical":
      return <Badge variant="destructive">Critical</Badge>
  }
}
