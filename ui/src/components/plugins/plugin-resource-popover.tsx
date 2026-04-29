import { useMutation, useQueryClient } from "@tanstack/react-query"
import { type AxiosError } from "axios"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { pluginsApi, type AppPlugin } from "@/api/plugins"
import { Button } from "@/components/ui/button"
import {
  Combobox,
  ComboboxContent,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxLabel,
  ComboboxList,
} from "@/components/ui/combobox"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import {
  CUSTOM_PLUGIN_RESOURCE_PRESET_VALUE,
  getPluginResourcePreset,
  getPluginResourcePresetLabel,
  normalizePluginResourceValues,
  PLUGIN_RESOURCE_PRESET_OPTIONS,
  type PluginResourceValues,
} from "@/lib/plugin-resources"

interface PluginResourcePopoverProps {
  appId: string
  appPlugin: AppPlugin
  children: React.ReactNode
}

export function PluginResourcePopover({
  appId,
  appPlugin,
  children,
}: PluginResourcePopoverProps) {
  const queryClient = useQueryClient()
  const [open, setOpen] = React.useState(false)
  const [formData, setFormData] = React.useState<PluginResourceValues>(() =>
    normalizePluginResourceValues(appPlugin),
  )

  React.useEffect(() => {
    if (open) {
      setFormData(normalizePluginResourceValues(appPlugin))
    }
  }, [open, appPlugin])

  const updateMutation = useMutation<unknown, AxiosError<{ error: string }>, PluginResourceValues>({
    mutationFn: (resources) => pluginsApi.updateAppPluginResources(appId, appPlugin.plugin_id, resources),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-plugins", appId] })
      toast.success("Plugin resources updated")
      setOpen(false)
    },
    onError: (error) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update resources",
      })
    },
  })

  const selectedPresetValue = getPluginResourcePreset(formData)?.value ?? CUSTOM_PLUGIN_RESOURCE_PRESET_VALUE
  const hasInvalidResourceValues = Object.values(formData).some((value) => value <= 0)

  const handlePresetChange = (value: string | null | undefined) => {
    const selectedOption = PLUGIN_RESOURCE_PRESET_OPTIONS.find((option) => option.value === value)
    if (!selectedOption) {
      return
    }

    setFormData({
      request_cpu: selectedOption.request_cpu,
      request_memory: selectedOption.request_memory,
      limit_cpu: selectedOption.limit_cpu,
      limit_memory: selectedOption.limit_memory,
    })
  }

  const handleResourceInputChange = (
    field: keyof PluginResourceValues,
    value: string,
  ) => {
    const parsedValue = Number.parseInt(value, 10)
    setFormData((current) => ({
      ...current,
      [field]: Number.isNaN(parsedValue) ? 0 : parsedValue,
    }))
  }

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    updateMutation.mutate(formData)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger>{children}</PopoverTrigger>
      <PopoverContent className="w-80">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <h4 className="font-medium text-sm">Plugin Resources</h4>
            <p className="text-xs text-muted-foreground">
              Configure CPU and memory for {appPlugin.plugin.name}
            </p>
          </div>

          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="plugin-resource-preset" className="text-xs">
                Resource Preset
              </FieldLabel>
              <FieldContent>
                <Combobox
                  value={selectedPresetValue}
                  onValueChange={handlePresetChange}
                  itemToStringLabel={getPluginResourcePresetLabel}
                >
                  <ComboboxInput
                    id="plugin-resource-preset"
                    readOnly
                    className="w-full cursor-pointer"
                  />
                  <ComboboxContent>
                    <ComboboxList>
                      <ComboboxGroup>
                        <ComboboxLabel>Recommended</ComboboxLabel>
                        {PLUGIN_RESOURCE_PRESET_OPTIONS
                          .filter((option) => option.group === "Recommended")
                          .map((option) => (
                            <ComboboxItem key={option.value} value={option.value}>
                              <Item size="xs" className="p-0">
                                <ItemContent>
                                  <ItemTitle>{option.label}</ItemTitle>
                                  <ItemDescription>{option.description}</ItemDescription>
                                </ItemContent>
                              </Item>
                            </ComboboxItem>
                          ))}
                      </ComboboxGroup>
                      <ComboboxGroup>
                        <ComboboxLabel>Specialized</ComboboxLabel>
                        {PLUGIN_RESOURCE_PRESET_OPTIONS
                          .filter((option) => option.group === "Specialized")
                          .map((option) => (
                            <ComboboxItem key={option.value} value={option.value}>
                              <Item size="xs" className="p-0">
                                <ItemContent>
                                  <ItemTitle>{option.label}</ItemTitle>
                                  <ItemDescription>{option.description}</ItemDescription>
                                </ItemContent>
                              </Item>
                            </ComboboxItem>
                          ))}
                      </ComboboxGroup>
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
                <FieldDescription>
                  Choosing a preset fills the fields below. You can still fine-tune the values manually.
                </FieldDescription>
              </FieldContent>
            </Field>

            <div className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel htmlFor="request_cpu" className="text-xs">
                  CPU Request (m)
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="request_cpu"
                    type="number"
                    min="1"
                    value={formData.request_cpu}
                    onChange={(e) => handleResourceInputChange("request_cpu", e.target.value)}
                    className="h-8"
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="limit_cpu" className="text-xs">
                  CPU Limit (m)
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="limit_cpu"
                    type="number"
                    min="1"
                    value={formData.limit_cpu}
                    onChange={(e) => handleResourceInputChange("limit_cpu", e.target.value)}
                    className="h-8"
                  />
                </FieldContent>
              </Field>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel htmlFor="request_memory" className="text-xs">
                  Memory Request (Mi)
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="request_memory"
                    type="number"
                    min="1"
                    value={formData.request_memory}
                    onChange={(e) => handleResourceInputChange("request_memory", e.target.value)}
                    className="h-8"
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="limit_memory" className="text-xs">
                  Memory Limit (Mi)
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="limit_memory"
                    type="number"
                    min="1"
                    value={formData.limit_memory}
                    onChange={(e) => handleResourceInputChange("limit_memory", e.target.value)}
                    className="h-8"
                  />
                </FieldContent>
              </Field>
            </div>
          </FieldGroup>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={updateMutation.isPending || hasInvalidResourceValues}>
              {updateMutation.isPending ? (
                <>
                  <Loader2 className="h-3 w-3 animate-spin mr-1.5" />
                  Saving...
                </>
              ) : (
                "Save"
              )}
            </Button>
          </div>
        </form>
      </PopoverContent>
    </Popover>
  )
}
