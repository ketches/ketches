<script setup lang="ts">
import { h, ref } from 'vue';

import { createAppEnvVar, deleteAppEnvVar, listAppEnvVars, updateAppEnvVar } from '@/api/app';
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
import type { appEnvVarModel, appModel } from '@/types/app';
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
import { Check, ChevronsLeft, ChevronsRight, Delete, Plus, SquarePen, X } from 'lucide-vue-next';
import { onMounted } from 'vue';
import { toast } from 'vue-sonner';

const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
});

const app = ref<appModel>(props.app)

const listData = ref<appEnvVarModel[]>([])

async function fetchListData(appID?: string) {
    if (appID) {
        const records = await listAppEnvVars(appID)
        listData.value = records
    }
}

onMounted(async () => {
    await fetchListData(app.value.appID)
})


const centeredHeader = (text: string) => h('div', { class: 'text-center' }, text)

const editingRowId = ref<string | null>(null)
const editKey = ref('')
const editValue = ref('')

const columns: ColumnDef<appEnvVarModel>[] = [
    {
        id: 'select',
        header: ({ table }) => h(Checkbox, {
            'class': [table.getIsAllPageRowsSelected() || table.getIsSomePageRowsSelected() ? '' : 'invisible group-hover:visible'],
            'modelValue': table.getIsAllPageRowsSelected() || (table.getIsSomePageRowsSelected() && 'indeterminate'),
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
        accessorKey: 'key',
        header: "变量名",
        cell: ({ row }) => {
            if (editingRowId.value === row.id || row.original.envVarID === 'new') {
                return h(Input, {
                    modelValue: editKey.value,
                    disabled: row.original.envVarID && row.original.envVarID !== 'new',
                    'onUpdate:modelValue': (val: string | number) => editKey.value = String(val),
                    class: 'font-mono'
                })
            }
            return h('div', { class: 'font-mono' }, row.original.key);
        },
    },
    {
        accessorKey: 'value',
        header: "变量值",
        cell: ({ row }) => {
            if (editingRowId.value === row.id || row.original.envVarID === 'new') {
                return h(Input, {
                    modelValue: editValue.value,
                    'onUpdate:modelValue': (val: string | number) => editValue.value = String(val),
                })
            }
            return h('div', {}, row.original.value);
        },
    },
    {
        id: 'actions',
        header: () => centeredHeader('操作'),
        cell: ({ row }) => {
            if (editingRowId.value === row.id || row.original.envVarID === 'new') {
                return h('div', { class: "flex justify-end mr-2 gap-2" }, [
                    h(Button, {
                        variant: 'outline',
                        class: 'text-green-500',
                        onClick: async () => {
                            if (editingRowId.value === 'new') {
                                const resp = await createAppEnvVar(app.value.appID, {
                                    key: editKey.value,
                                    value: editValue.value,
                                })
                                listData.value.push(resp)
                                toast.success('添加环境变量成功')
                                // editingRowId.value = null
                                // await fetchListData(app.value.appID)
                            } else {
                                const resp = await updateAppEnvVar(row.original.appID, row.original.envVarID, {
                                    value: editValue.value,
                                })
                                listData.value = listData.value.map(item => {
                                    if (item.envVarID === row.original.envVarID) {
                                        return { ...item, value: resp.value }
                                    }
                                    return item
                                })
                                // resetEnvVarRow()
                                toast.success('环境变量已更新')
                                // await fetchListData(app.value.appID)
                            }
                            resetEnvVarRow()
                        }
                    }, [
                        h(Check),
                    ]),
                    h(Button, {
                        variant: 'outline',
                        onClick: () => resetEnvVarRow()
                    }, [
                        h(X),
                    ]),
                ])
            }
            return h('div', { class: "flex justify-end mr-2 gap-2" }, [
                h(Button, {
                    variant: 'outline',
                    onClick: () => {
                        editingRowId.value = row.id
                        editKey.value = row.original.key
                        editValue.value = row.original.value
                    }
                }, () => [h(SquarePen)]),
                h(Button, {
                    variant: 'outline',
                    onClick: async () => {
                        await deleteAppEnvVar(row.original.appID, row.original.envVarID)
                        listData.value = listData.value.filter(item => item.envVarID !== row.original.envVarID)
                        toast.success('环境变量已删除')
                        // await fetchListData(app.value.appID)
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

function addEnvVarRow() {
    if (editingRowId.value) {
        toast.warning('请先保存当前编辑的环境变量')
        return
    }
    editingRowId.value = 'new'
    listData.value = [{
        appID: app.value.appID,
        envVarID: editingRowId.value,
        key: '',
        value: '',
    } as appEnvVarModel,
    ...listData.value,
    ];
}

function resetEnvVarRow() {
    listData.value = listData.value.filter(row => row.envVarID !== 'new');
    editingRowId.value = null
    editKey.value = ''
    editValue.value = ''
}


const table = useVueTable({
    data: listData,
    columns,
    getRowId: row => row.envVarID,
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
    <div>
        <h3 class="text-lg font-medium">
            环境变量
        </h3>
        <p class="text-sm text-muted-foreground">
            配置应用程序运行时所需的环境变量。
        </p>
    </div>
    <Separator />
    <div class="flex flex-col flex-grow">
        <div class="flex gap-2 items-center pb-4 w-full">
            <Input class="max-w-sm" placeholder="搜索环境变量"
                :model-value="table.getColumn('key')?.getFilterValue() as string"
                @update:model-value=" table.getColumn('key')?.setFilterValue($event)" />
            <Button variant="outline" size="sm" class="ml-auto" @click="addEnvVarRow">
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
                            No results.
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
