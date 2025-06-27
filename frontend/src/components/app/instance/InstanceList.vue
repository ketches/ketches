<script setup lang="ts">
import { listAppInstances } from "@/api/app";
import Badge from "@/components/ui/badge/Badge.vue";
import { Button } from "@/components/ui/button";
import Label from "@/components/ui/label/Label.vue";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { useResourceRefStore } from "@/stores/resourceRefStore";
import type { appInstanceModel } from "@/types/app";
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
import { RefreshCcw } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { h, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { appInstanceStatusDisplay } from "../data/appInstanceStatus";
import InstanceActions from "./InstanceActions.vue";

const route = useRoute();
const appID = route.params.id as string;

const resourceRefStore = useResourceRefStore();
const { activeAppRef } = storeToRefs(resourceRefStore);

const listData = ref<appInstanceModel[]>([]);
const totalCount = ref(0);

async function fetchListData(appID?: string) {
    if (appID) {
        const { instances } = (await listAppInstances(appID)) || {
            edition: 0,
            instances: [],
        };
        listData.value = instances;
        totalCount.value = instances.length;
    }
}

onMounted(async () => {
    await fetchListData(appID);
});

watch(activeAppRef, async (newAppRef) => {
    if (newAppRef && newAppRef.appID !== appID) {
        await fetchListData(newAppRef.appID);
    }
});

const centeredHeader = (text: string) => h("div", { class: "text-center" }, text);

const columns: ColumnDef<appInstanceModel>[] = [
    {
        accessorKey: "instanceName",
        header: () => h("div", { class: "ml-2" }, "实例名称"),
        cell: ({ row }) =>
            h("div", { class: "space-y-1 ml-2" }, [
                h(
                    "div",
                    { class: "font-medium" },
                    row.original.instanceName
                ),
                h(
                    "div",
                    { class: "text-sm text-muted-foreground font-mono" },
                    row.original.instanceIP
                ),
            ]),
    },
    {
        accessorKey: "status",
        header: () => centeredHeader("状态"),
        cell: ({ row }) => {
            const instanceStatusDisplay = appInstanceStatusDisplay(row.getValue("status"));
            return h("div", { class: `flex items-center justify-center` }, [
                h(
                    Badge,
                    {
                        variant: "secondary",
                        class: `capitalize flex justify-center text-center ${instanceStatusDisplay.fgColor}`,
                    },
                    () => [h(instanceStatusDisplay.icon, {}, instanceStatusDisplay.label), h("span", {}, instanceStatusDisplay.label)],
                ),
            ]);
        },
    },
    {
        accessorKey: "containerCount",
        header: "容器数",
        cell: ({ row }) =>
            h("div", { class: "font-mono" }, row.getValue("containerCount")),
    },
    {
        accessorKey: "nodeName",
        header: "所在节点",
        cell: ({ row }) => h("div", { class: "space-y-1 ml-2" }, [
            h(
                "div",
                { class: "font-medium" },
                row.original.nodeName
            ),
            h(
                "div",
                { class: "text-sm text-muted-foreground font-mono" },
                row.original.nodeIP
            ),
        ]),
    },
    {
        accessorKey: "runningDuration",
        header: () => centeredHeader("运行时长"),
        cell: ({ row }) =>
            h("div", { class: "text-center" }, row.getValue("runningDuration")),
    },
    {
        id: "actions",
        header: () => h('div', { class: 'text-center mr-2' }, "操作"),
        cell: ({ row }) =>
            h("div", { class: "flex justify-end mr-2" },
                h(InstanceActions, {
                    appInstance: row.original,
                    onActionCompleted: () =>
                        fetchListData(activeAppRef.value?.appID),
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
    data: listData,
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
</script>

<template>
    <div class="flex flex-col flex-grow">
        <div class="flex justify-between items-center py-4 w-full">
            <Label class="text-muted-foreground font-medium mx-2">应用实例</Label>
            <div class="flex gap-2">
                <Button variant="secondary" size="sm" @click="fetchListData(activeAppRef?.appID)">
                    <RefreshCcw class="w-4 h-4 mr-2" />
                    刷新
                </Button>
            </div>
        </div>
        <div class="rounded-md border">
            <Table>
                <TableHeader>
                    <TableRow v-for="headerGroup in table.getHeaderGroups()" :key="headerGroup.id" class="group">
                        <TableHead v-for="header in headerGroup.headers" :key="header.id"
                            :class="{ 'w-px whitespace-nowrap': header.column.id === 'actions' }">
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
                                    :class="{ 'w-px whitespace-nowrap': cell.column.id === 'actions' }">
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
    </div>
</template>
