import { Outlet } from "react-router-dom"
import { AppSidebar } from "@/components/app-sidebar"
import { AppHeader } from "@/components/layout/app-header"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { Toaster } from "sonner"

export function AppLayout() {
  return (
    <SidebarProvider className="h-svh w-full overflow-hidden">
      <AppSidebar />
      <SidebarInset className="flex h-svh md:h-[calc(100svh-1rem)] flex-col overflow-hidden border border-border/60 no-scrollbar">
        <AppHeader />
        <div className="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar p-4 md:p-6">
          <Outlet />
        </div>
      </SidebarInset>
      <Toaster position="bottom-right" richColors />
    </SidebarProvider>
  )
}
