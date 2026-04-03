import { FolderGit2, GalleryVerticalEnd, Box, Orbit, User } from "lucide-react"

import { TabsList, TabsTrigger } from "@/components/ui/tabs"
import type { RecycleBinTabKey } from "@/pages/recycle-bin/hooks/use-recycle-bin-resources"

interface RecycleBinTabsProps {
  activeTab: RecycleBinTabKey
  isAdmin: boolean
  totals: Record<RecycleBinTabKey, number>
}

function getTabCountLabel(total: number): string {
  return total > 0 ? `(${total})` : ""
}

export function RecycleBinTabs({ activeTab: _activeTab, isAdmin, totals }: RecycleBinTabsProps) {
  return (
    <TabsList>
      <TabsTrigger value="projects">
        <GalleryVerticalEnd />
        Projects {getTabCountLabel(totals.projects)}
      </TabsTrigger>
      <TabsTrigger value="apps">
        <Box />
        Applications {getTabCountLabel(totals.apps)}
      </TabsTrigger>
      <TabsTrigger value="envs">
        <Orbit />
        Environments {getTabCountLabel(totals.envs)}
      </TabsTrigger>
      <TabsTrigger value="code-repos">
        <FolderGit2 />
        Code Repositories {getTabCountLabel(totals["code-repos"])}
      </TabsTrigger>
      {isAdmin && (
        <TabsTrigger value="users">
          <User />
          Users {getTabCountLabel(totals.users)}
        </TabsTrigger>
      )}
    </TabsList>
  )
}
