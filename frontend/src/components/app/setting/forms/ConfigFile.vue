<script setup lang="ts">
import { deleteAppConfigFiles, listAppConfigFiles } from '@/api/app';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import Separator from '@/components/ui/separator/Separator.vue';
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from '@/components/ui/table';
import type { appConfigFileModel, appModel } from '@/types/app';
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
import { computed, h, onMounted, ref } from 'vue';
import { toast } from 'vue-sonner';
import CreateConfigFile from './CreateConfigFile.vue';
import UpdateConfigFile from './UpdateConfigFile.vue';

const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
});

const data = ref<appConfigFileModel[]>([]);
const sorting = ref<SortingState>([]);
const columnFilters = ref<ColumnFiltersState>([]);
const columnVisibility = ref<VisibilityState>({});
const rowSelection = ref({});
const expanded = ref<ExpandedState>({});

const centeredHeader = (text: string) => h('div', { class: 'text-center' }, text)
const columns: ColumnDef<appConfigFileModel>[] = [
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
        header: "配置文件",
        cell: ({ row }) => h('div', { class: 'font-mono' }, row.original.slug),
    },
    {
        accessorKey: 'mountPath',
        header: '挂载路径',
        cell: ({ row }) => h('div', { class: 'font-mono text-sm' }, row.getValue('mountPath')),
    },
    {
        accessorKey: 'fileMode',
        header: '文件权限',
        cell: ({ row }) => h(Badge, { variant: 'outline' }, () => row.getValue('fileMode')),
    },
    {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
            const configFile = row.original;
            return h('div', { class: 'flex items-center gap-2' }, [
                h(Button, {
                    variant: 'outline',
                    size: 'sm',
                    onClick: () => editConfigFile(configFile),
                }, () => h(SquarePen)),
                h(Button, {
                    variant: 'outline',
                    size: 'sm',
                    class: "text-destructive hover:text-destructive border-destructive/20 hover:bg-destructive/20",
                    onClick: () => deleteConfigFile(configFile.configFileID),
                }, () => h(Delete)),
            ]);
        },
    },
];

const table = useVueTable({
    data,
    columns,
    onSortingChange: updaterOrValue => valueUpdater(updaterOrValue, sorting),
    onColumnFiltersChange: updaterOrValue => valueUpdater(updaterOrValue, columnFilters),
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: updaterOrValue => valueUpdater(updaterOrValue, columnVisibility),
    onRowSelectionChange: updaterOrValue => valueUpdater(updaterOrValue, rowSelection),
    onExpandedChange: updaterOrValue => valueUpdater(updaterOrValue, expanded),
    getExpandedRowModel: getExpandedRowModel(),
    state: {
        get sorting() { return sorting.value },
        get columnFilters() { return columnFilters.value },
        get columnVisibility() { return columnVisibility.value },
        get rowSelection() { return rowSelection.value },
        get expanded() { return expanded.value },
    },
});

const openAddConfigFileDialog = ref(false);
const showUpdateDialog = ref(false);
const editingConfigFile = ref<appConfigFileModel | null>(null);

const selectedConfigFiles = computed(() => {
    return table.getFilteredSelectedRowModel().rows.map(row => row.original);
});

async function loadConfigFiles() {
    try {
        data.value = await listAppConfigFiles(props.app.appID);
    } catch (error) {
        console.error('Failed to load config files:', error);
        toast.error('加载配置文件失败');
    }
}

function addConfigFile() {
    openAddConfigFileDialog.value = true;
}

function editConfigFile(configFile: appConfigFileModel) {
    editingConfigFile.value = configFile;
    showUpdateDialog.value = true;
}

async function deleteConfigFile(configFileID: string) {
    try {
        await deleteAppConfigFiles(props.app.appID, [configFileID]);
        toast.success('配置文件删除成功');
        await loadConfigFiles();
    } catch (error) {
        console.error('Failed to delete config file:', error);
        toast.error('删除配置文件失败');
    }
}

async function deleteSelectedConfigFiles() {
    if (selectedConfigFiles.value.length === 0) {
        toast.warning('请选择要删除的配置文件');
        return;
    }

    try {
        const configFileIDs = selectedConfigFiles.value.map(cf => cf.configFileID);
        await deleteAppConfigFiles(props.app.appID, configFileIDs);
        toast.success(`成功删除 ${configFileIDs.length} 个配置文件`);
        rowSelection.value = {};
        await loadConfigFiles();
    } catch (error) {
        console.error('Failed to delete config files:', error);
        toast.error('删除配置文件失败');
    }
}

function onConfigFileCreated() {
    openAddConfigFileDialog.value = false;
    loadConfigFiles();
}

function onConfigFileUpdated() {
    showUpdateDialog.value = false;
    editingConfigFile.value = null;
    loadConfigFiles();
}

onMounted(() => {
    loadConfigFiles();
});
</script>

<template>
    <div>
        <h3 class="text-lg font-medium">
            配置文件
        </h3>
        <p class="text-sm text-muted-foreground">
            配置文件会挂载到容器的指定路径下。
        </p>
    </div>
    <Separator />
    <div class="flex flex-col flex-grow">
        <div class="flex gap-2 items-center pb-4 w-full">
            <Input placeholder="搜索配置文件..." :model-value="table.getColumn('slug')?.getFilterValue() as string"
                @update:model-value="table.getColumn('slug')?.setFilterValue($event)" class="max-w-sm" />
            <Button variant="outline" size="sm" class="ml-auto" @click="addConfigFile">
                <Plus />
                新增
            </Button>
        </div>
        <div class="rounded-md border">
            <Table>
                <TableHeader>
                    <TableRow v-for="headerGroup in table.getHeaderGroups()" :key="headerGroup.id">
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
                            <TableRow :data-state="row.getIsSelected() && 'selected'">
                                <TableCell v-for="cell in row.getVisibleCells()" :key="cell.id"
                                    :class="{ 'w-px whitespace-nowrap': cell.column.id === 'actions' || cell.column.id === 'select' }">
                                    <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
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
    <!-- Create Dialog -->
    <CreateConfigFile v-if="openAddConfigFileDialog" :app="app" @created="onConfigFileCreated"
        @cancel="openAddConfigFileDialog = false" />

    <!-- Update Dialog -->
    <UpdateConfigFile v-if="showUpdateDialog && editingConfigFile" :app="app" :config-file="editingConfigFile"
        @updated="onConfigFileUpdated" @cancel="showUpdateDialog = false; editingConfigFile = null" />

</template>