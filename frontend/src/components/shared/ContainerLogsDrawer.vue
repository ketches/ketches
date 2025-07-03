<script setup lang="ts">
import { getViewAppInstanceLogsUrl } from '@/api/app';
import { Select, SelectItem } from "@/components/ui/select";
import type { logsRequestModel } from "@/types/app";
import { Container, SquareDashed, Undo2 } from 'lucide-vue-next';
import { computed, nextTick, ref, toRefs, watch } from "vue";
import { toast } from 'vue-sonner';
import Button from '../ui/button/Button.vue';
import Checkbox from '../ui/checkbox/Checkbox.vue';
import { Dialog } from '../ui/dialog';
import DialogClose from '../ui/dialog/DialogClose.vue';
import DialogContent from '../ui/dialog/DialogContent.vue';
import DialogDescription from '../ui/dialog/DialogDescription.vue';
import DialogHeader from '../ui/dialog/DialogHeader.vue';
import DialogTitle from '../ui/dialog/DialogTitle.vue';
import { ScrollArea } from '../ui/scroll-area';
import SelectContent from '../ui/select/SelectContent.vue';
import SelectGroup from '../ui/select/SelectGroup.vue';
import SelectLabel from '../ui/select/SelectLabel.vue';
import SelectTrigger from '../ui/select/SelectTrigger.vue';
import SelectValue from '../ui/select/SelectValue.vue';

const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false,
    },
    appID: {
        type: String,
        required: true,
    },
    instanceName: {
        type: String,
        required: true,
    },
    containers: {
        type: Array as () => string[],
        default: () => [],
    },
    initContainers: {
        type: Array as () => string[],
        default: () => [],
    },
    showOptions: {
        type: Boolean,
        default: true,
    },
})

const emit = defineEmits(['update:modelValue']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const logsContent = ref("");

const containerName = ref<string>(props.containers?.[0] ?? "");
const logsRequest = ref<logsRequestModel>({
    follow: true,
    tailLines: 100,
    showTimestamps: false,
});

const { follow, tailLines, showTimestamps } = toRefs(logsRequest.value);

watch(open, async (isOpen) => {
    if (isOpen) {
        fetchAppInstanceLogs();
    } else {
        if (es) {
            es.close();
            es = null;
        }
    }
});

// Refetch logs when logsRequest or containerName changes and dialog is open
watch([logsRequest, containerName], () => {
    if (open.value) {
        fetchAppInstanceLogs();
    }
}, { deep: true });

let es: EventSource | null = null;

function fetchAppInstanceLogs() {
    logsContent.value = ""; // Clear previous logs
    if (!containerName.value) {
        toast.error("请选择容器", {
            description: "请先选择要查看日志的容器。",
        });
        return;
    }
    const logsUrl = getViewAppInstanceLogsUrl(
        props.appID,
        props.instanceName,
        containerName.value,
        logsRequest.value
    );

    if (es) {
        es.close();
        es = null;
    }

    es = new EventSource(logsUrl, { withCredentials: true });
    es.onmessage = (event) => {
        logsContent.value += event.data + "\n";
        scrollToBottom();
    };
    es.onerror = (error) => {
        if (es) {
            es.close();
            es = null;
        }
        toast.error("日志获取失败", {
            description: "无法获取实例日志，请稍后重试。",
        });
    };
}

const logsScrollRef = ref<HTMLElement | null>(null);

function scrollToBottom() {
    nextTick(() => {
        if (logsScrollRef.value) {
            logsScrollRef.value.scrollTop = logsScrollRef.value.scrollHeight;
        }
    });
}

watch(logsContent, () => {
    scrollToBottom();
}, { deep: true });

watch(open, (isOpen) => {
    if (isOpen) {
        nextTick(() => {
            scrollToBottom();
        });
    }
});
</script>

<template>
    <Dialog :open="open" @update:open="open = $event" close-on-esc>
        <DialogContent
            class="sm:min-w-full w-screen max-w-none max-h-none rounded-none flex flex-col h-full p-0 pt-2 gap-2"
            @pointer-down-outside="(event) => {
                const originalEvent = event.detail.originalEvent;
                const target = originalEvent.target as HTMLElement;
                if (originalEvent.offsetX > target.clientWidth || originalEvent.offsetY > target.clientHeight) {
                    event.preventDefault();
                }
            }">
            <DialogHeader>
                <DialogTitle>
                    <div class="flex items-center w-full gap-2 flex-nowrap whitespace-nowrap relative">
                        <div class="flex items-center gap-2 whitespace-nowrap">
                            <DialogClose as-child>
                                <Button variant="link" class="p-2">
                                    <Undo2 class="w-4 h-4" />
                                    <span>返回</span>
                                </Button>
                            </DialogClose>
                            <span class="text-lg font-semibold text-primary">应用实例日志</span>
                            <span class="text-sm font-light ml-2 font-mono bg-secondary px-2 rounded">
                                {{ props.instanceName }}
                            </span>
                        </div>
                        <div
                            class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 flex justify-center items-center gap-2 text-sm whitespace-nowrap font-normal">
                            <label
                                class="text-sm leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                                容器
                            </label>
                            <Select v-model="containerName" :default-value="containers?.[0]"
                                class="min-w-[140px] flex items-center gap-2">
                                <SelectTrigger class="w-fit">
                                    <SelectValue placeholder="选择容器" />
                                </SelectTrigger>
                                <SelectContent>
                                    <SelectGroup>
                                        <SelectLabel v-if="initContainers.length > 0">应用容器</SelectLabel>
                                        <SelectItem v-for="container in containers" :key="container" :value="container">
                                            <Container />
                                            {{ container }}
                                        </SelectItem>
                                    </SelectGroup>
                                    <SelectGroup v-if="initContainers.length > 0">
                                        <SelectLabel>初始化容器</SelectLabel>
                                        <SelectItem v-for="container in initContainers" :key="container"
                                            :value="container">
                                            <SquareDashed />
                                            {{ container }}
                                        </SelectItem>
                                    </SelectGroup>
                                </SelectContent>
                            </Select>
                        </div>
                        <div v-if="showOptions"
                            class="absolute right-0 top-1/2 -translate-y-1/2 flex gap-4 items-center font-normal text-sm whitespace-nowrap pr-16">
                            <div class="flex items-center gap-2">
                                <label
                                    class="text-sm leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">初始行</label>
                                <Select v-model="tailLines">
                                    <SelectTrigger class="w-fit">
                                        <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                        <SelectGroup>
                                            <SelectItem v-for="n in [100, 200, 500, 1000]" :key="n" :value="n">
                                                {{ n }}
                                            </SelectItem>
                                        </SelectGroup>
                                    </SelectContent>
                                </Select>
                            </div>
                            <div class="flex items-center gap-2">
                                <Checkbox v-model="follow" />
                                <label
                                    class="text-sm leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                                    保持刷新
                                </label>
                            </div>
                            <div class="flex items-center gap-2">
                                <Checkbox v-model="showTimestamps" />
                                <label
                                    class="text-sm leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                                    显示时间戳
                                </label>
                            </div>
                        </div>
                    </div>
                </DialogTitle>
                <DialogDescription v-show="false">
                </DialogDescription>
            </DialogHeader>
            <ScrollArea class="w-full h-full flex-1 min-h-0 flex flex-col text-gray-200 bg-gray-950 text-sm p-2">
                <pre class="whitespace-pre-wrap" ref="logsScrollRef">{{ logsContent }}</pre>
            </ScrollArea>
        </DialogContent>
    </Dialog>
</template>
