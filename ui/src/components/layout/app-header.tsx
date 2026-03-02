import { ChevronRightIcon } from "lucide-react"
import { Fragment } from "react"
import { Link } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { useBreadcrumbs } from "@/contexts/breadcrumb-context"

export function AppHeader() {
  const { breadcrumbs } = useBreadcrumbs()

  return (
    <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
      <div className="flex items-center gap-2 px-4">
        <SidebarTrigger className="-ml-1 text-muted-foreground" />
        <Separator
          orientation="vertical"
          className="mr-2 data-[orientation=vertical]:h-4 !self-center"
        />
        <div className="flex items-center gap-2">
          {(breadcrumbs ?? []).map((item, index) => (
            <Fragment key={index}>
              <div className="flex items-center gap-1">
                {item.href ? (
                  <Button variant="ghost" render={<Link to={item.href} />} className="px-2">
                    {item.icon && <item.icon className="h-4 w-4 text-muted-foreground mr-1" />}
                    {item.label}
                  </Button>
                ) : (
                  <Button variant="ghost" className="px-2 cursor-default hover:bg-transparent">
                    {item.icon && <item.icon className="h-4 w-4 text-muted-foreground mr-1" />}
                    {item.label}
                  </Button>
                )}
                {item.dropdown && item.dropdown}
              </div>
              {index < (breadcrumbs ?? []).length - 1 && (
                <ChevronRightIcon className="h-4 w-4 text-muted-foreground shrink-0" />
              )}
            </Fragment>
          ))}
        </div>
      </div>
    </header>
  )
}

