<script setup lang="ts">
import type { appModel } from '@/types/app';
import { ref } from 'vue';
import SidebarNav from './SidebarNav.vue';
import { appSettingItems } from './data/settings';

const props = defineProps({
    app: {
        type: Object as () => appModel,
        required: true,
    },
});

const currentTab = ref(appSettingItems[0].tab);

function onSwitchAppSetting(title: string) {
    currentTab.value = title;
}
</script>

<template>
    <div class="hidden space-y-2 py-4 pb-8 md:block">
        <!-- <div class="space-y-0.5">
            <h2 class="text-2xl font-bold tracking-tight">
                <Settings2 class="inline-block w-6 h-6 mr-2" />
                应用配置
            </h2>
        </div>
        <Separator class="my-6" /> -->
        <div class="flex flex-col space-y-8 lg:flex-row lg:space-x-12 lg:space-y-0">
            <aside>
                <SidebarNav :currentTab="currentTab" @switchAppSetting="onSwitchAppSetting" />
            </aside>
            <div class="w-full flex-1 overflow-auto">
                <div class="space-y-6 px-2">
                    <template v-for="(item, index) in appSettingItems" :key="index">
                        <component v-if="currentTab === item.tab" :is="item.comp" :app="props.app" />
                    </template>
                </div>
            </div>
        </div>
    </div>
</template>
