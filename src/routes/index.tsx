import { createBrowserRouter, Navigate } from "react-router-dom"
import { AppLayout } from "@/components/layout/app-layout"
import { DashboardPage } from "@/features/dashboard/dashboard-page"
import { AnalyticsPage } from "@/features/analytics/analytics-page"
import { UsersPage } from "@/features/users/users-page"
import { CustomersPage } from "@/features/customers/customers-page"
import { SubscriptionsPage } from "@/features/subscriptions/subscriptions-page"
import { TransactionsPage } from "@/features/transactions/transactions-page"
import { ProductsPage } from "@/features/products/products-page"
import { ReportsPage } from "@/features/reports/reports-page"
import { SettingsPage } from "@/features/settings/settings-page"

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppLayout />,
    children: [
      {
        index: true,
        element: <DashboardPage />,
      },
      {
        path: "analytics",
        element: <AnalyticsPage />,
      },
      {
        path: "users",
        element: <UsersPage />,
      },
      {
        path: "customers",
        element: <CustomersPage />,
      },
      {
        path: "subscriptions",
        element: <SubscriptionsPage />,
      },
      {
        path: "transactions",
        element: <TransactionsPage />,
      },
      {
        path: "products",
        element: <ProductsPage />,
      },
      {
        path: "reports",
        element: <ReportsPage />,
      },
      {
        path: "settings",
        element: <SettingsPage />,
      },
      {
        path: "*",
        element: <Navigate to="/" replace />,
      },
    ],
  },
])

