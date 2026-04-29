export interface PluginResourceValues {
  request_cpu: number
  request_memory: number
  limit_cpu: number
  limit_memory: number
}

export interface PluginResourcePresetOption extends PluginResourceValues {
  value: string
  label: string
  description: string
  group: "Recommended" | "Specialized"
}

export const CUSTOM_PLUGIN_RESOURCE_PRESET_VALUE = "custom"

export const PLUGIN_RESOURCE_PRESET_OPTIONS: PluginResourcePresetOption[] = [
  {
    value: "lightweight",
    label: "Lightweight",
    description: "50m / 64Mi request, 200m / 128Mi limit. Best for tiny helpers.",
    group: "Recommended",
    request_cpu: 50,
    request_memory: 64,
    limit_cpu: 200,
    limit_memory: 128,
  },
  {
    value: "standard",
    label: "Standard",
    description: "100m / 128Mi request, 500m / 256Mi limit. Balanced default for most plugins.",
    group: "Recommended",
    request_cpu: 100,
    request_memory: 128,
    limit_cpu: 500,
    limit_memory: 256,
  },
  {
    value: "compute-heavy",
    label: "Compute Heavy",
    description: "250m / 128Mi request, 1000m / 256Mi limit. Better for CPU-intensive jobs.",
    group: "Specialized",
    request_cpu: 250,
    request_memory: 128,
    limit_cpu: 1000,
    limit_memory: 256,
  },
  {
    value: "memory-heavy",
    label: "Memory Heavy",
    description: "100m / 256Mi request, 500m / 512Mi limit. Better for cache or parsing workloads.",
    group: "Specialized",
    request_cpu: 100,
    request_memory: 256,
    limit_cpu: 500,
    limit_memory: 512,
  },
]

export const DEFAULT_PLUGIN_RESOURCE_VALUES: PluginResourceValues = {
  request_cpu: 100,
  request_memory: 128,
  limit_cpu: 500,
  limit_memory: 256,
}

function toPluginResourceValues(
  values?: Partial<PluginResourceValues> | null,
): PluginResourceValues {
  return {
    request_cpu: values?.request_cpu ?? 0,
    request_memory: values?.request_memory ?? 0,
    limit_cpu: values?.limit_cpu ?? 0,
    limit_memory: values?.limit_memory ?? 0,
  }
}

function hasZeroedPluginResources(values: PluginResourceValues) {
  return values.request_cpu === 0
    && values.request_memory === 0
    && values.limit_cpu === 0
    && values.limit_memory === 0
}

export function normalizePluginResourceValues(
  values?: Partial<PluginResourceValues> | null,
): PluginResourceValues {
  const normalizedValues = toPluginResourceValues(values)

  if (hasZeroedPluginResources(normalizedValues)) {
    return { ...DEFAULT_PLUGIN_RESOURCE_VALUES }
  }

  return normalizedValues
}

export function getPluginResourcePreset(
  values: PluginResourceValues,
): PluginResourcePresetOption | undefined {
  return PLUGIN_RESOURCE_PRESET_OPTIONS.find((option) =>
    option.request_cpu === values.request_cpu
    && option.request_memory === values.request_memory
    && option.limit_cpu === values.limit_cpu
    && option.limit_memory === values.limit_memory,
  )
}

export function getPluginResourcePresetLabel(value: string | null | undefined) {
  if (value === CUSTOM_PLUGIN_RESOURCE_PRESET_VALUE) {
    return "Custom"
  }

  return PLUGIN_RESOURCE_PRESET_OPTIONS.find((option) => option.value === value)?.label ?? value ?? ""
}
