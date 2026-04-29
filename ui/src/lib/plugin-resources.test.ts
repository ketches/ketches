import { describe, expect, it } from "vitest"

import {
  CUSTOM_PLUGIN_RESOURCE_PRESET_VALUE,
  DEFAULT_PLUGIN_RESOURCE_VALUES,
  getPluginResourcePreset,
  getPluginResourcePresetLabel,
  normalizePluginResourceValues,
} from "@/lib/plugin-resources"

describe("plugin resource helpers", () => {
  it("falls back to default values when plugin resources are empty", () => {
    expect(normalizePluginResourceValues()).toEqual(DEFAULT_PLUGIN_RESOURCE_VALUES)
    expect(normalizePluginResourceValues({
      request_cpu: 0,
      request_memory: 0,
      limit_cpu: 0,
      limit_memory: 0,
    })).toEqual(DEFAULT_PLUGIN_RESOURCE_VALUES)
  })

  it("keeps explicit resource values unchanged", () => {
    expect(normalizePluginResourceValues({
      request_cpu: 80,
      request_memory: 96,
      limit_cpu: 300,
      limit_memory: 192,
    })).toEqual({
      request_cpu: 80,
      request_memory: 96,
      limit_cpu: 300,
      limit_memory: 192,
    })
  })

  it("matches the standard preset and resolves labels", () => {
    const preset = getPluginResourcePreset(DEFAULT_PLUGIN_RESOURCE_VALUES)

    expect(preset?.value).toBe("standard")
    expect(getPluginResourcePresetLabel(preset?.value)).toBe("Standard")
    expect(getPluginResourcePresetLabel(CUSTOM_PLUGIN_RESOURCE_PRESET_VALUE)).toBe("Custom")
  })
})
