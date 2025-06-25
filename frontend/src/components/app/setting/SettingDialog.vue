<script setup lang="ts">
import { getApp } from '@/api/app'
import { Dialog } from '@/components/ui/dialog'
import DialogContent from '@/components/ui/dialog/DialogContent.vue'
import DialogDescription from '@/components/ui/dialog/DialogDescription.vue'
import DialogHeader from '@/components/ui/dialog/DialogHeader.vue'
import DialogTitle from '@/components/ui/dialog/DialogTitle.vue'
import { useResourceRefStore } from '@/stores/resourceRefStore'
import type { appModel } from '@/types/app'
import { storeToRefs } from 'pinia'
import { computed, ref, watch } from 'vue'
import { toast } from 'vue-sonner'
import Settings from './Settings.vue'

const { activeAppRef } = storeToRefs(useResourceRefStore())

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    }
})

const emit = defineEmits(['update:modelValue']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const app = ref<appModel | null>(null);

watch(open, async (isOpen) => {
    if (isOpen) {
        if (!activeAppRef.value) {
            toast.error('获取应用信息失败', {
                description: '请重新加载应用信息后再尝试打开设置对话框。'
            });
            return;
        }
        app.value = await getApp(activeAppRef.value.appID)
    }
});
</script>

<template>
    <Dialog :open="open" @update:open="open = $event" close-on-esc>
        <DialogContent class="min-w-8/12 min-h-11/12">
            <DialogHeader v-show="false">
                <DialogTitle>
                </DialogTitle>
                <DialogDescription>
                </DialogDescription>
            </DialogHeader>
            <Settings v-if="app" :app="app" />
        </DialogContent>
    </Dialog>
</template>
