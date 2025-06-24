<script setup lang="ts">
import { listClusters } from '@/api/cluster';

const clusters = listClusters({
    pageNo: 1,
    pageSize: 10,
    query: "",
    sortBy: "created_at",
    sortOrder: "DESC",
}) || [];
</script>


<template>
    <div class="flex flex-col gap-4 p-4 pt-0">
        <h1 class="text-xl font-semibold">集群列表</h1>
        <DataTable :columns="columns" :data="clusters" :loading="loading" :row-key="'id'" />
    </div>
</template>