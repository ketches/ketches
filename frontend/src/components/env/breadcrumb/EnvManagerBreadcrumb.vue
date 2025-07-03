<script lang="ts" setup>
import {
    Breadcrumb,
    BreadcrumbItem,
    BreadcrumbList,
} from '@/components/ui/breadcrumb';
import EnvBreadcrumb from './EnvBreadcrumb.vue';

defineProps({
    envID: {
        type: String,
        default: '',
    },
});
</script>

<template>
    <Breadcrumb>
        <BreadcrumbList>
            <EnvBreadcrumb v-if="envID" :envID="envID" />
            <BreadcrumbItem v-else class="font-semibold"> 环境 </BreadcrumbItem>
        </BreadcrumbList>
    </Breadcrumb>
</template>