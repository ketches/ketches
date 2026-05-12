import { useQuery } from "@tanstack/react-query"
import { Box, KeyboardIcon, Orbit } from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"

import { appsApi } from "@/api/apps"
import { envsApi } from "@/api/envs"
import { Badge } from "@/components/ui/badge"
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command"
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog"
import { useProjectStore } from "@/stores/project"

interface SearchItem {
  id: string
  name: string
  type: "environment" | "application"
  description?: string
  environmentName?: string
}

interface GlobalSearchDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function GlobalSearchDialog({ open, onOpenChange }: GlobalSearchDialogProps) {
  const navigate = useNavigate()
  const { activeProjectId, activeEnvId } = useProjectStore()

  const { data: envsResponse, isLoading: isLoadingEnvs } = useQuery({
    queryKey: ['envs', activeProjectId],
    queryFn: () => envsApi.list(activeProjectId!),
    enabled: !!activeProjectId && open,
  })

  const { data: appsResponse, isLoading: isLoadingApps } = useQuery({
    queryKey: ['apps', activeEnvId],
    queryFn: () => appsApi.list(activeEnvId!),
    enabled: !!activeEnvId && open,
  })

  const envs = envsResponse?.items ?? []
  const apps = appsResponse?.items ?? []
  const safeEnvs = Array.isArray(envs) ? envs : []
  const safeApps = Array.isArray(apps) ? apps : []
  const isLoading = isLoadingEnvs || isLoadingApps

  const envMap = React.useMemo(() => {
    const map = new Map()
    safeEnvs.forEach(env => map.set(env.id, env.name))
    return map
  }, [safeEnvs])

  const currentEnvName = activeEnvId ? envMap.get(activeEnvId) : undefined

  const environments: SearchItem[] = safeEnvs.map(env => ({
    id: env.id,
    name: env.name,
    type: "environment" as const,
    description: env.slug,
  }))

  const applications: SearchItem[] = safeApps.map(app => ({
    id: app.id,
    name: app.name,
    type: "application" as const,
    description: app.slug,
    environmentName: currentEnvName,
  }))

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        onOpenChange(!open)
      }
    }

    document.addEventListener("keydown", down)
    return () => document.removeEventListener("keydown", down)
  }, [open, onOpenChange])

  const handleSelect = (item: SearchItem) => {
    onOpenChange(false)
    if (item.type === "environment") {
      navigate(`/environments/${item.id}`)
    } else {
      navigate(`/applications/${item.id}`)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent showCloseButton={false} className="p-0 gap-0 sm:max-w-140 top-20! translate-y-0">
        <Command className="**:[[cmdk-group-heading]]:px-2 **:[[cmdk-group-heading]]:font-medium **:[[cmdk-group-heading]]:text-muted-foreground [&_[cmdk-group]:not([hidden])_~[cmdk-group]]:pt-0 **:[[cmdk-group]]:px-2 [&_[cmdk-input-wrapper]_svg]:h-5 [&_[cmdk-input-wrapper]_svg]:w-5 **:[[cmdk-input]]:h-12 **:[[cmdk-item]]:px-2 **:[[cmdk-item]]:py-3 [&_[cmdk-item]_svg]:h-5 [&_[cmdk-item]_svg]:w-5">
          <CommandInput placeholder="Input keywords..." />
          <CommandList>
            {isLoading ? (
              <div className="py-6 text-center text-sm text-muted-foreground">
                Loading...
              </div>
            ) : (
              <>
                <CommandEmpty>No results found</CommandEmpty>
                {applications.length > 0 && (
                  <>
                    <CommandGroup heading="Application">
                      {applications.map((item) => (
                        <CommandItem
                          key={item.id}
                          value={`${item.name} ${item.description || ""}`}
                          onSelect={() => handleSelect(item)}
                          className="gap-2"
                        >
                          <Box className="h-4 w-4" />
                          <span>{item.name}</span>
                          <span className="text-xs text-muted-foreground font-mono">{item.description}</span>
                          <div className="flex-1" />
                          {item.environmentName && (
                            <Badge variant="outline" className="text-[10px] px-1.5 py-0 font-normal">
                              <Orbit className="h-2.5 w-2.5 mr-1" />
                              {item.environmentName}
                            </Badge>
                          )}
                        </CommandItem>
                      ))}
                    </CommandGroup>
                    {environments.length > 0 && <CommandSeparator />}
                  </>
                )}
                {environments.length > 0 && (
                  <CommandGroup heading="Environment">
                    {environments.map((item) => (
                      <CommandItem
                        key={item.id}
                        value={`${item.name} ${item.description || ""}`}
                        onSelect={() => handleSelect(item)}
                        className="gap-2"
                      >
                        <Orbit className="h-4 w-4" />
                        <span>{item.name}</span>
                        <span className="text-xs text-muted-foreground font-mono">{item.description}</span>
                      </CommandItem>
                    ))}
                  </CommandGroup>
                )}
              </>
            )}
          </CommandList>
          <div className="border-t px-2 py-1.5 text-xs text-muted-foreground flex items-center justify-between">
            <i>Press esc to exit.</i>
            <KeyboardIcon className="h-3 w-3" />
          </div>
        </Command>
      </DialogContent>
    </Dialog>
  )
}
