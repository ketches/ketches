<script setup lang="ts">
import { deleteApp, exportApps } from "@/api/app";
import { listApps } from "@/api/env";
import ConfirmDialog from "@/components/shared/ConfirmDialog.vue";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
    DropdownMenu,
    DropdownMenuCheckboxItem,
    DropdownMenuContent,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
    Pagination,
    PaginationContent,
    PaginationEllipsis,
    PaginationFirst,
    PaginationItem,
    PaginationLast,
    PaginationNext,
    PaginationPrevious,
} from "@/components/ui/pagination";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { useUserStore } from "@/stores/userStore";
import type { appModel } from "@/types/app";
import type { QueryAndPagedRequest } from "@/types/common";
import { valueUpdater } from "@/utils/valueUpdater";
import type {
    ColumnDef,
    ColumnFiltersState,
    ExpandedState,
    SortingState,
    VisibilityState,
} from "@tanstack/vue-table";
import {
    FlexRender,
    getCoreRowModel,
    getExpandedRowModel,
    getFilteredRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    useVueTable,
} from "@tanstack/vue-table";
import {
    ChevronDown,
    ChevronLeft,
    ChevronRight,
    ChevronsLeft,
    ChevronsRight,
    CloudDownload,
    CloudUpload,
    MoreVertical,
    Plus,
    RefreshCcw,
    Trash
} from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { h, onMounted, ref, watch } from "vue";
import { RouterLink } from "vue-router";
import { toast } from "vue-sonner";
import Badge from "../ui/badge/Badge.vue";
import DropdownMenuItem from "../ui/dropdown-menu/DropdownMenuItem.vue";
import DropdownMenuSeparator from "../ui/dropdown-menu/DropdownMenuSeparator.vue";
import AppActions from "./AppActions.vue";
import CreateApp from "./CreateApp.vue";
import { appStatusDisplay } from "./data/appStatus";

const userStore = useUserStore();
const { activeEnvRef } = storeToRefs(userStore);

const noData = ref(false);
const hasData = ref(false);
const pagedData = ref<appModel[]>([]);
const totalCount = ref(0);

const queryModel = ref<QueryAndPagedRequest>({
    pageNo: 1,
    pageSize: 10,
    query: "",
    sortBy: "created_at",
    sortOrder: "DESC",
});

async function fetchPagedData(envID?: string) {
    if (envID) {
        const { total, records } = (await listApps(envID, queryModel.value)) || {
            total: 0,
            records: [],
        };
        pagedData.value = records;
        totalCount.value = total;

        if (!queryModel.value.query) {
            noData.value = total === 0;
            hasData.value = total > 0;
        }
    }
}

const query = ref("");
function onQueryEnter() {
    if (queryModel.value.query !== query.value) {
        queryModel.value = {
            ...queryModel.value,
            query: query.value,
            pageNo: 1,
        };
    }
    fetchPagedData(activeEnvRef.value?.envID);
}

watch(queryModel, async () => {
    await fetchPagedData(activeEnvRef.value?.envID)
}, { deep: true });

onMounted(async () => {
    await fetchPagedData(activeEnvRef.value?.envID);
});

watch(activeEnvRef, async (newActiveEnvRef) => {
    await fetchPagedData(newActiveEnvRef?.envID);
});

const centeredHeader = (text: string) =>
    h("div", { class: "text-center" }, text);

const columns: ColumnDef<appModel>[] = [
    {
        id: "select",
        header: ({ table }) =>
            h(Checkbox, {
                class: [
                    table.getIsAllPageRowsSelected() || table.getIsSomePageRowsSelected()
                        ? ""
                        : "invisible group-hover:visible",
                ],
                modelValue:
                    table.getIsAllPageRowsSelected()
                        ? true
                        : table.getIsSomePageRowsSelected()
                            ? "indeterminate"
                            : false,
                "onUpdate:modelValue": (value) =>
                    table.toggleAllPageRowsSelected(!!value),
                ariaLabel: "Select all",
            }),
        cell: ({ row }) =>
            h(Checkbox, {
                class: [row.getIsSelected() ? "" : "invisible group-hover:visible"],
                modelValue: row.getIsSelected(),
                "onUpdate:modelValue": (value) => row.toggleSelected(!!value),
                ariaLabel: "Select row",
            }),
        enableSorting: false,
        enableHiding: false,
    },
    {
        accessorKey: 'slug',
        header: "应用",
        cell: ({ row }) =>
            h("div", { class: "space-y-1" }, [
                h(
                    RouterLink,
                    {
                        to: `/console/app/${row.original.appID}`,
                        class: "font-medium text-blue-500",
                    },
                    () => row.original.displayName || row.getValue("slug")
                ),
                h(
                    "div",
                    { class: "text-sm text-muted-foreground font-mono" },
                    row.getValue("slug")
                ),
            ]),
    },
    {
        accessorKey: "status",
        header: () => centeredHeader("状态"),
        cell: ({ row }) => {
            const statusDisplay = appStatusDisplay(row.getValue("status"));
            return h("div", { class: `flex items-center justify-center` }, [
                h(
                    Badge,
                    {
                        variant: "secondary",
                        class: `${statusDisplay.fgColor}`,
                    },
                    () => [
                        h(statusDisplay.icon, {
                            class: "h-4 w-4 mr-1",
                        }),
                        h("span", {}, statusDisplay.label),
                    ]
                ),
            ]);
        },
    },
    {
        accessorKey: "workloadType",
        header: () => centeredHeader("工作负载类型"),
        cell: ({ row }) =>
            h(
                "div",
                { class: "capitalize text-center" },
                row.getValue("workloadType")
            ),
    },
    {
        accessorKey: "containerImage",
        header: "容器镜像",
        cell: ({ row }) => h("div", { class: "" }, row.getValue("containerImage")),
    },
    {
        accessorKey: "replicas",
        header: () => centeredHeader("实例数"),
        cell: ({ row }) => {
            const actualReplicas = row.original.actualReplicas || 0;
            const replicas = row.getValue("replicas") || 0;
            return h(
                "div",
                { class: "text-center font-mono" },
                actualReplicas + "/" + replicas.toString()
            );
        },
    },
    {
        accessorKey: "createdAt",
        header: "创建时间",
        cell: ({ row }) => h("div", { class: "" }, row.getValue("createdAt")),
    },
    {
        id: "actions",
        header: () => h('div', { class: 'text-center mr-2' }, "操作"),
        cell: ({ row }) =>
            h("div", { class: "flex justify-end" },
                h(AppActions, {
                    app: row.original,
                    fromAppList: true,
                    onActionCompleted: () => fetchPagedData(activeEnvRef.value?.envID),
                })
            ),
    },
];

const sorting = ref<SortingState>([]);
const columnFilters = ref<ColumnFiltersState>([]);
const columnVisibility = ref<VisibilityState>({});
const rowSelection = ref({});
const expanded = ref<ExpandedState>({});

const table = useVueTable({
    data: pagedData,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    onSortingChange: (updaterOrValue) => valueUpdater(updaterOrValue, sorting),
    onColumnFiltersChange: (updaterOrValue) =>
        valueUpdater(updaterOrValue, columnFilters),
    onColumnVisibilityChange: (updaterOrValue) =>
        valueUpdater(updaterOrValue, columnVisibility),
    onRowSelectionChange: (updaterOrValue) =>
        valueUpdater(updaterOrValue, rowSelection),
    onExpandedChange: (updaterOrValue) => valueUpdater(updaterOrValue, expanded),
    state: {
        get sorting() {
            return sorting.value;
        },
        get columnFilters() {
            return columnFilters.value;
        },
        get columnVisibility() {
            return columnVisibility.value;
        },
        get rowSelection() {
            return rowSelection.value;
        },
        get expanded() {
            return expanded.value;
        },
    },
});

const openAppForm = ref(false);

async function exportSelectedApps(appIDs: string[]) {
    if (appIDs.length === 0) {
        alert("请选择要导出的应用");
        return;
    }

    await exportApps(appIDs);
}

async function updateSelectedApps(appIDs: string[]) {
    if (appIDs.length === 0) {
        alert("请选择要更新的应用");
        return;
    }
}

const showDeleteAppsDialog = ref(false);
async function deleteSelectedApps(appIDs: string[]) {
    if (appIDs.length === 0) {
        toast.error("请选择要删除的应用");
        return;
    }

    showDeleteAppsDialog.value = true;
}

async function doDeleteSelectedApps(appIDs: string[]) {
    if (appIDs.length === 0) {
        toast.error("请选择要删除的应用");
        return;
    }

    try {
        for (const appID of appIDs) {
            await deleteApp(appID);
        }
        toast.success("删除应用成功", {
            description: `成功删除 ${appIDs.length} 个应用`,
        });
    } catch (error) {
        toast.error("删除应用失败", {
            description: `删除应用失败: ${error}`,
        });
    }
}
</script>

<template>
    <div v-if="noData"
        class="flex flex-col flex-grow text-balance text-center text-sm text-muted-foreground justify-center items-center">
        <span class="block mb-2">当前环境还没有应用，让我们先来创建一个吧！</span>
        <Button variant="default" class="my-4" @click="openAppForm = true">
            <Plus />
            创建应用
        </Button>
    </div>
    <div v-if="hasData" class="flex flex-col flex-grow">
        <div class="flex gap-2 items-center py-4 w-full">
            <Input class="max-w-sm" placeholder="搜索应用" v-model="query" @keyup.enter="onQueryEnter" />
            <DropdownMenu>
                <DropdownMenuTrigger as-child>
                    <Button variant="outline" class="ml-auto">
                        Columns
                        <ChevronDown class="ml-2 h-4 w-4" />
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                    <DropdownMenuCheckboxItem v-for="column in table
                        .getAllColumns()
                        .filter((column) => column.getCanHide())" :key="column.id" class="capitalize"
                        :model-value="column.getIsVisible()" @update:model-value="
                            (value) => {
                                column.toggleVisibility(!!value);
                            }
                        ">
                        {{ column.id }}
                    </DropdownMenuCheckboxItem>
                </DropdownMenuContent>
            </DropdownMenu>
            <DropdownMenu>
                <DropdownMenuTrigger as-child>
                    <Button variant="outline">
                        批量操作
                        <ChevronDown />
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                    <DropdownMenuItem @click="
                        updateSelectedApps(
                            table
                                .getFilteredSelectedRowModel()
                                .rows.map((row) => row.original.appID)
                        )
                        ">
                        <RefreshCcw />
                        <span class="flex items-center">滚动更新</span>
                    </DropdownMenuItem>
                    <DropdownMenuItem class="text-destructive focus:text-destructive" @select.prevent="
                        deleteSelectedApps(
                            table
                                .getFilteredSelectedRowModel()
                                .rows.map((row) => row.original.appID)
                        )
                        ">
                        <Trash class="text-destructive" />
                        <span class="flex items-center">删除</span>
                    </DropdownMenuItem>
                    <ConfirmDialog v-if="showDeleteAppsDialog" title="删除选中应用？" description="您确定要删除已选中应用吗？此操作无法撤销。"
                        @confirm="
                            doDeleteSelectedApps(
                                table
                                    .getFilteredSelectedRowModel()
                                    .rows.map((row) => row.original.appID)
                            )
                            " @cancel="showDeleteAppsDialog = false" />
                    <DropdownMenuSeparator />
                    <DropdownMenuItem @click="
                        exportSelectedApps(
                            table
                                .getFilteredSelectedRowModel()
                                .rows.map((row) => row.original.appID)
                        )
                        ">
                        <CloudUpload />
                        <span class="flex items-center">导出</span>
                    </DropdownMenuItem>
                </DropdownMenuContent>
            </DropdownMenu>
            <Button variant="default" @click="openAppForm = true">
                <Plus />
                创建应用
            </Button>
            <DropdownMenu>
                <DropdownMenuTrigger as-child>
                    <Button variant="outline">
                        <MoreVertical />
                        <span class="sr-only">Advanced options</span>
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                    <DropdownMenuItem @click="
                        exportSelectedApps(
                            table
                                .getFilteredSelectedRowModel()
                                .rows.map((row) => row.original.appID)
                        )
                        ">
                        <CloudDownload />
                        导入
                    </DropdownMenuItem>
                </DropdownMenuContent>
            </DropdownMenu>
        </div>
        <div class="rounded-md border">
            <Table>
                <TableHeader>
                    <TableRow v-for="headerGroup in table.getHeaderGroups()" :key="headerGroup.id" class="group">
                        <TableHead v-for="header in headerGroup.headers" :key="header.id"
                            :class="{ 'w-px whitespace-nowrap': header.column.id === 'actions' || header.column.id === 'select' }">
                            <FlexRender v-if="!header.isPlaceholder" :render="header.column.columnDef.header"
                                :props="header.getContext()" />
                        </TableHead>
                    </TableRow>
                </TableHeader>
                <TableBody>
                    <template v-if="table.getRowModel().rows?.length">
                        <template v-for="row in table.getRowModel().rows" :key="row.id">
                            <TableRow :data-state="row.getIsSelected() && 'selected'" class="group">
                                <TableCell v-for="cell in row.getVisibleCells()" :key="cell.id"
                                    :class="{ 'w-px whitespace-nowrap': cell.column.id === 'actions' || cell.column.id === 'select' }">
                                    <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
                                </TableCell>
                            </TableRow>
                            <TableRow v-if="row.getIsExpanded()">
                                <TableCell :colspan="row.getAllCells().length">
                                    {{ JSON.stringify(row.original) }}
                                </TableCell>
                            </TableRow>
                        </template>
                    </template>

                    <TableRow v-else>
                        <TableCell :colspan="columns.length" class="h-24 text-center">
                            No results.
                        </TableCell>
                    </TableRow>
                </TableBody>
            </Table>
        </div>
        <div class="flex items-center justify-end space-x-2 py-4">
            <div class="flex-1 text-sm text-muted-foreground ml-2">
                <span v-if="table.getFilteredSelectedRowModel().rows.length">
                    已选中
                    <span class="font-mono">{{ table.getFilteredSelectedRowModel().rows.length }}
                    </span>
                    条记录，
                </span>
                <span>
                    共 <span class="font-mono">{{ totalCount }}</span> 条记录
                </span>
            </div>
            <div class="space-x-2">
                <Pagination v-model:page="queryModel.pageNo" :total="totalCount" :items-per-page="queryModel.pageSize"
                    :sibling-count="1" class="ml-auto">
                    <PaginationContent v-slot="{ items }">
                        <PaginationFirst>
                            <ChevronsLeft class="h-4 w-4" />
                        </PaginationFirst>
                        <PaginationPrevious>
                            <ChevronLeft class="h-4 w-4" />
                        </PaginationPrevious>

                        <template v-for="(item, index) in items">
                            <PaginationItem v-if="item.type === 'page'" :key="index" :value="item.value"
                                :class="queryModel.pageNo === item.value ? 'bg-secondary' : ''">
                                {{ item.value }}
                            </PaginationItem>
                            <PaginationEllipsis v-else :key="item.type" :index="index" />
                        </template>

                        <PaginationNext>
                            <ChevronRight class="h-4 w-4" />
                        </PaginationNext>
                        <PaginationLast>
                            <ChevronsRight class="h-4 w-4" />
                        </PaginationLast>
                    </PaginationContent>
                </Pagination>
            </div>
        </div>
    </div>
    <CreateApp v-model="openAppForm" @app-created="fetchPagedData(activeEnvRef?.envID)" />
</template>
