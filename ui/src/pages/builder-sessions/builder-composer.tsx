import { AlertCircle, ArrowUp } from "lucide-react"
import * as React from "react"

import type { BuilderModelOption } from "./builder-model-selector"
import { BuilderModelSelector } from "./builder-model-selector"

import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"

interface BuilderComposerProps {
  value: string
  onValueChange: (value: string) => void
  onSubmit: () => void
  isSubmitting: boolean
  modelValue: string | null
  modelOptions: BuilderModelOption[]
  onModelValueChange: (value: string | null) => void
  modelSelectionHint?: string
  modelError?: string
  statusText?: string
  statusError?: string
  showModelSelector?: boolean
  centered?: boolean
  composerRef?: React.RefObject<HTMLTextAreaElement | null>
}

export function BuilderComposer({
  value,
  onValueChange,
  onSubmit,
  isSubmitting,
  modelValue,
  modelOptions,
  onModelValueChange,
  modelSelectionHint,
  modelError,
  statusText,
  statusError,
  showModelSelector = true,
  centered = false,
  composerRef,
}: BuilderComposerProps) {
  const helperMessage = statusError || modelError || statusText || modelSelectionHint
  const helperTone = statusError || modelError ? "text-destructive" : "text-muted-foreground"
  const shellClassName = centered
    ? "w-full max-w-3xl rounded-lg border bg-background shadow-black/5"
    : "rounded-lg border bg-background"

  return (
    <div
      data-testid="builder-composer-shell"
      className={centered ? "w-full shrink-0" : "shrink-0 bg-background/95 px-4 pt-4 backdrop-blur supports-backdrop-filter:bg-background/80"}
    >
      <div className={shellClassName}>
        <Textarea
          ref={composerRef}
          data-testid="builder-composer"
          className={`min-h-28 border-0 bg-transparent px-4 py-4 text-sm shadow-none focus-visible:ring-0 ${centered ? "min-h-32 px-5 py-5 text-[15px]" : ""}`}
          placeholder="Describe what you want to build or change..."
          value={value}
          onChange={(event) => onValueChange(event.target.value)}
          onInput={(event) => onValueChange(event.currentTarget.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault()
              onSubmit()
            }
          }}
          disabled={isSubmitting}
        />

        <div
          data-testid="builder-composer-footer"
          className="flex items-center justify-between gap-3 border-t px-3 py-2 sm:px-4"
        >
          <div className="min-w-0 flex-1">
            {showModelSelector ? (
              <BuilderModelSelector
                value={modelValue}
                options={modelOptions}
                onValueChange={onModelValueChange}
                compact
                helperText={undefined}
                errorText={modelError}
              />
            ) : helperMessage ? (
              <div
                data-testid="builder-composer-inline-message"
                className={`flex items-start gap-2 text-sm ${helperTone}`}
              >
                {statusError ? <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" /> : null}
                <span>{helperMessage}</span>
              </div>
            ) : null}
          </div>

          <Button
            data-testid="builder-send-message"
            className="shrink-0 rounded-full"
            onClick={onSubmit}
            size="icon"
            disabled={isSubmitting}
          >
            <ArrowUp />
          </Button>
        </div>

        {/* {showModelSelector ? (
          helperMessage && (statusText || statusError) ? (
            <div
              data-testid="builder-composer-inline-message"
              className={`flex items-start gap-2 px-4 pb-3 text-sm ${helperTone}`}
            >
              {statusError ? <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" /> : null}
              <span>{statusError || statusText}</span>
            </div>
          ) : null
        ) : null} */}
      </div>
    </div>
  )
}
