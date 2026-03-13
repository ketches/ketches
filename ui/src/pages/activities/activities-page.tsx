import {
  activitiesApi,
  operationLogsApi,
  type OperationLogItem,
  type OperationLogSensitivity,
  type OperationLogStatus,
} from "@/api/operation-logs"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { useDebounce } from "@/hooks/use-debounce"
import { cn, formatDate, toTitleCase } from "@/lib/utils"
import { useAuthStore } from "@/stores/auth"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { endOfDay, format, startOfDay, subDays } from "date-fns"
import {
  Activity,
  CalendarIcon,
  CheckCircle2,
  Download,
  Loader2,
  Save,
  Shield,
  ShieldAlert,
  ShieldCheck,
  User,
  XCircle
} from "lucide-react"
import * as React from "react"
import type { DateRange } from "react-day-picker"
import { toast } from "sonner"


const ACTIVITY_RESOURCE_TYPE_OPTIONS = [
  { label: "All Resources", value: "all" },
  { label: "App", value: "app" },
  { label: "Code Repository", value: "code_repository" },
  { label: "Project", value: "project" },
  { label: "Environment", value: "env" },
  { label: "Deployment", value: "deployment" },
  { label: "Build", value: "build" },
  { label: "User", value: "user" },
  { label: "Session", value: "session" },
] as const

const ACTIVITY_ACTION_OPTIONS = [
  { label: "All Actions", value: "all" },
  { label: "Create", value: "create" },
  { label: "Update", value: "update" },
  { label: "Delete", value: "delete" },
  { label: "Deploy", value: "deploy" },
  { label: "Build", value: "build" },
  { label: "Rollback", value: "rollback" },
  { label: "Start", value: "start" },
  { label: "Stop", value: "stop" },
  { label: "Restart", value: "restart" },
  { label: "Sign In", value: "sign-in" },
  { label: "Sign Up", value: "sign-up" },
] as const


const ACTIVITY_SENSITIVITY_OPTIONS = [
  { label: "All Sensitivities", value: "all" },
  { label: "Public", value: "public" },
  { label: "Internal", value: "internal" },
  { label: "Sensitive", value: "sensitive" },
] as const

const ACTIVITY_STATUS_OPTIONS = [
  { label: "All Statuses", value: "all" },
  { label: "Success", value: "success" },
  { label: "Failure", value: "failure" },
] as const

const OPERATION_LOG_SENSITIVITY_COLOR_BADGE: Record<OperationLogSensitivity, { color: "green" | "yellow" | "red"; icon: React.ComponentType<{ className?: string }> }> = {
  public: {
    color: "green",
    icon: ShieldCheck,
  },
  internal: {
    color: "yellow",
    icon: Shield,
  },
  sensitive: {
    color: "red",
    icon: ShieldAlert,
  },
}

const OPERATION_LOG_STATUS_BADGE: Record<OperationLogStatus, { color: "green" | "red"; icon: React.ComponentType<{ className?: string }> }> = {
  success: {
    color: "green",
    icon: CheckCircle2,
  },
  failure: {
    color: "red",
    icon: XCircle,
  },
}


function StatusCell({ status }: { status: OperationLogStatus }) {
  const { color, icon: Icon } = OPERATION_LOG_STATUS_BADGE[status]
  return <ColorBadge color={color}>
    <Icon className="h-3 w-3" />
    {toTitleCase(status)}
  </ColorBadge>
}

function SensitivityCell({ sensitivity }: { sensitivity: OperationLogSensitivity }) {
  const { color, icon: Icon } = OPERATION_LOG_SENSITIVITY_COLOR_BADGE[sensitivity]
  return <ColorBadge color={color}>
    <Icon className="h-3 w-3" />
    {toTitleCase(sensitivity)}
  </ColorBadge>
}

export function ActivitiesPage() {
  const queryClient = useQueryClient()
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")

  const [search, setSearch] = React.useState("")
  const [action, setAction] = React.useState("")
  const [resourceType, setResourceType] = React.useState("")
  const now = new Date()
  const sevenDaysAgo = subDays(now, 7)
  const [start, setStart] = React.useState(startOfDay(sevenDaysAgo).toISOString())
  const [end, setEnd] = React.useState(endOfDay(now).toISOString())
  const [dateRange, setDateRange] = React.useState<DateRange | undefined>({
    from: sevenDaysAgo,
    to: now,
  })
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

  React.useEffect(() => {
    if (dateRange?.from) {
      setStart(startOfDay(dateRange.from).toISOString())
    } else {
      setStart("")
    }
    if (dateRange?.to) {
      setEnd(endOfDay(dateRange.to).toISOString())
    } else {
      setEnd("")
    }
    setPagination((p) => ({ ...p, pageIndex: 0 }))
  }, [dateRange])

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

  const toTitleCase = (str: string) => {
    return str
      .split(/[\s_-]+/)
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ')
  }

  const columns = React.useMemo<ColumnDef<OperationLogItem>[]>(
    () => [
      ...(isAdmin
        ? [
          {
            accessorKey: "username",
            header: "User",
            cell: ({ row }) => (
              <div className="inline-flex items-center gap-2 text-sm">
                <User className="h-3.5 w-3.5 text-muted-foreground" />
                <span>{row.original.username || "Unknown"}</span>
              </div>
            ),
          } satisfies ColumnDef<OperationLogItem>,
        ]
        : []),
      {
        accessorKey: "action",
        header: "Action",
        cell: ({ row }) => <span className="">{toTitleCase(row.original.action)}</span>,
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
        accessorKey: "sensitivity",
        header: "Sensitivity",
        cell: ({ row }) => <SensitivityCell sensitivity={row.original.sensitivity} />,
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => <StatusCell status={row.original.status} />,
      },
      {
        accessorKey: "client_ip",
        header: "IP",
        cell: ({ row }) => <span className="font-mono text-xs">{row.original.client_ip || "-"}</span>,
      },
      {
        accessorKey: "created_at",
        header: "Activity Time",
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">{formatDate(row.original.created_at)}</span>
        ),
      },
    ],
    [isAdmin]
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

  const leftToolbar = (
    <>
      <Input
        className="w-full sm:w-80"
        placeholder={isAdmin ? "Filter by user, action, resource..." : "Filter by action, resource..."}
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
        itemToStringLabel={(v) => ACTIVITY_ACTION_OPTIONS.find(o => o.value === v)?.label || v}
      >
        <ComboboxInput className="w-48" placeholder="Action" readOnly />
        <ComboboxContent>
          <ComboboxList>
            {ACTIVITY_ACTION_OPTIONS.map((item) => (
              <ComboboxItem key={item.value} value={item.value}>{item.label}</ComboboxItem>
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
        itemToStringLabel={(v) => ACTIVITY_RESOURCE_TYPE_OPTIONS.find(o => o.value === v)?.label || v}
      >
        <ComboboxInput className="w-52" placeholder="Resource Type" readOnly />
        <ComboboxContent>
          <ComboboxList>
            {ACTIVITY_RESOURCE_TYPE_OPTIONS.map((item) => (
              <ComboboxItem key={item.value} value={item.value}>{item.label}</ComboboxItem>
            ))}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>

      <Popover>
        <PopoverTrigger
          render={
            <Button
              variant="outline"
              className={cn(
                "w-65 justify-start text-left font-normal",
                !dateRange && "text-muted-foreground"
              )}
            />
          }
        >
          <CalendarIcon className="mr-1 h-4 w-4" />
          {dateRange?.from ? (
            dateRange.to ? (
              <>
                {format(dateRange.from, "LLL dd, y")} -{" "}
                {format(dateRange.to, "LLL dd, y")}
              </>
            ) : (
              format(dateRange.from, "LLL dd, y")
            )
          ) : (
            <span>Pick a date</span>
          )}
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="start">
          <Calendar
            autoFocus
            mode="range"
            defaultMonth={dateRange?.from}
            selected={dateRange}
            onSelect={setDateRange}
            numberOfMonths={2}
          />
        </PopoverContent>
      </Popover>

      <Combobox
        value={sensitivity}
        onValueChange={(value) => {
          setSensitivity((value as "all" | OperationLogSensitivity) || "all")
          setPagination((p) => ({ ...p, pageIndex: 0 }))
        }}
        itemToStringLabel={(v) => ACTIVITY_SENSITIVITY_OPTIONS.find(o => o.value === v)?.label || v}
      >
        <ComboboxInput className="w-40" placeholder="Sensitivity" readOnly />
        <ComboboxContent>
          <ComboboxList>
            {ACTIVITY_SENSITIVITY_OPTIONS.map((item) => (
              <ComboboxItem key={item.value} value={item.value}>{item.label}</ComboboxItem>
            ))}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>

      <Combobox
        value={status}
        onValueChange={(value) => {
          setStatus((value as "all" | OperationLogStatus) || "all")
          setPagination((p) => ({ ...p, pageIndex: 0 }))
        }}
        itemToStringLabel={(v) => ACTIVITY_STATUS_OPTIONS.find(o => o.value === v)?.label || v}
      >
        <ComboboxInput className="w-36" placeholder="Status" readOnly />
        <ComboboxContent>
          <ComboboxList>
            {ACTIVITY_STATUS_OPTIONS.map((item) => (
              <ComboboxItem key={item.value} value={item.value}>{item.label}</ComboboxItem>
            ))}
          </ComboboxList>
        </ComboboxContent>
      </Combobox></>
  )
  const rightToolbar = isAdmin ? (
    <>
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
    </>
  ) : (<></>)

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
          leftToolbar={() => leftToolbar}
          rightToolbar={() => rightToolbar}
          pagination={pagination}
          onPaginationChange={setPagination}
          totalCount={response?.pagination.total ?? 0}
          manualPagination
        />
      )}
    </div>
  )
}
