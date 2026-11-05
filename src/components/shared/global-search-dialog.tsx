import * as React from "react"
import { useNavigate } from "react-router-dom"
import {
  LayoutDashboard,
  BarChart3,
  Users,
  Building2,
  CreditCard,
  ArrowLeftRight,
  Boxes,
  FileSpreadsheet,
  Settings,
  Search,
} from "lucide-react"
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command"
import { Button } from "@/components/ui/button"

export function GlobalSearchDialog() {
  const [open, setOpen] = React.useState(false)
  const navigate = useNavigate()

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || e.key === "/") {
        if (
          (e.target instanceof HTMLElement && e.target.isContentEditable) ||
          e.target instanceof HTMLInputElement ||
          e.target instanceof HTMLTextAreaElement ||
          e.target instanceof HTMLSelectElement
        ) {
          return
        }
        e.preventDefault()
        setOpen((prev) => !prev)
      }
    }

    document.addEventListener("keydown", down)
    return () => document.removeEventListener("keydown", down)
  }, [])

  const handleSelect = (path: string) => {
    setOpen(false)
    navigate(path)
  }

  return (
    <>
      <Button
        variant="outline"
        onClick={() => setOpen(true)}
        className="relative h-8 w-8 p-0 md:h-8 md:w-56 md:justify-start md:px-3 text-muted-foreground font-normal"
      >
        <Search className="h-3.5 w-3.5 md:mr-2" />
        <span className="hidden md:inline-flex text-xs">Search dashboard...</span>
        <kbd className="pointer-events-none absolute right-1.5 top-1.5 hidden h-5 select-none items-center gap-0.5 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium opacity-100 md:flex">
          <span className="text-xs">⌘</span>K
        </kbd>
      </Button>

      <CommandDialog open={open} onOpenChange={setOpen}>
        <CommandInput placeholder="Type a command or search..." />
        <CommandList>
          <CommandEmpty>No results found.</CommandEmpty>
          <CommandGroup heading="Pages">
            <CommandItem onSelect={() => handleSelect("/")}>
              <LayoutDashboard className="mr-2 h-4 w-4" />
              <span>Overview</span>
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/analytics")}>
              <BarChart3 className="mr-2 h-4 w-4" />
              <span>Analytics</span>
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/users")}>
              <Users className="mr-2 h-4 w-4" />
              <span>Users</span>
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/customers")}>
              <Building2 className="mr-2 h-4 w-4" />
              <span>Customers</span>
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/subscriptions")}>
              <CreditCard className="mr-2 h-4 w-4" />
              <span>Subscriptions</span>
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/transactions")}>
              <ArrowLeftRight className="mr-2 h-4 w-4" />
              <span>Transactions</span>
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/products")}>
              <Boxes className="mr-2 h-4 w-4" />
              <span>Products</span>
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/reports")}>
              <FileSpreadsheet className="mr-2 h-4 w-4" />
              <span>Reports</span>
            </CommandItem>
          </CommandGroup>
          <CommandSeparator />
          <CommandGroup heading="Settings & Account">
            <CommandItem onSelect={() => handleSelect("/settings")}>
              <Settings className="mr-2 h-4 w-4" />
              <span>Workspace Settings</span>
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </CommandDialog>
    </>
  )
}
