import type {
  BuilderExport,
  BuilderExportDeployBuildRequest,
  BuilderExportInitialBuildPromotionRequest,
  BuilderExportPromotionPlan,
  BuilderExportPromotionRequest,
} from "@/api/builder-sessions"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"

interface BuilderWorkspaceExportPanelProps {
  sessionExports: BuilderExport[]
  promotingExportId: string | null
  setPromotingExportId: React.Dispatch<React.SetStateAction<string | null>>
  promotionForm: BuilderExportPromotionRequest
  setPromotionForm: React.Dispatch<React.SetStateAction<BuilderExportPromotionRequest>>
  buildPromotionForm: BuilderExportInitialBuildPromotionRequest
  setBuildPromotionForm: React.Dispatch<React.SetStateAction<BuilderExportInitialBuildPromotionRequest>>
  deployBuildForm: BuilderExportDeployBuildRequest
  setDeployBuildForm: React.Dispatch<React.SetStateAction<BuilderExportDeployBuildRequest>>
  exportPromotionPlan?: BuilderExportPromotionPlan
  createExportPending: boolean
  promoteExportPending: boolean
  promoteExportToBuildPending: boolean
  deployExportBuildPending: boolean
  promotedBuildData?: { build: { id: string } }
  onCreateExport: () => void
  onDownloadExport: (exportItem: BuilderExport) => Promise<void>
  onPromoteExport: (exportId: string) => void
  onPromoteExportToBuild: (exportId: string) => void
  onDeployExportBuild: () => void
}

export function BuilderWorkspaceExportPanel({
  sessionExports,
  promotingExportId,
  setPromotingExportId,
  promotionForm,
  setPromotionForm,
  buildPromotionForm,
  setBuildPromotionForm,
  deployBuildForm,
  setDeployBuildForm,
  exportPromotionPlan,
  createExportPending,
  promoteExportPending,
  promoteExportToBuildPending,
  deployExportBuildPending,
  promotedBuildData,
  onCreateExport,
  onDownloadExport,
  onPromoteExport,
  onPromoteExportToBuild,
  onDeployExportBuild,
}: BuilderWorkspaceExportPanelProps) {
  return (
    <div className="rounded-2xl border bg-background px-4 py-3 text-sm text-muted-foreground shadow-sm">
      <div className="mb-2 flex items-center justify-between gap-3">
        <span className="font-medium text-foreground">Exports</span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onCreateExport}
          disabled={createExportPending}
        >
          Create export
        </Button>
      </div>
      {sessionExports.length === 0 ? (
        <div>No exports yet.</div>
      ) : (
        <div className="space-y-2">
          {sessionExports.map((exportItem) => (
            <div key={exportItem.id} className="space-y-2 rounded-lg border px-3 py-2">
              <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <div className="truncate text-foreground">{exportItem.file_name}</div>
                  <div className="text-xs text-muted-foreground">{exportItem.kind}</div>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => void onDownloadExport(exportItem)}
                  >
                    Download export
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setPromotingExportId((current) => current === exportItem.id ? null : exportItem.id)}
                  >
                    Promote to repository
                  </Button>
                </div>
              </div>
              {promotingExportId === exportItem.id ? (
                <div className="space-y-2 border-t pt-2">
                  <Input
                    name="builder_export_name"
                    placeholder="Repository name"
                    value={promotionForm.name ?? ""}
                    onInput={(event) => setPromotionForm((current) => ({ ...current, name: (event.target as HTMLInputElement).value }))}
                  />
                  <Input
                    name="builder_export_slug"
                    placeholder="Repository slug"
                    value={promotionForm.slug ?? ""}
                    onInput={(event) => setPromotionForm((current) => ({ ...current, slug: (event.target as HTMLInputElement).value }))}
                  />
                  <Input
                    name="builder_export_git_repo_url"
                    placeholder="Git repository URL"
                    value={promotionForm.git_repo_url}
                    onInput={(event) => {
                      const value = (event.target as HTMLInputElement).value
                      setPromotionForm((current) => ({ ...current, git_repo_url: value }))
                      setBuildPromotionForm((current) => ({ ...current, git_repo_url: value }))
                    }}
                  />
                  <Input
                    name="builder_export_git_username"
                    placeholder="Git username"
                    value={promotionForm.git_username ?? ""}
                    onInput={(event) => {
                      const value = (event.target as HTMLInputElement).value
                      setPromotionForm((current) => ({ ...current, git_username: value }))
                      setBuildPromotionForm((current) => ({ ...current, git_username: value }))
                    }}
                  />
                  <Input
                    name="builder_export_git_password"
                    placeholder="Git password"
                    value={promotionForm.git_password ?? ""}
                    onInput={(event) => {
                      const value = (event.target as HTMLInputElement).value
                      setPromotionForm((current) => ({ ...current, git_password: value }))
                      setBuildPromotionForm((current) => ({ ...current, git_password: value }))
                    }}
                  />
                  {exportPromotionPlan ? (
                    <div className="space-y-2 rounded-lg border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                      <div>Plan: {exportPromotionPlan.planned_project_kind}</div>
                      <div>Suggested env: {exportPromotionPlan.suggested_build_env_id}</div>
                      <div>Suggested image: {exportPromotionPlan.suggested_image_name}</div>
                      <div>Can trigger initial build: {exportPromotionPlan.can_trigger_initial_build ? "yes" : "no"}</div>
                      {exportPromotionPlan.missing_requirements.length > 0 ? (
                        <div>{exportPromotionPlan.missing_requirements.join(", ")}</div>
                      ) : null}
                    </div>
                  ) : null}
                  <Input
                    name="builder_export_registry_id"
                    placeholder="Container registry ID"
                    value={buildPromotionForm.registry_id}
                    onInput={(event) => setBuildPromotionForm((current) => ({ ...current, registry_id: (event.target as HTMLInputElement).value }))}
                  />
                  {promotedBuildData ? (
                    <div className="space-y-2 rounded-lg border bg-muted/20 px-3 py-2">
                      <div className="text-xs text-muted-foreground">Initial build ready: {promotedBuildData.build.id}</div>
                      <Input
                        name="builder_export_deploy_target_env_id"
                        placeholder="Target env ID"
                        value={deployBuildForm.target_env_id}
                        onInput={(event) => setDeployBuildForm((current) => ({ ...current, target_env_id: (event.target as HTMLInputElement).value }))}
                      />
                      <Input
                        name="builder_export_deploy_name"
                        placeholder="App name"
                        value={deployBuildForm.name ?? ""}
                        onInput={(event) => setDeployBuildForm((current) => ({ ...current, name: (event.target as HTMLInputElement).value }))}
                      />
                      <Input
                        name="builder_export_deploy_slug"
                        placeholder="App slug"
                        value={deployBuildForm.slug ?? ""}
                        onInput={(event) => setDeployBuildForm((current) => ({ ...current, slug: (event.target as HTMLInputElement).value }))}
                      />
                    </div>
                  ) : null}
                  <div className="flex justify-end">
                    <div className="flex gap-2">
                      <Button
                        type="button"
                        size="sm"
                        onClick={() => onPromoteExport(exportItem.id)}
                        disabled={promoteExportPending || !promotionForm.git_repo_url}
                      >
                        Promote export
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        onClick={() => onPromoteExportToBuild(exportItem.id)}
                        disabled={
                          promoteExportToBuildPending ||
                          !buildPromotionForm.git_repo_url ||
                          !buildPromotionForm.registry_id ||
                          !buildPromotionForm.build_env_id
                        }
                      >
                        Promote to initial build
                      </Button>
                      {promotedBuildData ? (
                        <Button
                          type="button"
                          size="sm"
                          onClick={onDeployExportBuild}
                          disabled={
                            deployExportBuildPending ||
                            !deployBuildForm.repository_id ||
                            !deployBuildForm.build_id ||
                            !deployBuildForm.target_env_id
                          }
                        >
                          Deploy build
                        </Button>
                      ) : null}
                    </div>
                  </div>
                </div>
              ) : null}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
