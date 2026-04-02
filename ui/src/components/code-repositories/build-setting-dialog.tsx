import * as React from "react"

import type {
  BuildArgPair,
  BuildSetting,
  CreateBuildSettingRequest,
  UpdateBuildSettingRequest,
} from "@/api/code-repositories"
import { registryProviderLabels, type ContainerRegistry } from "@/api/container-registries"
import { GitRefSelect } from "@/components/code-repositories/git-ref-select"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  parseBuildArgs,
  serializeBuildArgPairs,
  validateBuildArgPairs,
} from "@/lib/build-setting-build-args"
import { Textarea } from "@/components/ui/textarea"
import { KeyValueInput } from "../shared/key-value-input"

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
  const [buildArgPairs, setBuildArgPairs] = React.useState<BuildArgPair[]>([])
  const [buildArgsMode, setBuildArgsMode] = React.useState<"structured" | "advanced">("structured")
  const [buildArgsRaw, setBuildArgsRaw] = React.useState("")

  React.useEffect(() => {
    if (!open) return

    if (mode === "edit" && setting) {
      const parsed = setting.build_arg_pairs?.length
        ? { mode: "structured" as const, pairs: setting.build_arg_pairs, raw: "" }
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
      setBuildArgPairs(parsed.pairs)
      setBuildArgsMode(parsed.mode)
      setBuildArgsRaw(parsed.raw)
      return
    }

    const nextForm = {
      ...defaultFormState,
      image_name: repoSlug ?? "",
    }
    setForm(nextForm)
    setBuildArgPairs([])
    setBuildArgsMode("structured")
    setBuildArgsRaw("")
  }, [mode, open, repoSlug, setting])


  const handleSubmit = () => {
    if (!form.name.trim() || !form.image_name.trim() || !form.registry_id.trim()) {
      return
    }

    let finalBuildArgs = ""
    let finalPairs: BuildArgPair[] | undefined = undefined

    if (buildArgsMode === "structured") {
      const validationError = validateBuildArgPairs(buildArgPairs)
      if (validationError) {
        return
      }

      const normalizedPairs = buildArgPairs
        .map((pair) => ({ key: pair.key.trim(), value: pair.value.trim() }))
        .filter((pair) => pair.key)
        .sort((a, b) => a.key.localeCompare(b.key))

      finalBuildArgs = serializeBuildArgPairs(normalizedPairs)
      finalPairs = normalizedPairs
    } else {
      finalBuildArgs = buildArgsRaw
    }

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
      build_args: finalBuildArgs,
      build_arg_pairs: finalPairs,
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-180 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{mode === "create" ? "Add Build Setting" : "Edit Build Setting"}</DialogTitle>
          <DialogDescription>
            Configure source, image output, build platforms, cache, and build args.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
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
            <div className="flex items-center gap-2">
              <Checkbox
                id="registry_cache_enabled"
                checked={form.registry_cache_enabled}
                onCheckedChange={(checked) => setForm((current) => ({ ...current, registry_cache_enabled: checked }))}
              />
              <label htmlFor="registry_cache_enabled" className="cursor-pointer">
                Enable registry cache
              </label>
            </div>
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


          <Field>
            <div className="flex items-center justify-between">
              <FieldLabel>Build Args</FieldLabel>
              <Button
                variant="ghost"
                size="sm"
                className="h-auto p-0 text-xs"
                onClick={() => setBuildArgsMode(m => m === "structured" ? "advanced" : "structured")}
              >
                {buildArgsMode === "structured" ? "Switch to Advanced" : "Switch to Structured"}
              </Button>
            </div>
            <FieldContent>
              {buildArgsMode === "advanced" ? (
                <Textarea
                  value={buildArgsRaw}
                  onChange={(e) => setBuildArgsRaw(e.target.value)}
                  placeholder="KEY=value"
                  className="font-mono text-sm"
                  rows={5}
                />
              ) : (
                <KeyValueInput
                  value={buildArgPairs}
                  onChange={setBuildArgPairs}
                  keyPlaceholder="ARG_KEY"
                  valuePlaceholder="ARG_VALUE"
                />
              )}
            </FieldContent>
          </Field>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSubmit} disabled={isPending}>
            {mode === "create" ? "Add" : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
