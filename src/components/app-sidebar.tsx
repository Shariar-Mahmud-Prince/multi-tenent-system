import * as React from "react"
import {
  Compass,
  Repeat,
  BarChart3,
  Layers,
  Users,
  FileSpreadsheet,
  Building2,
  ArrowLeftRight,
  Settings,
  Sparkles,
  Plus,
  Mail,
  Circle,
} from "lucide-react"

import { NavMain, NavItem } from "@/components/nav-main"
import { NavUser } from "@/components/nav-user"
import { TeamSwitcher, Team } from "@/components/team-switcher"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"

const teams: Team[] = [
  {
    name: "Acme Inc.",
    logo: Circle,
    plan: "Enterprise",
  },
  {
    name: "Nexus Labs",
    logo: Layers,
    plan: "Growth",
  },
]

const mainNavItems: NavItem[] = [
  {
    title: "Dashboard",
    url: "/",
    icon: Compass,
  },
  {
    title: "Lifecycle",
    url: "/subscriptions",
    icon: Repeat,
  },
  {
    title: "Analytics",
    url: "/analytics",
    icon: BarChart3,
  },
  {
    title: "Projects",
    url: "/products",
    icon: Layers,
  },
  {
    title: "Team",
    url: "/users",
    icon: Users,
  },
]

const documentsNavItems: NavItem[] = [
  {
    title: "Customers",
    url: "/customers",
    icon: Building2,
  },
  {
    title: "Transactions",
    url: "/transactions",
    icon: ArrowLeftRight,
  },
  {
    title: "Reports",
    url: "/reports",
    icon: FileSpreadsheet,
  },
  {
    title: "Assistant",
    url: "/settings",
    icon: Sparkles,
  },
  {
    title: "Settings",
    url: "/settings",
    icon: Settings,
  },
]

const currentUser = {
  name: "Sarah Jenkins",
  email: "sarah.j@acme.inc",
  avatar: "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=100&h=100&fit=crop&crop=faces",
  role: "Owner",
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar variant="inset" collapsible="icon" {...props}>
      <SidebarHeader className="border-b border-border/40 pb-2">
        <TeamSwitcher teams={teams} />
        <div className="px-1 pt-1 flex items-center gap-1.5 group-data-[collapsible=icon]:hidden">
          <Button
            size="sm"
            onClick={() => toast.success("Quick create action opened")}
            className="flex-1 justify-start gap-1.5 h-8 bg-primary text-primary-foreground font-medium rounded-md px-2.5 text-xs"
          >
            <Plus className="h-3.5 w-3.5" />
            <span>Quick Create</span>
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={() => toast.info("No unread messages")}
            className="h-8 w-8 shrink-0 border-border"
          >
            <Mail className="h-3.5 w-3.5 text-muted-foreground" />
          </Button>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={mainNavItems} label="" />
        <NavMain items={documentsNavItems} label="Documents" />
      </SidebarContent>
      <SidebarFooter className="border-t border-border/40">
        <NavUser user={currentUser} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
