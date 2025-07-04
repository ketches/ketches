<script setup lang="ts">
import { h, ref, toRef, watch } from 'vue';

import { listClusterNodes } from '@/api/cluster';
import Badge from '@/components/ui/badge/Badge.vue';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/table';
import type { clusterModel, clusterNodeModel } from '@/types/cluster';
import { valueUpdater } from '@/utils/valueUpdater';
import type {
    ColumnDef,
    ColumnFiltersState,
    ExpandedState,
    SortingState,
    VisibilityState,
} from '@tanstack/vue-table';
import {
    FlexRender,
    getCoreRowModel,
    getExpandedRowModel,
    getFilteredRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    useVueTable,
} from '@tanstack/vue-table';
import { ChevronsLeft, ChevronsRight, CircleCheck, CircleDashed, RefreshCcw } from 'lucide-vue-next';
import { onMounted } from 'vue';
import { RouterLink } from 'vue-router';
import NodeActions from './NodeActions.vue';

const props = defineProps({
    cluster: {
        type: Object as () => clusterModel,
        required: true,
    },
});

const cluster = toRef(props, 'cluster');

const listData = ref<clusterNodeModel[]>([])

async function fetchListData(clusterID?: string) {
    if (clusterID) {
        const records = await listClusterNodes(clusterID)
        listData.value = records
    }
}

onMounted(async () => {
    await fetchListData(cluster.value?.clusterID)
})

watch(cluster, async (newCluster) => {
    if (newCluster.clusterID) {
        await fetchListData(newCluster.clusterID)
    }
})

const centeredHeader = (text: string) => h('div', { class: 'text-center' }, text)

const columns: ColumnDef<clusterNodeModel>[] = [
    {
        accessorKey: 'nodeName',
        header: "节点名称",
        cell: ({ row }) => {
            return h("div", { class: "space-y-1" }, [
                h(
                    RouterLink,
                    {
                        to: { name: 'clusterNodePage', params: { id: row.original.clusterID, nodeName: row.original.nodeName } },
                        class: "font-medium text-blue-500",
                    },
                    () => row.original.nodeName || row.getValue("internalIP")
                ),
                h(
                    "div",
                    { class: "text-sm text-muted-foreground font-mono" },
                    row.original.internalIP || "未知 IP"
                ),
            ]);
        }
    },
    {
        accessorKey: "roles",
        header: () => centeredHeader("角色"),
        cell: ({ row }) =>
            h(
                "div",
                { class: "capitalize text-center" },
                row.original.roles.join(", ") || "无"
            ),
    },
    {
        accessorKey: "osImage",
        header: () => centeredHeader("服务器"),
        cell: ({ row }) => {
            return h('div', { class: "text-center font-mono" },
                row.original.osImage || '未知'
            );
        }
    },
    {
        accessorKey: "operatingSystem",
        header: () => centeredHeader("系统架构"),
        cell: ({ row }) => {
            const os = row.original.operatingSystem;
            const arch = row.original.architecture;
            return h('div', { class: "text-center font-mono" },
                os && arch ? `${os}/${arch}` : '未知'
            );
        }
    },
    {
        accessorKey: "kubeletVersion",
        header: () => centeredHeader("Kubernetes 版本"),
        cell: ({ row }) => {
            return h('div', { class: "text-center font-mono" },
                row.original.kubeletVersion || '未知'
            );
        }
    },
    {
        accessorKey: "containerRuntimeVersion",
        header: () => centeredHeader("容器运行时"),
        cell: ({ row }) => {
            return h('div', { class: "text-center font-mono" },
                row.original.containerRuntimeVersion || '未知'
            );
        }
    },
    {
        accessorKey: "ready",
        header: () => centeredHeader("状态"),
        cell: ({ row }) => {
            return h('div', { class: "text-center" }, [
                h(Badge, {
                    variant: 'secondary',
                    class: row.original.ready ? 'text-green-500' : 'text-gray-500',
                }, [
                    h(row.original.ready ? CircleCheck : CircleDashed, { class: 'w-4 h-4 mr-1' }),
                    h('span', row.original.ready ? '就绪' : '未就绪')
                ])
            ]);
        }
    },
    {
        id: "actions",
        header: () => h('div', { class: 'text-center mr-2' }, "操作"),
        cell: ({ row }) =>
            h("div", { class: "flex justify-end" },
                h(NodeActions, {
                    node: row.original,
                    fromNodeList: true,
                })
            ),
    },
]

const sorting = ref<SortingState>([])
const columnFilters = ref<ColumnFiltersState>([])
const columnVisibility = ref<VisibilityState>({})
const rowSelection = ref({})
const expanded = ref<ExpandedState>({})

const table = useVueTable({
    data: listData,
    columns,
    getRowId: row => row.nodeName,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getExpandedRowModel: getExpandedRowModel(),
    onSortingChange: updaterOrValue => valueUpdater(updaterOrValue, sorting),
    onColumnFiltersChange: updaterOrValue => valueUpdater(updaterOrValue, columnFilters),
    onColumnVisibilityChange: updaterOrValue => valueUpdater(updaterOrValue, columnVisibility),
    onRowSelectionChange: updaterOrValue => valueUpdater(updaterOrValue, rowSelection),
    onExpandedChange: updaterOrValue => valueUpdater(updaterOrValue, expanded),
    state: {
        get sorting() { return sorting.value },
        get columnFilters() { return columnFilters.value },
        get columnVisibility() { return columnVisibility.value },
        get rowSelection() { return rowSelection.value },
        get expanded() { return expanded.value },
    },
})

</script>

<template>
    <div class="flex flex-col flex-grow">
        <div class="flex gap-2 items-center pb-4 w-full">
            <Input class="max-w-sm" placeholder="搜索节点"
                :model-value="table.getColumn('nodeName')?.getFilterValue() as string"
                @update:model-value=" table.getColumn('nodeName')?.setFilterValue($event)" />
            <Button variant="outline" size="sm" class="ml-auto" @click="fetchListData(cluster?.clusterID)">
                <RefreshCcw />
                刷新
            </Button>
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
                            <span class="text-muted-foreground">No results.</span>
                        </TableCell>
                    </TableRow>
                </TableBody>
            </Table>
        </div>
        <div class="flex items-center justify-end space-x-2 py-4">
            <div class="space-x-2">
                <Button variant="outline" size="sm" :disabled="!table.getCanPreviousPage()"
                    v-if="table.getCanPreviousPage()" @click="table.previousPage()">
                    <ChevronsLeft class="h-4 w-4" />
                </Button>
                <Button variant="outline" size="sm" :disabled="!table.getCanNextPage()" v-if="table.getCanNextPage()"
                    @click="table.nextPage()">
                    <ChevronsRight class="h-4 w-4" />
                </Button>
            </div>
        </div>
    </div>
</template>
