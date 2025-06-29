<script setup lang="ts">
import { listProjectMembers } from '@/api/project'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
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
import type { QueryAndPagedRequest } from '@/types/common'
import { type projectMemberModel } from '@/types/project'
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
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, Plus } from 'lucide-vue-next'
import { storeToRefs } from 'pinia'
import { h, onMounted, ref, watch } from 'vue'
import AddMember from './AddMember.vue'
import MemberActions from './MemberActions.vue'
import RoleCell from './RoleCell.vue'

const userStore = useUserStore()
const { activeProjectRef } = storeToRefs(userStore)

const pagedData = ref<projectMemberModel[]>([])
const totalCount = ref(0)

const queryModel = ref<QueryAndPagedRequest>({
    pageNo: 1,
    pageSize: 10,
    query: '',
    sortBy: 'created_at',
    sortOrder: 'DESC',
})

async function fetchPagedData(projectID?: string) {
    if (projectID) {
        const { total, records } = await listProjectMembers(projectID, queryModel.value) || { total: 0, records: [] }
        pagedData.value = records
        totalCount.value = total
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
    await fetchPagedData(activeProjectRef.value?.projectID)
}, { deep: true })

onMounted(async () => {
    await fetchPagedData(activeProjectRef.value?.projectID)
})

watch(
    () => activeProjectRef.value,
    async (newActiveProject) => {
        queryModel.value.pageNo = 1
        await fetchPagedData(newActiveProject?.projectID)
    }
)

const centeredHeader = (text: string) => h('div', { class: 'text-center' }, text)

const columns: ColumnDef<projectMemberModel>[] = [
    {
        id: 'select',
        header: ({ table }) => h(Checkbox, {
            'class': [table.getIsAllPageRowsSelected() || table.getIsSomePageRowsSelected() ? '' : 'invisible group-hover:visible'],
            'modelValue':
                table.getIsAllPageRowsSelected()
                    ? true
                    : table.getIsSomePageRowsSelected()
                        ? 'indeterminate'
                        : false,
            'onUpdate:modelValue': value => table.toggleAllPageRowsSelected(!!value),
            'ariaLabel': 'Select all',
        }),
        cell: ({ row }) => h(Checkbox, {
            'class': [row.getIsSelected() ? '' : 'invisible group-hover:visible'],
            'modelValue': row.getIsSelected(),
            'onUpdate:modelValue': value => row.toggleSelected(!!value),
            'ariaLabel': 'Select row',
        }),
        enableSorting: false,
        enableHiding: false,
    },
    {
        accessorKey: 'username',
        header: "用户名",
        cell: ({ row }) => h('div', { class: 'space-y-1' }, [
            h('div', { class: 'font-medium' }, row.original.fullname || row.getValue('username')),
            h('div', { class: 'text-sm text-muted-foreground font-mono' }, row.getValue('username'))
        ]),
    },
    {
        accessorKey: 'email',
        header: "邮箱",
        cell: ({ row }) => h('div', { class: '' }, row.getValue('email')),
    },
    {
        accessorKey: 'phone',
        header: "手机号码",
        cell: ({ row }) => h('div', { class: '' }, row.getValue('phone')),
    },
    {
        accessorKey: 'projectRole',
        header: () => centeredHeader('角色'),
        cell: ({ row }) => {
            return h(RoleCell, {
                member: row.original,
                activeProjectID: activeProjectRef.value?.projectID || '',
                onRoleUpdated: () => fetchPagedData(activeProjectRef.value?.projectID)
            });
        },
    },
    {
        accessorKey: 'createdAt',
        header: () => centeredHeader('加入时间'),
        cell: ({ row }) => h('div', { class: 'text-center' }, String(row.getValue('createdAt')).slice(0, 10)),
    },
    {
        id: 'actions',
        header: () => centeredHeader('操作'),
        cell: ({ row }) => h('div', { class: "flex justify-end mr-2" }, h(MemberActions, {
            member: row.original,
            onActionCompleted: () => fetchPagedData(activeProjectRef.value?.projectID)
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

const addMember = ref(false)
</script>

<template>
    <div class="flex flex-col flex-grow">
        <div class="flex gap-2 items-center py-4 w-full">
            <Input class="max-w-sm" placeholder="搜索成员" v-model="query" @keyup.enter="onQueryEnter" />
            <Button variant="default" @click="addMember = true" class="ml-auto">
                <Plus />
                添加成员
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
    <AddMember v-model="addMember" @member-added="fetchPagedData(activeProjectRef?.projectID)" />
</template>
