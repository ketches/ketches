import * as React from "react"

import type {
  BuildArgPair,
  BuildSetting,
  CreateBuildSettingRequest,
  UpdateBuildSettingRequest,
} from "@/api/code-repositories"
import { registryProviderLabels, type ContainerRegistry } from "@/api/container-registries"
import { GitRefSelect } from "@/components/code-repositories/git-ref-select"
import { BuildSettingBuildArgsEditor } from "@/components/code-repositories/build-setting-build-args-editor"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from "@/components/ui/sheet"
import {
  parseBuildArgs,
  serializeBuildArgPairs,
  validateBuildArgPairs,
} from "@/lib/build-setting-build-args"

const PLATFORM_OPTIONS = [
  { label: "linux/amd64", value: "linux/amd64" },
  { label: "linux/amd64,linux/arm64", value: "linux/amd64,linux/arm64" },
] as const

export type BuildSettingSheetSubmitPayload = CreateBuildSettingRequest & UpdateBuildSettingRequest

interface BuildSettingSheetProps {
  mode: "create" | "edit"
  open: boolean
  onOpenChange: (open: boolean) => void
  repoId: string
  repoSlug?: string
  registries: Pick<ContainerRegistry, "id" | "name" | "provider">[]
  setting?: BuildSetting | null
  isPending?: boolean
  onSubmit: (payload: BuildSettingSheetSubmitPayload) => void
}

type FormState = {
  name: string
  git_ref: string
  dockerfile_path: string
  build_context: string
  image_name: string
  registry_id: string
  platforms: string
  registry_cache_enabled: boolean
  registry_cache_ref: string
}

const defaultFormState: FormState = {
  name: "Default",
  git_ref: "",
  dockerfile_path: "Dockerfile",
  build_context: ".",
  image_name: "",
  registry_id: "",
  platforms: "linux/amd64",
  registry_cache_enabled: true,
  registry_cache_ref: "",
}

export function BuildSettingSheet({
  mode,
  open,
  onOpenChange,
  repoId,
  repoSlug,
  registries,
  setting,
  isPending,
  onSubmit,
}: BuildSettingSheetProps) {
  const [form, setForm] = React.useState<FormState>(defaultFormState)
  const [buildArgMode, setBuildArgMode] = React.useState<"structured" | "advanced">("structured")
  const [buildArgPairs, setBuildArgPairs] = React.useState<BuildArgPair[]>([])
  const [buildArgRaw, setBuildArgRaw] = React.useState("")

  React.useEffect(() => {
    if (!open) return

    if (mode === "edit" && setting) {
      const parsed = setting.build_arg_pairs?.length
        ? { mode: "structured" as const, pairs: setting.build_arg_pairs, raw: setting.build_args ?? "" }
        : parseBuildArgs(setting.build_args ?? "")

      setForm({
        name: setting.name,
        git_ref: setting.git_ref ?? "",
        dockerfile_path: setting.dockerfile_path ?? "Dockerfile",
        build_context: setting.build_context ?? ".",
        image_name: setting.image_name ?? "",
        registry_id: setting.registry_id ?? "",
        platforms: setting.platforms || "linux/amd64",
        registry_cache_enabled: setting.registry_cache_enabled ?? true,
        registry_cache_ref: setting.registry_cache_ref ?? "",
      })
      setBuildArgMode(parsed.mode)
      setBuildArgPairs(parsed.pairs)
      setBuildArgRaw(parsed.raw)
      return
    }

    const nextForm = {
      ...defaultFormState,
      image_name: repoSlug ?? "",
    }
    setForm(nextForm)
    setBuildArgMode("structured")
    setBuildArgPairs([])
    setBuildArgRaw("")
  }, [mode, open, repoSlug, setting])

  const handleBuildArgModeChange = React.useCallback((nextMode: "structured" | "advanced") => {
    if (nextMode === "advanced") {
      setBuildArgRaw((current) => current || serializeBuildArgPairs(buildArgPairs))
      setBuildArgMode("advanced")
      return
    }

    const parsed = parseBuildArgs(buildArgRaw || serializeBuildArgPairs(buildArgPairs))
    if (parsed.mode === "structured") {
      setBuildArgPairs(parsed.pairs)
      setBuildArgMode("structured")
    }
  }, [buildArgPairs, buildArgRaw])

  const handleSubmit = () => {
    if (!form.name.trim() || !form.image_name.trim() || !form.registry_id.trim()) {
      return
    }

    if (buildArgMode === "structured") {
      const validationError = validateBuildArgPairs(buildArgPairs)
      if (validationError) {
        return
      }
    }

    const normalizedPairs = buildArgPairs
      .map((pair) => ({ key: pair.key.trim(), value: pair.value.trim() }))
      .filter((pair) => pair.key)
      .sort((a, b) => a.key.localeCompare(b.key))

    onSubmit({
      name: form.name.trim(),
      git_ref: form.git_ref.trim(),
      dockerfile_path: form.dockerfile_path.trim(),
      build_context: form.build_context.trim(),
      image_name: form.image_name.trim(),
      registry_id: form.registry_id,
      platforms: form.platforms,
      registry_cache_enabled: form.registry_cache_enabled,
      registry_cache_ref: form.registry_cache_ref.trim(),
      build_args: buildArgMode === "structured" ? serializeBuildArgPairs(normalizedPairs) : buildArgRaw.trim(),
      build_arg_pairs: buildArgMode === "structured" ? normalizedPairs : undefined,
    })
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="sm:max-w-xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{mode === "create" ? "Add Build Setting" : "Edit Build Setting"}</SheetTitle>
          <SheetDescription>
            Configure source, image output, build platforms, cache, and build args.
          </SheetDescription>
        </SheetHeader>

        <div className="space-y-6 px-4 pb-4">
          <section className="space-y-4">
            <div>
              <h3 className="text-sm font-medium">Source</h3>
            </div>
            <Field>
              <FieldLabel>Name *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="e.g. backend, frontend"
                  value={form.name}
                  onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Git Branch / Tag</FieldLabel>
              <FieldContent>
                <GitRefSelect
                  repoId={repoId}
                  value={form.git_ref}
                  onValueChange={(value) => setForm((current) => ({ ...current, git_ref: value ?? "" }))}
                  className="w-full"
                />
              </FieldContent>
            </Field>
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel>Dockerfile path</FieldLabel>
                <FieldContent>
                  <Input
                    value={form.dockerfile_path}
                    onChange={(event) => setForm((current) => ({ ...current, dockerfile_path: event.target.value }))}
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel>Build context</FieldLabel>
                <FieldContent>
                  <Input
                    value={form.build_context}
                    onChange={(event) => setForm((current) => ({ ...current, build_context: event.target.value }))}
                  />
                </FieldContent>
              </Field>
            </div>
          </section>

          <section className="space-y-4">
            <div>
              <h3 className="text-sm font-medium">Image Output</h3>
            </div>
            <Field>
              <FieldLabel>Image name *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="my-service"
                  value={form.image_name}
                  onChange={(event) => setForm((current) => ({ ...current, image_name: event.target.value }))}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Container registry *</FieldLabel>
              <FieldContent>
                <Combobox
                  value={form.registry_id}
                  onValueChange={(value) => setForm((current) => ({ ...current, registry_id: value ?? "" }))}
                  itemToStringLabel={(value) => registries.find((registry) => registry.id === value)?.name ?? value ?? ""}
                >
                  <ComboboxInput placeholder="Select registry" />
                  <ComboboxContent>
                    <ComboboxList>
                      {registries.map((registry) => (
                        <ComboboxItem key={registry.id} value={registry.id}>
                          {registry.name} ({registryProviderLabels[registry.provider]})
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </section>

          <section className="space-y-4">
            <div>
              <h3 className="text-sm font-medium">Platforms & Cache</h3>
            </div>
            <Field>
              <FieldLabel>Target platforms</FieldLabel>
              <FieldContent>
                <Combobox
                  value={form.platforms}
                  onValueChange={(value) => setForm((current) => ({ ...current, platforms: value ?? "linux/amd64" }))}
                  itemToStringLabel={(value) => PLATFORM_OPTIONS.find((option) => option.value === value)?.label ?? value ?? ""}
                >
                  <ComboboxInput placeholder="Select platform target" />
                  <ComboboxContent>
                    <ComboboxList>
                      {PLATFORM_OPTIONS.map((option) => (
                        <ComboboxItem key={option.value} value={option.value}>
                          {option.label}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Registry cache</FieldLabel>
              <FieldContent>
                <Checkbox
                  checked={form.registry_cache_enabled}
                  onCheckedChange={(checked) => setForm((current) => ({ ...current, registry_cache_enabled: checked }))}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Registry cache ref</FieldLabel>
              <FieldContent>
                <Input
                  value={form.registry_cache_ref}
                  onChange={(event) => setForm((current) => ({ ...current, registry_cache_ref: event.target.value }))}
                  placeholder="registry.example.com/demo/app:buildcache-setting-1"
                />
              </FieldContent>
            </Field>
          </section>

          <section className="space-y-4">
            <div>
              <h3 className="text-sm font-medium">Build Args</h3>
            </div>
            <BuildSettingBuildArgsEditor
              mode={buildArgMode}
              pairs={buildArgPairs}
              raw={buildArgRaw}
              onModeChange={handleBuildArgModeChange}
              onPairsChange={setBuildArgPairs}
              onRawChange={setBuildArgRaw}
            />
          </section>
        </div>

        <SheetFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSubmit} disabled={isPending}>
            {mode === "create" ? "Add" : "Save"}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
