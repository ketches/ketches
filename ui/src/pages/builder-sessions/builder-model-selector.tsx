import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Field, FieldContent, FieldDescription, FieldLabel } from "@/components/ui/field"

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
}

function groupLabel(scope: BuilderModelOption["scope"]) {
  return scope === "project" ? "Project models" : "My models"
}

export function BuilderModelSelector({ value, options, onValueChange }: BuilderModelSelectorProps) {
  const groupedOptions = options.reduce<Record<string, BuilderModelOption[]>>((groups, option) => {
    const label = groupLabel(option.scope)
    groups[label] ??= []
    groups[label].push(option)
    return groups
  }, {})

  return (
    <Field>
      <FieldLabel htmlFor="builder-model-selector">Model</FieldLabel>
      <FieldContent>
        <Combobox
          value={value}
          onValueChange={onValueChange}
          itemToStringLabel={(selectedValue: string) => {
            const selected = options.find((option) => option.key === selectedValue)
            return selected?.modelLabel ?? selectedValue ?? ""
          }}
        >
          <ComboboxInput id="builder-model-selector" placeholder="Select a model" className="w-full" />
          <ComboboxContent>
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
        <div className="mt-3 space-y-2" data-testid="builder-model-selector-groups">
          {Object.entries(groupedOptions).map(([group, groupOptions]) => (
            <div key={group} className="space-y-1">
              <div className="text-xs font-medium text-muted-foreground">{group}</div>
              {groupOptions.map((option) => (
                <div key={option.key} className="text-sm">
                  <div>{option.modelLabel}</div>
                  <div className="text-xs text-muted-foreground">
                    {option.providerLabel} · {option.scope === "project" ? "Project" : "User"}
                  </div>
                </div>
              ))}
            </div>
          ))}
        </div>
      </FieldContent>
      <FieldDescription>Select which configured model Builder should use for this conversation.</FieldDescription>
    </Field>
  )
}
