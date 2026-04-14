import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Plus, Save, Scale, Trash2 } from "lucide-react"
import { Controller, useFieldArray, useForm, type Resolver } from "react-hook-form"
import { toast } from "sonner"
import * as z from "zod"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { KeyValueInput } from "@/components/shared/key-value-input"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Field, FieldContent, FieldDescription, FieldError, FieldLabel, FieldTitle } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { useProjectRole } from "@/hooks/useProjectRole"
import type { AxiosError } from "axios"

const PLACEMENT_RULE_OPTIONS = [
  { value: "nodeName", label: "Specific Node Name", description: "Strictly schedule on this exact node" },
  { value: "nodeSelector", label: "Node Selector (Labels)", description: "Schedule on nodes matching label key-value pairs" },
  { value: "nodeAffinity", label: "Node Affinity (Advanced)", description: "Advanced affinity rules in Kubernetes format" },
]

const schedulingSchema = z.object({
  rule_type: z.string(),
  node_name: z.string().optional(),
  node_selectors: z.array(z.object({
    key: z.string().min(1),
    value: z.string()
  })),
  node_affinity: z.string().optional(),
  tolerations: z.array(z.object({
    key: z.string().optional(),
    operator: z.string(),
    value: z.string().optional(),
    effect: z.string(),
    toleration_seconds: z.coerce.number().optional()
  })),
})

type SchedulingFormValues = {
  rule_type: string
  node_name?: string
  node_selectors: { key: string; value: string }[]
  node_affinity?: string
  tolerations: {
    key?: string
    operator: string
    value?: string
    effect: string
    toleration_seconds?: number
  }[]
}

interface SchedulingConfigProps {
  app: App
}

export function SchedulingConfig({ app }: SchedulingConfigProps) {
  const queryClient = useQueryClient()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'


  let initialNodeSelectors: any[] = []
  try {
    const parsed = JSON.parse(app.scheduling_rule?.node_selector || "{}")
    initialNodeSelectors = Object.entries(parsed).map(([key, value]) => ({ key, value }))
  } catch { }

  let initialTolerations: any[] = []
  try {
    initialTolerations = JSON.parse(app.scheduling_rule?.tolerations || "[]")
  } catch { }

  const { control, register, handleSubmit, watch, formState: { errors } } = useForm<SchedulingFormValues>({
    resolver: zodResolver(schedulingSchema) as Resolver<SchedulingFormValues>,
    defaultValues: {
      rule_type: app.scheduling_rule?.rule_type || "nodeSelector",
      node_name: app.scheduling_rule?.node_name || "",
      node_selectors: initialNodeSelectors,
      node_affinity: app.scheduling_rule?.node_affinity || "",
      tolerations: initialTolerations.map((t: any) => ({
        key: t.key || "",
        operator: t.operator || "Equal",
        value: t.value || "",
        effect: t.effect || "NoSchedule",
        toleration_seconds: t.tolerationSeconds || undefined
      })),
    },
  })

  const { fields: tolerationFields, append: appendToleration, remove: removeToleration } = useFieldArray({
    control,
    name: "tolerations",
  })

  const ruleType = watch("rule_type")

  const updateMutation = useMutation({
    mutationFn: (values: z.infer<typeof schedulingSchema>) => {
      const nodeSelectorObj = values.node_selectors.reduce((acc: any, curr) => {
        acc[curr.key] = curr.value
        return acc
      }, {})

      return appsApi.updateScheduling(app.id, {
        rule_type: values.rule_type,
        node_name: values.node_name,
        node_selector: JSON.stringify(nodeSelectorObj),
        node_affinity: values.node_affinity,
        tolerations: JSON.stringify(values.tolerations.map(t => ({
          key: t.key,
          operator: t.operator,
          value: t.value,
          effect: t.effect,
          tolerationSeconds: t.toleration_seconds
        }))),
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app', app.id] })
      toast.success("Scheduling configuration updated")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to update scheduling", {
        description: err.response?.data?.error || "Unknown error"
      })
    }
  })

  return (
    <Card>
      <form onSubmit={handleSubmit((v) => updateMutation.mutate(v))}>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Scale className="h-4 w-4" /> Scheduling Rules
          </CardTitle>
          <CardDescription>Control where your application pods are placed on the cluster nodes</CardDescription>
          <CardAction>
            {!isViewer && (
              <Button type="submit" disabled={updateMutation.isPending}>
                <Save />
                {updateMutation.isPending ? "Saving..." : "Save"}
              </Button>
            )}
          </CardAction>
        </CardHeader>
        <CardContent>
          <div className="space-y-4 mt-4">
            <div className="grid grid-cols-2 items-start gap-2">
              <Field>
                <FieldLabel className="text-muted-foreground">Placement Rule</FieldLabel>
                <FieldContent>
                  <Controller
                    control={control}
                    name="rule_type"
                    render={({ field }) => (
                      <Combobox
                        value={field.value}
                        onValueChange={(v: string | null) => v && field.onChange(v)}
                        itemToStringLabel={(v) => PLACEMENT_RULE_OPTIONS.find((o) => o.value === v)?.label ?? v ?? ""}
                      >
                        <ComboboxInput />
                        <ComboboxContent>
                          <ComboboxList>
                            {[
                              { value: "nodeName", label: "Specific Node Name", description: "Strictly schedule on this exact node" },
                              { value: "nodeSelector", label: "Node Selector (Labels)", description: "Schedule on nodes matching label key-value pairs" },
                              { value: "nodeAffinity", label: "Node Affinity (Advanced)", description: "Advanced affinity rules in Kubernetes format" },
                            ].map((option) => (
                              <ComboboxItem key={option.value} value={option.value}>
                                <div className="flex flex-col gap-0.5">
                                  <span>{option.label}</span>
                                  <span className="text-muted-foreground text-[10px] leading-relaxed">{option.description}</span>
                                </div>
                              </ComboboxItem>
                            ))}
                          </ComboboxList>
                        </ComboboxContent>
                      </Combobox>
                    )}
                  />
                </FieldContent>
                {ruleType === "nodeName" && <FieldDescription><i>Strictly schedule on this specific node.</i></FieldDescription>}
                {ruleType === "nodeSelector" && <FieldDescription><i>Schedule on nodes matching these label key-value pairs.</i></FieldDescription>}
                {ruleType === "nodeAffinity" && <FieldDescription><i>Advanced affinity rules in Kubernetes format.</i></FieldDescription>}
                {errors.rule_type && <FieldError>{errors.rule_type.message}</FieldError>}
              </Field>


              {ruleType === "nodeName" && (
                <Field>
                  <FieldLabel className="text-muted-foreground">Node Name</FieldLabel>
                  <FieldContent>
                    <Input placeholder="e.g. gke-cluster-node-1234" {...register("node_name")} />
                  </FieldContent>
                  {errors.node_name && <FieldError>{errors.node_name.message}</FieldError>}
                </Field>
              )}

              {ruleType === "nodeSelector" && (
                <div className="grid grid-cols-1 gap-2">
                  <FieldLabel className="text-muted-foreground">Node Selector Labels</FieldLabel>
                  <Controller
                    control={control}
                    name="node_selectors"
                    render={({ field }) => (
                      <KeyValueInput
                        value={field.value}
                        onChange={field.onChange}
                        keyPlaceholder="label-key"
                        valuePlaceholder="label-value"
                      />
                    )}
                  />
                </div>
              )}

              {ruleType === "nodeAffinity" && (
                <Field>
                  <FieldLabel className="text-muted-foreground">Node Affinity (JSON)</FieldLabel>
                  <FieldContent>
                    <Textarea
                      placeholder='{ "requiredDuringSchedulingIgnoredDuringExecution": { ... } }'
                      className="text-xs font-mono min-h-32 max-h-48 resize-y break-all whitespace-pre-wrap"
                      {...register("node_affinity")}
                    />
                  </FieldContent>
                  {errors.node_affinity && <FieldError>{errors.node_affinity.message}</FieldError>}
                </Field>
              )}

            </div>

            <div className="pt-4 border-t space-y-4">
              <div className="flex items-center justify-between">
                <FieldTitle className="text-base">Tolerations</FieldTitle>
                <Button type="button" variant="outline" onClick={() => appendToleration({ key: "", operator: "Equal", value: "", effect: "NoSchedule" })}>
                  <Plus /> Add Toleration
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">Allow pods to be scheduled on nodes with matching taints</p>

              <div className="space-y-4">
                {tolerationFields.map((field, index) => (
                  <div key={field.id} className="p-4 border rounded-lg grid grid-cols-2 md:grid-cols-5 gap-2 relative bg-muted/20">
                    <Field>
                      <FieldLabel className="text-[10px] uppercase text-muted-foreground">Key</FieldLabel>
                      <FieldContent><Input {...register(`tolerations.${index}.key`)} /></FieldContent>
                    </Field>
                    <Field>
                      <FieldLabel className="text-[10px] uppercase text-muted-foreground">Operator</FieldLabel>
                      <FieldContent>
                        <Controller
                          control={control}
                          name={`tolerations.${index}.operator`}
                          render={({ field }) => (
                            <Combobox
                              value={field.value}
                              onValueChange={(v: string | null) => v && field.onChange(v)}
                            >
                              <ComboboxInput />
                              <ComboboxContent>
                                <ComboboxList>
                                  {[
                                    { value: "Equal", label: "Equal", description: "Key/value must match exactly" },
                                    { value: "Exists", label: "Exists", description: "Key must exist (value ignored)" },
                                  ].map((option) => (
                                    <ComboboxItem key={option.value} value={option.value}>
                                      <div className="flex flex-col gap-0.5">
                                        <span>{option.label}</span>
                                        <span className="text-muted-foreground text-[10px] leading-relaxed">{option.description}</span>
                                      </div>
                                    </ComboboxItem>
                                  ))}
                                </ComboboxList>
                              </ComboboxContent>
                            </Combobox>
                          )}
                        />
                      </FieldContent>
                    </Field>
                    <Field>
                      <FieldLabel className="text-[10px] uppercase text-muted-foreground">Value</FieldLabel>
                      <FieldContent><Input {...register(`tolerations.${index}.value`)} disabled={watch(`tolerations.${index}.operator`) === "Exists"} /></FieldContent>
                    </Field>
                    <Field>
                      <FieldLabel className="text-[10px] uppercase text-muted-foreground">Effect</FieldLabel>
                      <FieldContent>
                        <Controller
                          control={control}
                          name={`tolerations.${index}.effect`}
                          render={({ field }) => (
                            <Combobox
                              value={field.value}
                              onValueChange={(v: string | null) => v && field.onChange(v)}
                            >
                              <ComboboxInput />
                              <ComboboxContent>
                                <ComboboxList>
                                  {[
                                    { value: "NoSchedule", label: "NoSchedule", description: "Do not schedule unless tolerated" },
                                    { value: "PreferNoSchedule", label: "PreferNoSchedule", description: "Prefer not to schedule unless tolerated" },
                                    { value: "NoExecute", label: "NoExecute", description: "Evict existing pods unless tolerated" },
                                    { value: "", label: "Any", description: "Matches all effects" },
                                  ].map((option) => (
                                    <ComboboxItem key={option.value} value={option.value}>
                                      <div className="flex flex-col gap-0.5">
                                        <span>{option.label}</span>
                                        <span className="text-muted-foreground text-[10px] leading-relaxed">{option.description}</span>
                                      </div>
                                    </ComboboxItem>
                                  ))}
                                </ComboboxList>
                              </ComboboxContent>
                            </Combobox>
                          )}
                        />
                      </FieldContent>
                    </Field>
                    <div className="flex items-end gap-2">
                      <Field>
                        <FieldLabel className="text-[10px] uppercase text-muted-foreground">Sec.</FieldLabel>
                        <FieldContent><Input type="number" {...register(`tolerations.${index}.toleration_seconds`)} /></FieldContent>
                      </Field>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10" onClick={() => removeToleration(index)}>
                        <Trash2 />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </CardContent>
      </form>
    </Card>
  )
}

