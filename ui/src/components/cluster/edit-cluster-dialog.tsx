import { AxiosError } from "axios"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { InfoIcon, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi, type Cluster } from "@/api/clusters"
import { Button } from "@/components/ui/button"
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
import { Textarea } from "@/components/ui/textarea"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"

interface EditClusterDialogProps {
	open?: boolean
	onOpenChange?: (open: boolean) => void
	cluster: Cluster | null
	onSuccess?: () => void
}

export function EditClusterDialog({
	open: controlledOpen,
	onOpenChange: setControlledOpen,
	cluster,
	onSuccess,
}: EditClusterDialogProps) {
	const [internalOpen, setInternalOpen] = React.useState(false)
	const open = controlledOpen !== undefined ? controlledOpen : internalOpen
	const setOpen = setControlledOpen || setInternalOpen
	const queryClient = useQueryClient()

	const [errors, setErrors] = React.useState<{
		name?: string
		description?: string
	}>({})

	const [formData, setFormData] = React.useState({
		name: "",
		description: "",
	})

	React.useEffect(() => {
		if (cluster && open) {
			setFormData({
				name: cluster.name,
				description: cluster.description || "",
			})
			setErrors({})
		}
	}, [cluster, open])

	const updateBasicMutation = useMutation({
		mutationFn: (data: { name: string; description: string }) => {
			if (!cluster) throw new Error("No cluster selected")
			return clustersApi.update(cluster.id, data)
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["clusters"] })
			toast.success("Cluster updated successfully")
			setOpen(false)
			onSuccess?.()
		},
		onError: (error: AxiosError<{ error: string }>) => {
			toast.error("Error", {
				description: error.response?.data?.error || "Failed to update cluster",
			})
		},
	})

	const validateBasicForm = () => {
		const newErrors: typeof errors = {}

		if (!formData.name.trim()) {
			newErrors.name = "Name is required"
		} else if (formData.name.length < 2) {
			newErrors.name = "Name must be at least 2 characters"
		} else if (formData.name.length > 50) {
			newErrors.name = "Name must be less than 50 characters"
		}

		setErrors(newErrors)
		return Object.keys(newErrors).length === 0
	}

  const handleSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
		e.preventDefault()

		if (!validateBasicForm()) {
			return
		}

		updateBasicMutation.mutate({
			name: formData.name,
			description: formData.description,
		})
	}

	return (
		<Dialog open={open} onOpenChange={setOpen}>
			<DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
				<form onSubmit={handleSubmit}>
					<DialogHeader>
						<DialogTitle>Edit Cluster</DialogTitle>
						<DialogDescription>
							Update cluster information.
						</DialogDescription>
					</DialogHeader>

					<div className="grid gap-4 py-4">
						<Field>
							<FieldLabel>
								Name *
								<Tooltip>
									<TooltipTrigger
										tabIndex={-1}
										render={
											<button type="button" className="text-muted-foreground hover:text-foreground transition-colors outline-none">
												<InfoIcon className="h-3.5 w-3.5" />
											</button>
										}
									/>
									<TooltipContent side="top" align="start" className="max-w-100 flex-col items-start">
										<p className="text-xs">- 2-50 characters.</p>
									</TooltipContent>
								</Tooltip>
							</FieldLabel>
							<FieldContent>
								<Input
									placeholder="My Cluster"
									value={formData.name}
									onChange={(e) => setFormData((prev) => ({ ...prev, name: e.target.value }))}
									aria-invalid={!!errors.name}
								/>
							</FieldContent>
							{errors.name && (
								<FieldError>
									<span className="text-destructive text-xs">{errors.name}</span>
								</FieldError>
							)}
						</Field>

						<Field>
							<FieldLabel>Slug</FieldLabel>
							<FieldContent>
								<Input
									value={cluster?.slug || ""}
									disabled
									className="bg-muted font-mono"
								/>
							</FieldContent>
						</Field>

						<Field>
							<FieldLabel>Description</FieldLabel>
							<FieldContent>
								<Textarea
									placeholder="Brief description of the cluster."
									value={formData.description}
									onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
									className="min-h-20 max-h-48 resize-y break-all whitespace-pre-wrap"
								/>
							</FieldContent>
						</Field>
					</div>

					<DialogFooter className="flex gap-2 sm:justify-end">
						<Button type="button" variant="outline" onClick={() => setOpen(false)}>
							Cancel
						</Button>
						<Button type="submit" disabled={updateBasicMutation.isPending}>
							{updateBasicMutation.isPending ? (
								<>
									<Loader2 className="h-4 w-4 animate-spin mr-2" />
									Saving...
								</>
							) : (
								"Save Changes"
							)}
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	)
}

export default EditClusterDialog
