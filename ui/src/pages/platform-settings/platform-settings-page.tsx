import { ArrowUpCircle, History, Settings2, Type } from "lucide-react"
import * as React from "react"

import { PageHeader } from "@/components/layout/page-header"
import { PlatformAuditLogTab } from "@/components/platform-settings/platform-audit-log-tab"
import { PlatformGeneralTab } from "@/components/platform-settings/platform-general-tab"
import { PlatformUpgradeManagementTab } from "@/components/platform-settings/platform-upgrade-management-tab"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { usePlatformBranding } from "@/hooks/use-platform-settings"
import { useCheckPlatformUpdateMutation, usePlatformUpdateStatusQuery } from "@/hooks/use-platform-update"
import { useVersion } from "@/hooks/useVersion"

export function PlatformSettingsPage() {
  const version = useVersion()
  const { data: branding } = usePlatformBranding()
  const statusQuery = usePlatformUpdateStatusQuery()
  const checkMutation = useCheckPlatformUpdateMutation()
  const [activeTab, setActiveTab] = React.useState("general")

  const status = checkMutation.data ?? statusQuery.data
  const brandName = branding?.name || "Ketches Admin"

  const handleCheckForUpdates = () => {
    checkMutation.mutate({ mode: "manual" })
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={[{ label: "Platform Settings", icon: Settings2 }]} />

      <div className="flex flex-col gap-4">
        <div className="flex items-start gap-4">
          <div className="flex aspect-square size-14 items-center justify-center overflow-hidden rounded-lg">
            <img src="/ketches.svg" alt={brandName} className="size-full object-cover rounded-lg" />
          </div>

          <div>
            <h1 className="text-2xl font-bold tracking-tight">Platform Settings</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {brandName} {version ? `· ${version}` : ""}
            </p>
            {status?.recommended_shared_version ? (
              <p className="mt-1 text-sm font-medium">
                Recommended update {status.recommended_shared_version}
              </p>
            ) : null}
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="general">
            <Type />
            General
          </TabsTrigger>
          <TabsTrigger value="upgrade-management">
            <ArrowUpCircle />
            Upgrade Management
          </TabsTrigger>
          <TabsTrigger value="audit-log">
            <History />
            Audit Log
          </TabsTrigger>
        </TabsList>
        <TabsContent value="general" className="mt-2 space-y-4">
          <PlatformGeneralTab />
        </TabsContent>
        <TabsContent value="upgrade-management" className="mt-2 space-y-4">
          <PlatformUpgradeManagementTab
            status={status}
            isStatusLoading={statusQuery.isLoading}
            isChecking={checkMutation.isPending}
            onCheckForUpdates={handleCheckForUpdates}
          />
        </TabsContent>
        <TabsContent value="audit-log" className="mt-2 space-y-4">
          <PlatformAuditLogTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}
