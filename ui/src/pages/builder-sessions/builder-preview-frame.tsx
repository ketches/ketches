interface BuilderPreviewFrameProps {
  frameUrl: string
}

export function BuilderPreviewFrame({ frameUrl }: BuilderPreviewFrameProps) {
  return (
    <div className="overflow-hidden rounded-lg border bg-background shadow-sm" data-testid="builder-preview-frame-shell">
      <iframe
        title="Builder preview"
        data-testid="builder-preview-iframe"
        src={frameUrl}
        sandbox="allow-scripts"
        className="h-[420px] w-full border-0 bg-white"
      />
    </div>
  )
}
