<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { ref } from 'vue'
import { appSettingItems } from './data/settings'

const props = defineProps({
    currentTab: {
        type: String,
        default: appSettingItems[0].tab,
    },
})

const currentTab = ref(props.currentTab)

function switchAppSetting(tab: string) {
    currentTab.value = tab
    emit('switchAppSetting', tab)
}

const emit = defineEmits(['switchAppSetting'])

</script>

<template>
    <nav class="flex flex-wrap gap-2 lg:flex-col lg:gap-1">
        <Button v-for="item in appSettingItems" :key="item.title" variant="ghost" :class="cn(
            'text-full justify-start',
            currentTab === item.tab && 'bg-muted hover:bg-muted',
        )" @click="switchAppSetting(item.tab)">
            <component :is="item.icon" class="mr-2 h-4 w-4" />
            {{ item.title }}
        </Button>
    </nav>
</template>
