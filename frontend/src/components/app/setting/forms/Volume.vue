<script setup lang="ts">
import { deleteAppVolume, listAppVolumes } from '@/api/app';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/table';
import type { appModel, appVolumeModel } from '@/types/app';
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
import { ChevronsLeft, ChevronsRight, Delete, Plus, SquarePen } from 'lucide-vue-next';
import { h, onMounted, ref, toRef, watch } from 'vue';
import { toast } from 'vue-sonner';
import { accessModeRefs, volumeModeRefs } from '../data/settings';
import CreateVolume from './CreateVolume.vue';
import UpdateVolume from './UpdateVolume.vue';

const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
});

const app = toRef(props, 'app');
const listData = ref<appVolumeModel[]>([])

async function fetchListData(appID?: string) {
    if (appID) {
        const records = await listAppVolumes(appID)
        listData.value = records
    }
}

onMounted(async () => {
    await fetchListData(app.value.appID)
})

watch(app, async (newApp) => {
    if (newApp.appID) {
        await fetchListData(newApp.appID)
    }
})

const centeredHeader = (text: string) => h('div', { class: 'text-center' }, text)

const columns: ColumnDef<appVolumeModel>[] = [
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
        accessorKey: 'slug',
        header: "存储卷",
        cell: ({ row }) => h('div', { class: 'font-mono' }, row.original.slug),
    },
    {
        accessorKey: 'volumeType',
        header: () => centeredHeader('存储类型'),
        cell: ({ row }) => h('div', { class: 'text-center' }, row.original.volumeType),
    },
    {
        accessorKey: 'mountPath',
        header: "挂载路径",
        cell: ({ row }) => h("div", { class: "space-y-1 ml-2" }, row.original.subPath ? [
            h(
                "div",
                { class: "font-mono" },
                row.original.mountPath
            ),
            h(
                "div",
                { class: "text-sm text-muted-foreground font-mono" },
                row.original.subPath
            ),
        ] : [
            h(
                "div",
                { class: "font-mono" },
                row.original.mountPath
            ),
        ]),
    },
    {
        accessorKey: 'storageClass',
        header: () => centeredHeader('存储类'),
        cell: ({ row }) => h('div', { class: 'text-center' }, row.original.storageClass || '-'),
    },
    {
        accessorKey: 'capacity',
        header: () => centeredHeader('容量'),
        cell: ({ row }) => h('div', { class: 'text-center' }, row.original.capacity + 'Gi'),
    },
    {
        accessorKey: 'accessModes',
        header: () => centeredHeader('访问模式'),
        cell: ({ row }) => h('div', { class: 'text-center' }, row.original.accessModes.map(mode => h(Badge, { variant: "secondary" }, accessModeRefs[mode].label || mode))),
    },
    {
        accessorKey: 'volumeMode',
        header: () => centeredHeader('存储模式'),
        cell: ({ row }) => h('div', { class: 'text-center' }, volumeModeRefs[row.original.volumeMode || 'Filesystem']?.label),
    },
    {
        id: 'actions',
        header: () => centeredHeader('操作'),
        cell: ({ row }) => {
            return h('div', { class: "flex justify-end mr-2 gap-2" }, [
                h(Button, {
                    variant: 'outline',
                    onClick: () => {
                        selectedVolume.value = row.original
                        openUpdateVolumeDialog.value = true
                    }
                }, () => [h(SquarePen)]),
                h(Button, {
                    variant: 'outline',
                    onClick: async () => {
                        await deleteAppVolume(row.original.appID, row.original.volumeID)
                        listData.value = listData.value.filter(item => item.volumeID !== row.original.volumeID)
                        toast.success('存储卷已删除')
                    }
                }, () => [h(Delete)])
            ])
        }
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
    getRowId: row => row.volumeID,
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

const openAddVolumeDialog = ref(false)
const openUpdateVolumeDialog = ref(false)
const selectedVolume = ref<appVolumeModel | null>(null)
</script>

<template>
    <div>
        <h3 class="text-lg font-medium">
            存储卷
        </h3>
        <p class="text-sm text-muted-foreground">
            配置应用存储卷，例如数据库或文件存储。
        </p>
    </div>
    <Separator />
    <div class="flex flex-col flex-grow">
        <div class="flex gap-2 items-center pb-4 w-full">
            <Input class="max-w-sm" placeholder="搜索环境变量"
                :model-value="table.getColumn('slug')?.getFilterValue() as string"
                @update:model-value=" table.getColumn('slug')?.setFilterValue($event)" />
            <Button variant="outline" size="sm" class="ml-auto" @click="openAddVolumeDialog = true">
                <Plus />
                新增
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
    <CreateVolume v-model="openAddVolumeDialog" :appID="app.appID" @close="openAddVolumeDialog = false"
        @volume-created="fetchListData(app.appID)" />
    <UpdateVolume v-model="openUpdateVolumeDialog" v-if="selectedVolume" :volume="selectedVolume"
        @close="openUpdateVolumeDialog = false" @volume-updated="fetchListData(app.appID)" />
</template>
