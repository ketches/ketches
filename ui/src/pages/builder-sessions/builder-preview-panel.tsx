import { CircleAlert, Download, Eye, FileOutput } from "lucide-react"

import type { BuilderPreviewSummary } from "@/api/builder-sessions"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

interface BuilderPreviewPanelProps {
  preview?: BuilderPreviewSummary
  onDownload: () => void
  onOpenPreview: () => void
}

export function BuilderPreviewPanel({ preview, onDownload, onOpenPreview }: BuilderPreviewPanelProps) {
  const resolvedPreview = preview ?? {
    available: false,
    status: "unavailable",
    resolved_run_id: "",
    published_at: null,
    completed_at: null,
    output_root: "",
    default_entry_path: "",
    download_available: false,
    preview_available: false,
    is_stale: false,
    newer_run_id: "",
    newer_run_status: "",
  }

  return (
    <Card data-testid="builder-preview-panel" className="border-blue-200/70 bg-linear-to-br from-blue-50 via-background to-background shadow-none">
      <CardHeader className="space-y-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <FileOutput className="h-4 w-4 text-blue-600" />
          <span>Preview output</span>
        </CardTitle>
        <CardDescription>
          Durable output from the latest successful Builder snapshot, separate from the live workspace files.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {!resolvedPreview.available ? (
          <div className="rounded-lg border border-dashed bg-muted/20 px-4 py-4 text-sm text-muted-foreground">
            No successful preview is available yet.
          </div>
        ) : (
          <>
            <div className="space-y-1 text-sm">
              <div className="font-medium text-foreground">Latest durable preview</div>
              <div className="text-muted-foreground">Run {resolvedPreview.resolved_run_id}</div>
              {resolvedPreview.output_root ? (
                <div className="text-muted-foreground">Output root: {resolvedPreview.output_root}</div>
              ) : null}
            </div>

            {resolvedPreview.status === "delivery_only" ? (
              <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
                Preview is unavailable for this output, but the snapshot can still be downloaded.
              </div>
            ) : null}

            {resolvedPreview.is_stale ? (
              <div className="rounded-lg border border-orange-200 bg-orange-50 px-4 py-3 text-sm text-orange-900">
                <div className="flex items-start gap-2">
                  <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
                  <div>
                    <div className="font-medium">A newer run exists</div>
                    <div>
                      {resolvedPreview.newer_run_id} · {resolvedPreview.newer_run_status}
                    </div>
                  </div>
                </div>
              </div>
            ) : null}

            <div className="flex flex-wrap gap-2">
              {resolvedPreview.preview_available ? (
                <Button type="button" onClick={onOpenPreview}>
                  <Eye className="h-4 w-4" />
                  Open preview
                </Button>
              ) : null}
              {resolvedPreview.download_available ? (
                <Button type="button" variant="outline" onClick={onDownload}>
                  <Download className="h-4 w-4" />
                  Download snapshot
                </Button>
              ) : null}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}
