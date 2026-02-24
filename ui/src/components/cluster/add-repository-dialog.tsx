import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi, type CreateHelmRepositoryRequest } from "@/api/clusters"
import { Button } from "@/components/ui/button"
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface AddRepositoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  clusterId: string
}

export function AddRepositoryDialog({
  open,
  onOpenChange,
  clusterId,
}: AddRepositoryDialogProps) {
  const queryClient = useQueryClient()

  const [form, setForm] = React.useState<CreateHelmRepositoryRequest>({
    name: "",
    url: "",
    type: "helm",
    username: "",
    password: "",
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateHelmRepositoryRequest) =>
      clustersApi.createHelmRepository(clusterId, data),
    onSuccess: () => {
      toast.success("Repository added successfully")
      queryClient.invalidateQueries({
        queryKey: ["helm-repositories", clusterId],
      })
      onOpenChange(false)
      resetForm()
    },
    onError: (error: any) => {
      toast.error("Failed to add repository", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const resetForm = () => {
    setForm({
      name: "",
      url: "",
      type: "helm",
      username: "",
      password: "",
    })
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const data: CreateHelmRepositoryRequest = {
      name: form.name,
      url: form.url,
      type: form.type,
    }
    if (form.username) data.username = form.username
    if (form.password) data.password = form.password
    createMutation.mutate(data)
  }

  // Set URL prefix hint based on type
  const urlPlaceholder =
    form.type === "oci"
      ? "oci://registry.example.com/charts"
      : "https://charts.example.com"

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add Helm Repository</DialogTitle>
            <DialogDescription>
              Add a Helm chart repository or OCI registry to install extensions
              from.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel htmlFor="repo-name">Name</FieldLabel>
              <FieldContent>
                <Input
                  id="repo-name"
                  placeholder="my-charts"
                  value={form.name}
                  onChange={(e) =>
                    setForm({ ...form, name: e.target.value })
                  }
                  required
                />
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="repo-type">Type</FieldLabel>
              <FieldContent>
                <Select
                  value={form.type}
                  onValueChange={(value: "helm" | "oci") =>
                    setForm({ ...form, type: value, url: "" })
                  }
                >
                  <SelectTrigger id="repo-type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="helm">Helm Repository</SelectItem>
                    <SelectItem value="oci">OCI Registry</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-[11px] text-muted-foreground mt-1">
                  {form.type === "oci"
                    ? "OCI registries store Helm charts as OCI artifacts (e.g., Docker Hub, GitHub Container Registry)."
                    : "Standard Helm chart repository serving an index.yaml file."}
                </p>
              </FieldContent>
            </Field>

            <Field>
              <FieldLabel htmlFor="repo-url">URL</FieldLabel>
              <FieldContent>
                <Input
                  id="repo-url"
                  placeholder={urlPlaceholder}
                  value={form.url}
                  onChange={(e) =>
                    setForm({ ...form, url: e.target.value })
                  }
                  required
                />
              </FieldContent>
            </Field>

            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel htmlFor="repo-username">
                  Username{" "}
                  <span className="text-muted-foreground font-normal">
                    (optional)
                  </span>
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="repo-username"
                    placeholder="Username"
                    value={form.username}
                    onChange={(e) =>
                      setForm({ ...form, username: e.target.value })
                    }
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="repo-password">
                  Password{" "}
                  <span className="text-muted-foreground font-normal">
                    (optional)
                  </span>
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="repo-password"
                    type="password"
                    placeholder="Password or token"
                    value={form.password}
                    onChange={(e) =>
                      setForm({ ...form, password: e.target.value })
                    }
                  />
                </FieldContent>
              </Field>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createMutation.isPending}>
              {createMutation.isPending ? (
                <>
                  <Loader2 className="animate-spin" />
                  Adding...
                </>
              ) : (
                "Add Repository"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
