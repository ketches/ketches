import {
  activitiesApi,
  operationLogsApi,
  type OperationLogItem,
  type OperationLogSensitivity,
  type OperationLogStatus,
} from "@/api/operation-logs"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Input } from "@/components/ui/input"
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate } from "@/lib/utils"
import { useAuthStore } from "@/stores/auth"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { format, subDays } from "date-fns"
import {
  Activity,
  CheckCircle2,
  Download,
  Loader2,
  Save,
  Shield,
  ShieldAlert,
  ShieldCheck,
  User,
  XCircle,
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

const ACTIVITY_ACTION_OPTIONS = [
  "create",
  "update",
  "delete",
  "deploy",
  "build",
  "rollback",
  "start",
  "stop",
  "restart",
  "sign-in",
  "sign-up",
] as const

const ACTIVITY_RESOURCE_TYPE_OPTIONS = [
  "app",
  "code_repository",
  "project",
  "env",
  "deployment",
  "build",
  "user",
  "session",
] as const

function StatusCell({ status }: { status: OperationLogStatus }) {
  if (status === "success") {
    return (
      <div className="inline-flex items-center gap-1 text-xs text-green-600">
        <CheckCircle2 className="h-3.5 w-3.5" />
        <span>Success</span>
      </div>
    )
  }

  return (
    <div className="inline-flex items-center gap-1 text-xs text-red-600">
      <XCircle className="h-3.5 w-3.5" />
      <span>Failure</span>
    </div>
  )
}

function SensitivityCell({ sensitivity }: { sensitivity: OperationLogSensitivity }) {
  if (sensitivity === "sensitive") {
    return (
      <div className="inline-flex items-center gap-1 text-xs text-red-600">
        <ShieldAlert className="h-3.5 w-3.5" />
        <span>Sensitive</span>
      </div>
    )
  }

  if (sensitivity === "internal") {
    return (
      <div className="inline-flex items-center gap-1 text-xs text-amber-600">
        <Shield className="h-3.5 w-3.5" />
        <span>Internal</span>
      </div>
    )
  }

  return (
    <div className="inline-flex items-center gap-1 text-xs text-sky-600">
      <ShieldCheck className="h-3.5 w-3.5" />
      <span>Public</span>
    </div>
  )
}

export function ActivitiesPage() {
  const queryClient = useQueryClient()
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")

  const [search, setSearch] = React.useState("")
  const [action, setAction] = React.useState("")
  const [resourceType, setResourceType] = React.useState("")
  const now = new Date()
  const sevenDaysAgo = subDays(now, 7)
  const [start, setStart] = React.useState(format(sevenDaysAgo, "yyyy-MM-dd'T'HH:mm"))
  const [end, setEnd] = React.useState(format(now, "yyyy-MM-dd'T'HH:mm"))
  const [sensitivity, setSensitivity] = React.useState<"all" | OperationLogSensitivity>("all")
  const [status, setStatus] = React.useState<"all" | OperationLogStatus>("all")
  const [pagination, setPagination] = React.useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [retentionDays, setRetentionDays] = React.useState("90")

  const debouncedSearch = useDebounce(search, 300)

  const { data: settings, isLoading: settingsLoading } = useQuery({
    queryKey: ["operation-log-settings"],
    queryFn: () => operationLogsApi.getOperationLogSettings(),
    enabled: isAdmin,
  })

  React.useEffect(() => {
    if (settings?.retention_days) {
      setRetentionDays(String(settings.retention_days))
    }
  }, [settings])

  const { data: response, isLoading, isFetching } = useQuery({
    queryKey: [
      "activities",
      isAdmin,
      debouncedSearch,
      action,
      resourceType,
      start,
      end,
      sensitivity,
      status,
      pagination.pageIndex,
      pagination.pageSize,
    ],
    queryFn: () => {
      const params = {
        search: debouncedSearch || undefined,
        action: action || undefined,
        resource_type: resourceType || undefined,
        start: start || undefined,
        end: end || undefined,
        sensitivity: sensitivity === "all" ? undefined : sensitivity,
        status: status === "all" ? undefined : status,
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
      }
      return isAdmin
        ? operationLogsApi.listOperationLogs(params)
        : activitiesApi.list(params)
    },
  })

  const items = React.useMemo(() => response?.items ?? [], [response])

  const updateSettingsMutation = useMutation({
    mutationFn: (days: number) => operationLogsApi.updateOperationLogSettings(days),
    onSuccess: () => {
      toast.success("Retention settings updated")
      queryClient.invalidateQueries({ queryKey: ["operation-log-settings"] })
    },
    onError: (error: Error) => {
      toast.error("Failed to update retention settings", {
        description: error.message,
      })
    },
  })

  const columns = React.useMemo<ColumnDef<OperationLogItem>[]>(
    () => [
      {
        accessorKey: "created_at",
        header: "Time",
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">{formatDate(row.original.created_at)}</span>
        ),
      },
      {
        accessorKey: "username",
        header: "User",
        cell: ({ row }) => (
          <div className="inline-flex items-center gap-2 text-sm">
            <User className="h-3.5 w-3.5 text-muted-foreground" />
            <span>{row.original.username || "Unknown"}</span>
          </div>
        ),
      },
      {
        accessorKey: "action",
        header: "Action",
        cell: ({ row }) => <span className="text-sm font-medium">{row.original.action}</span>,
      },
      {
        accessorKey: "resource_type",
        header: "Resource",
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {row.original.resource_type} / {row.original.resource_id || "-"}
          </span>
        ),
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => <StatusCell status={row.original.status} />,
      },
      {
        accessorKey: "sensitivity",
        header: "Sensitivity",
        cell: ({ row }) => <SensitivityCell sensitivity={row.original.sensitivity} />,
      },
      {
        accessorKey: "client_ip",
        header: "IP",
        cell: ({ row }) => <span className="font-mono text-xs">{row.original.client_ip || "-"}</span>,
      },
    ],
    []
  )

  const handleExport = async () => {
    try {
      await operationLogsApi.exportOperationLogsCSV({
        search: debouncedSearch || undefined,
        action: action || undefined,
        resource_type: resourceType || undefined,
        start: start || undefined,
        end: end || undefined,
        sensitivity: sensitivity === "all" ? undefined : sensitivity,
        status: status === "all" ? undefined : status,
        page: pagination.pageIndex + 1,
        page_size: pagination.pageSize,
      })
      toast.success("CSV export started")
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unknown error"
      toast.error("Failed to export logs", { description: message })
    }
  }

  const saveRetention = () => {
    const days = Number(retentionDays)
    if (!Number.isInteger(days) || days < 1) {
      toast.error("Retention days must be a positive integer")
      return
    }
    updateSettingsMutation.mutate(days)
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader items={[{ label: "Activities", icon: Activity }]} />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Activities</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {isAdmin
              ? "Review global operation logs and manage retention policy."
              : "Track your recent actions and project events."}
          </p>
        </div>
      </div>

      {isAdmin && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Operation Log Settings</CardTitle>
          </CardHeader>
          <CardContent className="flex items-center gap-2">
            <Input
              type="number"
              min={1}
              value={retentionDays}
              onChange={(e) => setRetentionDays(e.target.value)}
              className="w-44"
              disabled={settingsLoading || updateSettingsMutation.isPending}
            />
            <Button onClick={saveRetention} disabled={settingsLoading || updateSettingsMutation.isPending}>
              {updateSettingsMutation.isPending ? <Loader2 className="animate-spin" /> : <Save />}
              Save Retention Days
            </Button>
            <Button variant="secondary" onClick={handleExport}>
              <Download />
              Export CSV
            </Button>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="space-y-3">
          <CardTitle className="text-sm">Filters</CardTitle>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              className="w-full sm:w-80"
              placeholder="Search by action/resource/user"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value)
                setPagination((p) => ({ ...p, pageIndex: 0 }))
              }}
            />

            <Combobox
              value={action || "all"}
              onValueChange={(value) => {
                setAction(!value || value === "all" ? "" : value)
                setPagination((p) => ({ ...p, pageIndex: 0 }))
              }}
            >
              <ComboboxInput className="w-48" placeholder="Action" readOnly />
              <ComboboxContent>
                <ComboboxList>
                  <ComboboxItem value="all">All actions</ComboboxItem>
                  {ACTIVITY_ACTION_OPTIONS.map((item) => (
                    <ComboboxItem key={item} value={item}>{item}</ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>

            <Combobox
              value={resourceType || "all"}
              onValueChange={(value) => {
                setResourceType(!value || value === "all" ? "" : value)
                setPagination((p) => ({ ...p, pageIndex: 0 }))
              }}
            >
              <ComboboxInput className="w-52" placeholder="Resource Type" readOnly />
              <ComboboxContent>
                <ComboboxList>
                  <ComboboxItem value="all">All resource types</ComboboxItem>
                  {ACTIVITY_RESOURCE_TYPE_OPTIONS.map((item) => (
                    <ComboboxItem key={item} value={item}>{item}</ComboboxItem>
                  ))}
                </ComboboxList>
              </ComboboxContent>
            </Combobox>

            <Input
              className="w-full sm:w-52"
              type="datetime-local"
              value={start}
              onChange={(e) => {
                setStart(e.target.value)
                setPagination((p) => ({ ...p, pageIndex: 0 }))
              }}
            />

            <Input
              className="w-full sm:w-52"
              type="datetime-local"
              value={end}
              onChange={(e) => {
                setEnd(e.target.value)
                setPagination((p) => ({ ...p, pageIndex: 0 }))
              }}
            />

            <Combobox
              value={sensitivity}
              onValueChange={(value) => {
                setSensitivity((value as "all" | OperationLogSensitivity) || "all")
                setPagination((p) => ({ ...p, pageIndex: 0 }))
              }}
            >
              <ComboboxInput className="w-40" placeholder="Sensitivity" readOnly />
              <ComboboxContent>
                <ComboboxList>
                  <ComboboxItem value="all">All sensitivities</ComboboxItem>
                  <ComboboxItem value="public">Public</ComboboxItem>
                  <ComboboxItem value="internal">Internal</ComboboxItem>
                  <ComboboxItem value="sensitive">Sensitive</ComboboxItem>
                </ComboboxList>
              </ComboboxContent>
            </Combobox>

            <Combobox
              value={status}
              onValueChange={(value) => {
                setStatus((value as "all" | OperationLogStatus) || "all")
                setPagination((p) => ({ ...p, pageIndex: 0 }))
              }}
            >
              <ComboboxInput className="w-36" placeholder="Status" readOnly />
              <ComboboxContent>
                <ComboboxList>
                  <ComboboxItem value="all">All statuses</ComboboxItem>
                  <ComboboxItem value="success">Success</ComboboxItem>
                  <ComboboxItem value="failure">Failure</ComboboxItem>
                </ComboboxList>
              </ComboboxContent>
            </Combobox>
          </div>
        </CardHeader>

        <CardContent>
          {items.length === 0 && !isLoading && !isFetching ? (
            <EmptyState
              title="No activities found"
              description="Your activities and operation logs will appear here when actions occur."
              icon={Activity}
            />
          ) : (
            <DataTable
              columns={columns}
              data={items}
              isLoading={isLoading || isFetching}
              pagination={pagination}
              onPaginationChange={setPagination}
              totalCount={response?.pagination.total ?? 0}
              manualPagination
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
