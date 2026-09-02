import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { getErrorMessage } from "@/lib/utils"

export interface EnvVarSpec {
  id?: string
  key: string
  value: string
  is_secret?: boolean
  has_value?: boolean
}

interface EnvVarEditorProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  app: App
  envVar?: EnvVarSpec | null
  onSuccess?: () => void
}

export function EnvVarEditor({
  open,
  onOpenChange,
  app,
  envVar,
  onSuccess,
}: EnvVarEditorProps) {
  const queryClient = useQueryClient()

  const [formData, setFormData] = React.useState<EnvVarSpec>({
    key: "",
    value: "",
    is_secret: false,
  })

  const [errors, setErrors] = React.useState<Record<string, string>>({})

  // Reset form when dialog opens/closes or envVar changes
  React.useEffect(() => {
    if (open) {
      if (envVar) {
        setFormData({
          id: envVar.id,
          key: envVar.key || "",
          value: envVar.value || "",
          is_secret: envVar.is_secret || false,
        })
      } else {
        setFormData({
          key: "",
          value: "",
          is_secret: false,
        })
      }
      setErrors({})
    }
  }, [open, envVar])

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {}

    if (!formData.key.trim()) {
      newErrors.key = "Key is required"
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const saveMutation = useMutation({
    mutationFn: async (data: EnvVarSpec) => {
      if (data.id) {
        return await appsApi.updateEnvVar(data.id, data)
      } else {
        return await appsApi.addEnvVar(app.id, data)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-env-vars", app.id] })
      toast.success(
        envVar ? "Environment variable updated" : "Environment variable added"
      )
      onSuccess?.()
    },
    onError: (error: unknown) => {
      toast.error("Failed to save environment variable", {
        description: getErrorMessage(error, "Unknown error"),
      })
    },
  })

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!validateForm()) return

    saveMutation.mutate(formData)
  }

  const isEditing = !!envVar

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>
              {envVar ? "Edit Environment Variable" : "Add Environment Variable"}
            </DialogTitle>
            <DialogDescription>
              Configure key-value pairs for your application environment
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <Field>
              <FieldLabel htmlFor="key">Key *</FieldLabel>
              <FieldContent>
                <Input
                  id="key"
                  value={formData.key}
                  onChange={(e) =>
                    setFormData((prev) => ({ ...prev, key: e.target.value }))
                  }
                  placeholder="DATABASE_URL"
                  className="font-mono"
                  required
                  disabled={isEditing}
                />
              </FieldContent>
              {errors.key && <FieldError>{errors.key}</FieldError>}
              {isEditing && (
                <p className="text-xs text-muted-foreground mt-1">
                  Key cannot be changed after creation
                </p>
              )}
            </Field>

            <Field orientation="horizontal">
              <Checkbox
                id="is-secret"
                checked={formData.is_secret}
                onCheckedChange={(checked) =>
                  setFormData((prev) => ({ ...prev, is_secret: checked === true }))
                }
              />
              <FieldContent>
                <FieldLabel htmlFor="is-secret">Sensitive value</FieldLabel>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="value">Value</FieldLabel>
              <FieldContent>
                <Input
                  id="value"
                  value={formData.value}
                  onChange={(e) =>
                    setFormData((prev) => ({ ...prev, value: e.target.value }))
                  }
                  placeholder="postgresql://..."
                  className="font-mono"
                />
              </FieldContent>
            </Field>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={saveMutation.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={saveMutation.isPending}>
              {saveMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin" />
              )}
              {envVar ? "Update" : "Add"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
