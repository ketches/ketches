import { RotateCcw, Trash2 } from "lucide-react"
import * as React from "react"

import { PageHeader } from "@/components/layout/page-header"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Tabs, TabsContent } from "@/components/ui/tabs"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useAuthStore } from "@/stores/auth"
import { RecycleBinAppsTab } from "./components/recycle-bin-apps-tab"
import { RecycleBinCodeRepositoriesTab } from "./components/recycle-bin-code-repositories-tab"
import { RecycleBinEnvsTab } from "./components/recycle-bin-envs-tab"
import { RecycleBinProjectsTab } from "./components/recycle-bin-projects-tab"
import { RecycleBinTabs } from "./components/recycle-bin-tabs"
import { RecycleBinUsersTab } from "./components/recycle-bin-users-tab"
import { type RecycleBinTabKey, useRecycleBinResources } from "./hooks/use-recycle-bin-resources"

interface BatchActionTable {
  getFilteredSelectedRowModel: () => {
    rows: unknown[]
  }
}

export function RecycleBinPage() {
  const systemRole = useAuthStore((state) => state.user?.role)
  const isAdmin = systemRole === "admin"
  const projectRole = useProjectRole()
  const isViewer = projectRole !== "owner" && projectRole !== "developer"

  const [activeTab, setActiveTab] = React.useState<RecycleBinTabKey>("projects")
  const [restoreDialogOpen, setRestoreDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)

  const resources = useRecycleBinResources({ activeTab, isAdmin })

  React.useEffect(() => {
    if (!isAdmin && activeTab === "users") {
      setActiveTab("projects")
    }
  }, [activeTab, isAdmin])

  const breadcrumbs = [{ label: "Recycle Bin", icon: Trash2 }]

  const leftToolbar = React.useCallback(() => (
    <Input
      className="flex max-w-sm min-w-75 flex-1"
      placeholder="Search deleted resources..."
      value={resources.searchQuery}
      onChange={(event) => resources.setSearchQuery(event.target.value)}
    />
  ), [resources])

  const batchActions = React.useCallback((table: BatchActionTable) => {
    const count = table.getFilteredSelectedRowModel().rows.length
    if (count === 0) {
      return null
    }

    return (
      <>
        <Button variant="outline" onClick={() => setRestoreDialogOpen(true)}>
          <RotateCcw />
          Restore ({count})
        </Button>
        <Button variant="destructive" onClick={() => setDeleteDialogOpen(true)}>
          <Trash2 />
          Delete ({count})
        </Button>
      </>
    )
  }, [])

  return (
    <div className="flex flex-1 flex-col gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Recycle Bin</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Restore or permanently delete soft-deleted resources
          </p>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as RecycleBinTabKey)}>
        <RecycleBinTabs
          activeTab={activeTab}
          isAdmin={isAdmin}
          totals={{
            projects: resources.projects.paginationInfo?.total ?? 0,
            apps: resources.apps.paginationInfo?.total ?? 0,
            envs: resources.envs.paginationInfo?.total ?? 0,
            "code-repos": resources.codeRepos.paginationInfo?.total ?? 0,
            users: resources.users.paginationInfo?.total ?? 0,
          }}
        />

        {isAdmin && (
          <TabsContent value="users" className="mt-2">
            <RecycleBinUsersTab
              data={resources.users.data}
              isLoading={resources.users.isLoading}
              isFetching={resources.users.isFetching}
              leftToolbar={leftToolbar}
              batchActions={batchActions}
              rowSelection={resources.users.rowSelection}
              onRowSelectionChange={resources.users.setRowSelection}
              onRefresh={resources.users.refetch}
              totalCount={resources.users.paginationInfo?.total ?? 0}
              pagination={resources.users.pagination}
              onPaginationChange={resources.users.setPagination}
              isAdmin={isAdmin}
              restoringItemId={resources.restoringItemId}
              deletingItemId={resources.deletingItemId}
              onRestoreSingle={(id) => resources.handleRestoreSingle(id, "user")}
              onDeleteSingle={(id) => resources.handleDeleteSingle(id, "user")}
            />
          </TabsContent>
        )}

        <TabsContent value="projects" className="mt-2">
          <RecycleBinProjectsTab
            data={resources.projects.data}
            isLoading={resources.projects.isLoading}
            isFetching={resources.projects.isFetching}
            leftToolbar={leftToolbar}
            batchActions={batchActions}
            rowSelection={resources.projects.rowSelection}
            onRowSelectionChange={resources.projects.setRowSelection}
            onRefresh={resources.projects.refetch}
            totalCount={resources.projects.paginationInfo?.total ?? 0}
            pagination={resources.projects.pagination}
            onPaginationChange={resources.projects.setPagination}
            isViewer={isViewer}
            restoringItemId={resources.restoringItemId}
            deletingItemId={resources.deletingItemId}
            onRestoreSingle={(id) => resources.handleRestoreSingle(id, "project")}
            onDeleteSingle={(id) => resources.handleDeleteSingle(id, "project")}
          />
        </TabsContent>

        <TabsContent value="apps" className="mt-2">
          <RecycleBinAppsTab
            data={resources.apps.data}
            isLoading={resources.apps.isLoading}
            isFetching={resources.apps.isFetching}
            leftToolbar={leftToolbar}
            batchActions={batchActions}
            rowSelection={resources.apps.rowSelection}
            onRowSelectionChange={resources.apps.setRowSelection}
            onRefresh={resources.apps.refetch}
            totalCount={resources.apps.paginationInfo?.total ?? 0}
            pagination={resources.apps.pagination}
            onPaginationChange={resources.apps.setPagination}
            isViewer={isViewer}
            restoringItemId={resources.restoringItemId}
            deletingItemId={resources.deletingItemId}
            onRestoreSingle={(id) => resources.handleRestoreSingle(id, "app")}
            onDeleteSingle={(id) => resources.handleDeleteSingle(id, "app")}
          />
        </TabsContent>

        <TabsContent value="envs" className="mt-2">
          <RecycleBinEnvsTab
            data={resources.envs.data}
            isLoading={resources.envs.isLoading}
            isFetching={resources.envs.isFetching}
            leftToolbar={leftToolbar}
            batchActions={batchActions}
            rowSelection={resources.envs.rowSelection}
            onRowSelectionChange={resources.envs.setRowSelection}
            onRefresh={resources.envs.refetch}
            totalCount={resources.envs.paginationInfo?.total ?? 0}
            pagination={resources.envs.pagination}
            onPaginationChange={resources.envs.setPagination}
            isViewer={isViewer}
            restoringItemId={resources.restoringItemId}
            deletingItemId={resources.deletingItemId}
            onRestoreSingle={(id) => resources.handleRestoreSingle(id, "env")}
            onDeleteSingle={(id) => resources.handleDeleteSingle(id, "env")}
          />
        </TabsContent>

        <TabsContent value="code-repos" className="mt-2">
          <RecycleBinCodeRepositoriesTab
            data={resources.codeRepos.data}
            isLoading={resources.codeRepos.isLoading}
            isFetching={resources.codeRepos.isFetching}
            leftToolbar={leftToolbar}
            batchActions={batchActions}
            rowSelection={resources.codeRepos.rowSelection}
            onRowSelectionChange={resources.codeRepos.setRowSelection}
            onRefresh={resources.codeRepos.refetch}
            totalCount={resources.codeRepos.paginationInfo?.total ?? 0}
            pagination={resources.codeRepos.pagination}
            onPaginationChange={resources.codeRepos.setPagination}
            isViewer={isViewer}
            restoringItemId={resources.restoringItemId}
            deletingItemId={resources.deletingItemId}
            onRestoreSingle={(id) => resources.handleRestoreSingle(id, "code-repo")}
            onDeleteSingle={(id) => resources.handleDeleteSingle(id, "code-repo")}
          />
        </TabsContent>
      </Tabs>

      <AlertDialog open={restoreDialogOpen} onOpenChange={setRestoreDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Restore Resources</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to restore {resources.selectedCount} {resources.selectedResourceLabel}?
              This will make them visible and usable again.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                resources.handleRestore()
                setRestoreDialogOpen(false)
              }}
            >
              Restore
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Permanently Delete Resources</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to permanently delete {resources.selectedCount} {resources.selectedResourceLabel}?
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                resources.handleDelete()
                setDeleteDialogOpen(false)
              }}
              variant="destructive"
            >
              Permanently Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={resources.conflictDialogOpen} onOpenChange={resources.setConflictDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cannot Delete Environment</AlertDialogTitle>
            <AlertDialogDescription>
              This environment contains {resources.conflictApps.length} deleted application(s). Please permanently delete or restore these applications first:
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="max-h-50 overflow-y-auto">
            <ul className="list-disc pl-6">
              {resources.conflictApps.map((app) => (
                <li key={app.id}>
                  {app.name} ({app.slug})
                </li>
              ))}
            </ul>
          </div>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => resources.setConflictDialogOpen(false)}>OK</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
