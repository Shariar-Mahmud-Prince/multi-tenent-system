export type TransactionStatus = "completed" | "pending" | "failed" | "refunded";

export interface Transaction {
  id: string;
  customerName: string;
  customerEmail: string;
  customerAvatar?: string;
  amount: number;
  currency: string;
  status: TransactionStatus;
  date: string;
  paymentMethod: string;
  plan: string;
}

export type UserRole = "Owner" | "Admin" | "Member" | "Billing" | "Viewer";
export type UserStatus = "active" | "invited" | "suspended";

export interface User {
  id: string;
  name: string;
  email: string;
  avatar: string;
  role: UserRole;
  status: UserStatus;
  twoFactorEnabled: boolean;
  lastActive: string;
  createdAt: string;
}

export type CustomerTier = "Enterprise" | "Pro" | "Growth" | "Starter";
export type CustomerHealth = "Excellent" | "Good" | "At Risk" | "Critical";

export interface Customer {
  id: string;
  name: string;
  company: string;
  email: string;
  avatar: string;
  tier: CustomerTier;
  mrr: number;
  totalSpent: number;
  health: CustomerHealth;
  status: "active" | "churned" | "trialing";
  joinedDate: string;
  country: string;
}

export interface Subscription {
  id: string;
  customerName: string;
  customerEmail: string;
  plan: string;
  billingCycle: "monthly" | "annually";
  amount: number;
  status: "active" | "past_due" | "canceled" | "trialing";
  startDate: string;
  renewalDate: string;
  seats: number;
}

export interface Product {
  id: string;
  name: string;
  description: string;
  priceMonthly: number;
  priceAnnually: number;
  activeSubscribers: number;
  mrrContribution: number;
  features: string[];
  status: "active" | "draft" | "archived";
  category: "core" | "addon" | "enterprise";
}

export interface AuditLog {
  id: string;
  user: string;
  action: string;
  entity: string;
  ipAddress: string;
  timestamp: string;
  status: "success" | "warning" | "failure";
}

export interface NotificationItem {
  id: string;
  title: string;
  message: string;
  time: string;
  read: boolean;
  type: "info" | "success" | "warning" | "error";
}

export interface KPIData {
  title: string;
  value: string;
  change: string;
  isPositive: boolean;
  description: string;
  chartData: { value: number }[];
}

