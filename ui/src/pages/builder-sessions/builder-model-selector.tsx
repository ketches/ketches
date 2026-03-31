import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Field, FieldContent, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { InputGroupAddon } from "@/components/ui/input-group"
import { Brain } from "lucide-react"

export interface BuilderModelOption {
  key: string
  modelLabel: string
  providerLabel: string
  scope: "project" | "user"
  providerKey: string
  modelProfileKey: string
}

interface BuilderModelSelectorProps {
  value: string | null
  options: BuilderModelOption[]
  onValueChange: (value: string | null) => void
  compact?: boolean
  helperText?: string
  errorText?: string
}

function groupLabel(scope: BuilderModelOption["scope"]) {
  return scope === "project" ? "Project models" : "My models"
}

function groupOptions(options: BuilderModelOption[]) {
  return options.reduce<Record<string, BuilderModelOption[]>>((groups, option) => {
    const label = groupLabel(option.scope)
    groups[label] ??= []
    groups[label].push(option)
    return groups
  }, {})
}

function ModelCombobox({
  value,
  options,
  onValueChange,
  className,
}: {
  value: string | null
  options: BuilderModelOption[]
  onValueChange: (value: string | null) => void
  className: string
}) {
  const groupedOptions = groupOptions(options)

  return (
    <Combobox
      value={value}
      onValueChange={onValueChange}
      itemToStringLabel={(selectedValue: string) => {
        const selected = options.find((option) => option.key === selectedValue)
        return selected?.modelLabel ?? selectedValue ?? ""
      }}
    >
      <ComboboxInput id="builder-model-selector" placeholder="Select a model" className={className} >
        <InputGroupAddon>
          <Brain />
        </InputGroupAddon>
      </ComboboxInput>
      <ComboboxContent alignOffset={-24} className="w-auto">
        <ComboboxEmpty>No models found.</ComboboxEmpty>
        <ComboboxList>
          {Object.entries(groupedOptions).map(([group, groupOptions]) => (
            <div key={group} className="px-2 py-1.5">
              <div className="px-2 py-1 text-xs font-medium text-muted-foreground">{group}</div>
              {groupOptions.map((option) => (
                <ComboboxItem key={option.key} value={option.key}>
                  <div className="flex min-w-0 flex-col">
                    <span>{option.modelLabel}</span>
                    <span className="text-xs text-muted-foreground">
                      {option.providerLabel} · {option.scope === "project" ? "Project" : "User"}
                    </span>
                  </div>
                </ComboboxItem>
              ))}
            </div>
          ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  )
}

export function BuilderModelSelector({
  value,
  options,
  onValueChange,
  compact = false,
  helperText,
  errorText,
}: BuilderModelSelectorProps) {
  const selectedOption = options.find((option) => option.key === value) ?? null

  if (compact) {
    return (
      <div data-testid="builder-model-selector-compact" className="space-y-1">
        <div className="flex min-w-0 items-center gap-2">
          <ModelCombobox value={value} options={options} onValueChange={onValueChange} className="w-fit text-sm" />
          <span className="sr-only" data-testid="builder-model-selector-selection" aria-live="polite">
            {selectedOption ? `${selectedOption.modelLabel} · ${selectedOption.providerLabel}` : "No model selected"}
          </span>
        </div>
        {errorText ? <FieldError>{errorText}</FieldError> : null}
      </div>
    )
  }

  return (
    <Field>
      <FieldLabel htmlFor="builder-model-selector">Model</FieldLabel>
      <FieldContent>
        <ModelCombobox value={value} options={options} onValueChange={onValueChange} className="w-full" />
      </FieldContent>
      {errorText ? <FieldError>{errorText}</FieldError> : null}
      {helperText ? <FieldDescription>{helperText}</FieldDescription> : null}
    </Field>
  )
}
