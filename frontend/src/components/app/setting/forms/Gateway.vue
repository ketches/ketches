<script setup lang="ts">
import { h, onMounted, ref, toRef, watch } from "vue";

import { deleteAppGateway, listAppGateways } from "@/api/app";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { appGatewayModel, appModel } from "@/types/app";
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
  ChevronsLeft,
  ChevronsRight,
  Delete,
  Plus,
  SquarePen,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import CreateGateway from "./CreateGateway.vue";
import UpdateGateway from "./UpdateGateway.vue";

const props = defineProps({
  app: {
    type: Object as () => appModel,
    required: true,
  },
});

const app = toRef(props, "app");

const listData = ref<appGatewayModel[]>([]);

async function fetchListData(appID?: string) {
  if (appID) {
    const data = await listAppGateways(appID);
    listData.value = data;
  }
}

onMounted(async () => {
  await fetchListData(app.value.appID);
});

watch(app, async (newApp) => {
  if (newApp.appID) {
    await fetchListData(newApp.appID);
  }
});

const centeredHeader = (text: string) =>
  h("div", { class: "text-center" }, text);

const columns: ColumnDef<appGatewayModel>[] = [
  {
    id: "select",
    header: ({ table }) =>
      h(Checkbox, {
        class: [
          table.getIsAllPageRowsSelected() || table.getIsSomePageRowsSelected()
            ? ""
            : "invisible group-hover:visible",
        ],
        modelValue: table.getIsAllPageRowsSelected()
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
    accessorKey: "port",
    header: "端口",
    cell: ({ row }) => h("div", {}, row.original.port),
  },
  {
    accessorKey: "protocol",
    header: "协议",
    cell: ({ row }) => h("div", {}, row.original.protocol),
  },
  {
    accessorKey: "accessAddress",
    header: "访问地址",
    cell: ({ row }) => {
      if (
        row.original.protocol === "http" ||
        row.original.protocol === "https"
      ) {
        return h(
          "div",
          { class: row.original.exposed ? "" : "text-muted-foreground" },
          `${
            row.original.domain
              ? row.original.domain + row.original?.path || ""
              : "-"
          }`,
        );
      } else if (
        row.original.protocol === "tcp" ||
        row.original.protocol === "udp"
      ) {
        return h(
          "div",
          { class: row.original.exposed ? "" : "text-muted-foreground" },
          `0.0.0.0:${row.original.gatewayPort}`,
        );
      }
    },
  },
  {
    accessorKey: "exposed",
    header: "对外访问",
    cell: ({ row }) =>
      h(Switch, {
        checked: row.original.exposed,
        onChange: async (checked: boolean) => {
          toast.success("网关对外访问状态更新成功" + checked);
        },
      }),
  },
  {
    id: "actions",
    header: () => centeredHeader("操作"),
    cell: ({ row }) => {
      return h("div", { class: "flex justify-end mr-2 gap-2" }, [
        h(
          Button,
          {
            variant: "outline",
            onClick: () => {
              selectedGateway.value = row.original;
              openUpdateGatewayDialog.value = true;
            },
          },
          () => [h(SquarePen)],
        ),
        h(
          Button,
          {
            variant: "outline",
            onClick: async () => {
              await deleteAppGateway(row.original.appID, [
                row.original.gatewayID,
              ]);
              listData.value = listData.value.filter(
                (item) => item.gatewayID !== row.original.gatewayID,
              );
              toast.success("网关删除成功");
            },
          },
          () => [h(Delete)],
        ),
      ]);
    },
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
  getRowId: (row) => row.gatewayID,
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

const openAddGatewayDialog = ref(false);
const openUpdateGatewayDialog = ref(false);
const selectedGateway = ref<appGatewayModel | null>(null);
</script>

<template>
  <div>
    <h3 class="text-lg font-medium">端口网关</h3>
    <p class="text-sm text-muted-foreground">配置应用程序的网络网关。</p>
  </div>
  <Separator />
  <div class="flex flex-col flex-grow">
    <div class="flex gap-2 items-center pb-4 w-full">
      <Input
        class="max-w-sm"
        placeholder="搜索网关"
        :model-value="
          table.getColumn('accessAddress')?.getFilterValue() as string
        "
        @update:model-value="
          table.getColumn('accessAddress')?.setFilterValue($event)
        "
      />
      <Button
        variant="outline"
        size="sm"
        class="ml-auto"
        @click="openAddGatewayDialog = true"
      >
        <Plus />
        新增
      </Button>
    </div>
    <div class="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow
            v-for="headerGroup in table.getHeaderGroups()"
            :key="headerGroup.id"
            class="group"
          >
            <TableHead
              v-for="header in headerGroup.headers"
              :key="header.id"
              :class="{
                'w-px whitespace-nowrap':
                  header.column.id === 'actions' ||
                  header.column.id === 'select',
              }"
            >
              <FlexRender
                v-if="!header.isPlaceholder"
                :render="header.column.columnDef.header"
                :props="header.getContext()"
              />
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="table.getRowModel().rows?.length">
            <template v-for="row in table.getRowModel().rows" :key="row.id">
              <TableRow
                :data-state="row.getIsSelected() && 'selected'"
                class="group"
              >
                <TableCell
                  v-for="cell in row.getVisibleCells()"
                  :key="cell.id"
                  :class="{
                    'w-px whitespace-nowrap':
                      cell.column.id === 'actions' ||
                      cell.column.id === 'select',
                  }"
                >
                  <FlexRender
                    :render="cell.column.columnDef.cell"
                    :props="cell.getContext()"
                  />
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
        <Button
          variant="outline"
          size="sm"
          :disabled="!table.getCanPreviousPage()"
          v-if="table.getCanPreviousPage()"
          @click="table.previousPage()"
        >
          <ChevronsLeft class="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="sm"
          :disabled="!table.getCanNextPage()"
          v-if="table.getCanNextPage()"
          @click="table.nextPage()"
        >
          <ChevronsRight class="h-4 w-4" />
        </Button>
      </div>
    </div>
  </div>
  <CreateGateway
    v-model="openAddGatewayDialog"
    :appID="app.appID"
    @close="openAddGatewayDialog = false"
    @gateway-created="fetchListData(app.appID)"
  />
  <UpdateGateway
    v-model="openUpdateGatewayDialog"
    v-if="selectedGateway"
    :gateway="selectedGateway"
    @close="openUpdateGatewayDialog = false"
    @gateway-updated="fetchListData(app.appID)"
  />
</template>
