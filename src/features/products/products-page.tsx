import * as React from "react"
import {
  Check,
  Plus,
  Sparkles,
  Users,
  DollarSign,
  TrendingUp,
  Settings2,
} from "lucide-react"
import { PageHeader } from "@/components/shared/page-header"
import { KPICard } from "@/components/shared/kpi-card"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog"
import { mockProducts } from "@/services/mockData"
import { Product } from "@/types"
import { formatCurrency, formatNumber } from "@/lib/formatters"
import { toast } from "sonner"

export function ProductsPage() {
  const [products, setProducts] = React.useState<Product[]>(mockProducts)
  const [isAddOpen, setIsAddOpen] = React.useState(false)
  const [newTitle, setNewTitle] = React.useState("")
  const [newDesc, setNewDesc] = React.useState("")
  const [newPrice, setNewPrice] = React.useState("99")

  const totalProductMRR = products.reduce((acc, p) => acc + p.mrrContribution, 0)
  const totalSubscribers = products.reduce((acc, p) => acc + p.activeSubscribers, 0)

  const handleCreateProduct = (e: React.FormEvent) => {
    e.preventDefault()
    if (!newTitle) return

    const newProd: Product = {
      id: `prod_${Date.now()}`,
      name: newTitle,
      description: newDesc || "High-performance SaaS product module.",
      priceMonthly: Number(newPrice) || 99,
      priceAnnually: (Number(newPrice) || 99) * 10,
      activeSubscribers: 1,
      mrrContribution: Number(newPrice) || 99,
      features: [
        "Automated provisioning",
        "API rate limit: 10,000 req/min",
        "Standard SLA support",
      ],
      status: "active",
      category: "addon",
    }

    setProducts([...products, newProd])
    setIsAddOpen(false)
    setNewTitle("")
    setNewDesc("")
    toast.success(`Published new product catalog entry: ${newTitle}`)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Products"
        description="Configure subscription tiers, API quotas, add-on modules, and pricing definitions."
        badge={`${products.length} Products`}
      >
        <Button
          size="sm"
          onClick={() => setIsAddOpen(true)}
          className="h-9 gap-1.5"
        >
          <Plus className="h-4 w-4" />
          <span>New Product</span>
        </Button>
      </PageHeader>

      {/* KPI Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KPICard
          title="Catalog MRR"
          value={formatCurrency(totalProductMRR)}
          change="+18.5%"
          isPositive={true}
          description="Across all listed plans"
          icon={DollarSign}
        />
        <KPICard
          title="Total Subscribers"
          value={formatNumber(totalSubscribers)}
          change="+340 this month"
          isPositive={true}
          description="Active license assignments"
          icon={Users}
        />
        <KPICard
          title="Top Product"
          value="Cloud Core"
          change="$42,100 MRR"
          isPositive={true}
          description="32.8% of platform revenue"
          icon={Sparkles}
        />
        <KPICard
          title="Avg. Commitment"
          value="14.2 mos"
          change="+2.1 mos"
          isPositive={true}
          description="Annual commitments"
          icon={TrendingUp}
        />
      </div>

      {/* Products Grid */}
      <div className="grid gap-6 md:grid-cols-2">
        {products.map((product) => (
          <Card key={product.id} className="flex flex-col justify-between">
            <CardHeader className="space-y-3">
              <div className="flex items-start justify-between gap-2">
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <CardTitle className="text-lg">
                      {product.name}
                    </CardTitle>
                    <Badge variant="outline" className="text-xs capitalize">
                      {product.category}
                    </Badge>
                  </div>
                  <CardDescription className="line-clamp-2">
                    {product.description}
                  </CardDescription>
                </div>
                <div className="text-right shrink-0">
                  <div className="text-xl font-bold font-mono">
                    ${product.priceMonthly}
                    <span className="text-xs text-muted-foreground font-normal">/mo</span>
                  </div>
                  <div className="text-xs text-muted-foreground">
                    ${product.priceAnnually}/yr
                  </div>
                </div>
              </div>

              {/* Metrics */}
              <div className="flex items-center gap-4 rounded-md border p-2 text-xs">
                <div>
                  <span className="text-muted-foreground">Subscribers: </span>
                  <span className="font-semibold">{formatNumber(product.activeSubscribers)}</span>
                </div>
                <div className="h-3 w-px bg-border" />
                <div>
                  <span className="text-muted-foreground">MRR Share: </span>
                  <span className="font-semibold">{formatCurrency(product.mrrContribution)}</span>
                </div>
              </div>
            </CardHeader>

            <CardContent className="space-y-2">
              <span className="text-xs font-semibold">Included Features:</span>
              <ul className="space-y-1 text-xs text-muted-foreground">
                {product.features.map((feature, idx) => (
                  <li key={idx} className="flex items-center gap-2">
                    <Check className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                    <span>{feature}</span>
                  </li>
                ))}
              </ul>
            </CardContent>

            <CardFooter className="pt-4 border-t flex items-center justify-between">
              <Button
                variant="outline"
                size="sm"
                onClick={() => toast.info(`Editing settings for ${product.name}`)}
                className="gap-1.5"
              >
                <Settings2 className="h-4 w-4" />
                <span>Configure</span>
              </Button>
              <Button
                size="sm"
                onClick={() => toast.success(`Simulating checkout link for ${product.name}`)}
              >
                Checkout Link
              </Button>
            </CardFooter>
          </Card>
        ))}
      </div>

      {/* Add Product Dialog */}
      <Dialog open={isAddOpen} onOpenChange={setIsAddOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Add Product / Plan</DialogTitle>
            <DialogDescription>
              Define a new product tier or expansion add-on for the public checkout catalog.
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleCreateProduct} className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="product-title">Product Title</Label>
              <Input
                id="product-title"
                required
                placeholder="Nexus Vector DB Cluster"
                value={newTitle}
                onChange={(e) => setNewTitle(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="product-desc">Short Description</Label>
              <Input
                id="product-desc"
                placeholder="Real-time indexing engine for LLM search"
                value={newDesc}
                onChange={(e) => setNewDesc(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="product-price">Monthly Price ($ USD)</Label>
              <Input
                id="product-price"
                type="number"
                required
                min="0"
                value={newPrice}
                onChange={(e) => setNewPrice(e.target.value)}
              />
            </div>

            <DialogFooter className="pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsAddOpen(false)}
              >
                Cancel
              </Button>
              <Button type="submit">
                Create Product
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
