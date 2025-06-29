<script setup lang="ts">
import { listClusters } from '@/api/cluster'
import { Button } from '@/components/ui/button'
import {
    DropdownMenu,
    DropdownMenuCheckboxItem,
    DropdownMenuContent,
    DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
    Pagination,
    PaginationContent,
    PaginationEllipsis,
    PaginationFirst,
    PaginationItem,
    PaginationLast,
    PaginationNext,
    PaginationPrevious,
} from '@/components/ui/pagination'
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/table'
import { useUserStore } from '@/stores/userStore'
import type { clusterModel } from '@/types/cluster'
import type { QueryAndPagedRequest } from '@/types/common'
import { valueUpdater } from '@/utils/valueUpdater'
import type {
    ColumnDef,
    ColumnFiltersState,
    ExpandedState,
    SortingState,
    VisibilityState,
} from '@tanstack/vue-table'
import {
    FlexRender,
    getCoreRowModel,
    getExpandedRowModel,
    getFilteredRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    useVueTable,
} from '@tanstack/vue-table'
import { ChevronDown, ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, Plus } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { h, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import ClusterActions from './ClusterActions.vue'
import CreateCluster from './CreateCluster.vue'

const userStore = useUserStore()
const { activeProjectRef } = storeToRefs(userStore)

const noData = ref(false)
const hasData = ref(false)
const pagedData = ref<clusterModel[]>([])
const totalCount = ref(0)

const queryModel = ref<QueryAndPagedRequest>({
    pageNo: 1,
    pageSize: 10,
    query: '',
    sortBy: 'created_at',
    sortOrder: 'DESC',
})

async function fetchPagedData() {
    const { total, records } = await listClusters(queryModel.value) || {
        total: 0,
        records: [],
    }
    pagedData.value = records
    totalCount.value = total

    if (!queryModel.value.query) {
        noData.value = total === 0
        hasData.value = total > 0
    }
}

const query = ref('')
function onQueryEnter() {
    if (queryModel.value.query !== query.value) {
        queryModel.value.query = query.value
        queryModel.value.pageNo = 1; // Reset to first page on new query
    }
}

watch(queryModel, async () => {
    await fetchPagedData()
}, { deep: true })

onMounted(async () => {
    await fetchPagedData()
})

watch(activeProjectRef, async (newActiveProject) => {
    await fetchPagedData()
})

const centeredHeader = (text: string) => h('div', { class: 'text-center' }, text)

const columns: ColumnDef<clusterModel>[] = [
    {
        accessorKey: 'slug',
        header: () => h("div", { class: "ml-2" }, "集群"),
        cell: ({ row }) => h("div", { class: "space-y-1 ml-2" }, [
            h(
                RouterLink,
                {
                    to: `/console/cluster/${row.original.clusterID}`,
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
        accessorKey: 'createdAt',
        header: '创建时间',
        cell: ({ row }) => h('div', { class: '' }, row.getValue('createdAt')),
    },
    {
        id: 'actions',
        header: () => h('div', { class: 'text-center mr-2' }, "操作"),
        cell: ({ row }) => h('div', { class: "flex justify-end mr-2" }, h(ClusterActions, {
            cluster: row.original,
            onActionCompleted: () => fetchPagedData()
        })),
    },
]

const sorting = ref<SortingState>([])
const columnFilters = ref<ColumnFiltersState>([])
const columnVisibility = ref<VisibilityState>({})
const rowSelection = ref({})
const expanded = ref<ExpandedState>({})

const table = useVueTable({
    data: pagedData,
    columns,
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

const openClusterForm = ref(false)
</script>

<template>
    <div v-if="noData"
        class="flex flex-col flex-grow text-balance text-center text-sm text-muted-foreground justify-center items-center">
        <span class="block mb-2">当前项目还没有集群，让我们先来创建一个吧！</span>
        <Button variant="default" class="my-4" @click="openClusterForm = true">
            <Plus />
            创建集群
        </Button>
    </div>
    <div v-if="hasData" class="flex flex-col flex-grow">
        <div class="flex gap-2 items-center py-4 w-full">
            <Input class="max-w-sm" placeholder="搜索集群" v-model="query" @keyup.enter="onQueryEnter" />
            <DropdownMenu>
                <DropdownMenuTrigger as-child>
                    <Button variant="outline" class="ml-auto">
                        Columns
                        <ChevronDown class="ml-2 h-4 w-4" />
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                    <DropdownMenuCheckboxItem
                        v-for="column in table.getAllColumns().filter((column) => column.getCanHide())" :key="column.id"
                        class="capitalize" :model-value="column.getIsVisible()" @update:model-value="(value) => {
                            column.toggleVisibility(!!value)
                        }">
                        {{ column.id }}
                    </DropdownMenuCheckboxItem>
                </DropdownMenuContent>
            </DropdownMenu>
            <Button variant="default" @click="openClusterForm = true">
                <Plus />
                创建集群
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
                            No results.
                        </TableCell>
                    </TableRow>
                </TableBody>
            </Table>
        </div>
        <div class="flex items-center justify-end space-x-2 py-4">
            <div class="flex-1 text-sm text-muted-foreground ml-2">
                <span v-if="table.getFilteredSelectedRowModel().rows.length">
                    已选中 <span class="font-mono">{{ table.getFilteredSelectedRowModel().rows.length }}
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
    <CreateCluster v-model="openClusterForm" @cluster-created="fetchPagedData()" />
</template>
