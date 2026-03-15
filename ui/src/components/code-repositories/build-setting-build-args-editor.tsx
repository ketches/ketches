import type { BuildArgPair } from "@/api/code-repositories"
import { KeyValueInput } from "@/components/shared/key-value-input"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"

interface BuildSettingBuildArgsEditorProps {
  mode: "structured" | "advanced"
  pairs: BuildArgPair[]
  raw: string
  onModeChange: (mode: "structured" | "advanced") => void
  onPairsChange: (pairs: BuildArgPair[]) => void
  onRawChange: (raw: string) => void
}

export function BuildSettingBuildArgsEditor({
  mode,
  pairs,
  raw,
  onModeChange,
  onPairsChange,
  onRawChange,
}: BuildSettingBuildArgsEditorProps) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant={mode === "structured" ? "default" : "outline"}
          onClick={() => onModeChange("structured")}
        >
          Structured
        </Button>
        <Button
          type="button"
          variant={mode === "advanced" ? "default" : "outline"}
          onClick={() => onModeChange("advanced")}
        >
          Advanced
        </Button>
      </div>

      {mode === "structured" ? (
        <KeyValueInput
          value={pairs}
          onChange={onPairsChange}
          keyPlaceholder="ARG_KEY"
          valuePlaceholder="ARG_VALUE"
        />
      ) : (
        <Textarea
          value={raw}
          onChange={(event) => onRawChange(event.target.value)}
          placeholder={"ALPHA=first\nZETA=last"}
          className="min-h-40 font-mono"
        />
      )}
    </div>
  )
}
