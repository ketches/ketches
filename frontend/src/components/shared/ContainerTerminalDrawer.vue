<script setup lang="ts">
import { getExecAppInstanceTerminalUrl } from "@/api/app";
import { Select, SelectItem } from "@/components/ui/select";
import { Box, Plug, SquareDashed, Undo2 } from 'lucide-vue-next';
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import { toast } from "vue-sonner";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";
import Button from '../ui/button/Button.vue';
import { Dialog } from '../ui/dialog';
import DialogClose from '../ui/dialog/DialogClose.vue';
import DialogContent from '../ui/dialog/DialogContent.vue';
import DialogDescription from '../ui/dialog/DialogDescription.vue';
import DialogHeader from '../ui/dialog/DialogHeader.vue';
import DialogTitle from '../ui/dialog/DialogTitle.vue';
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
})

const emit = defineEmits(['update:modelValue']);

const open = computed({
    get: () => props.modelValue,
    set: (value: boolean) => {
        emit('update:modelValue', value);
    }
});

const containerName = ref<string>(props.containers?.[0] ?? "");

const terminalRef = ref<HTMLElement | null>(null);
let terminal: Terminal | null = null;
let fitAddon: FitAddon | null = null;

let ws: WebSocket | null = null;

function initTerminal() {
    if (terminalRef.value) {
        const style = getComputedStyle(terminalRef.value);
        terminal = new Terminal({
            fontSize: style.getPropertyPriority('--font-size') === 'important' ? parseInt(style.getPropertyValue('--font-size').trim()) : 14,
            theme: {
                background: style.getPropertyValue('--color-gray-950').trim() || "#0a0a0a",
                foreground: style.getPropertyValue('--color-gray-200').trim() || "#e5e7eb",
            },
            fontFamily: 'monospace',
            cursorBlink: true,
            disableStdin: false,
        });
        fitAddon = new FitAddon();
        terminal.loadAddon(fitAddon);
        terminal.open(terminalRef.value);
        fitAddon.fit();
        terminal.focus();

        const wsUrl = getExecAppInstanceTerminalUrl(
            props.appID,
            props.instanceName,
            containerName.value || ""
        ).replace(/^http/, "ws");
        ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            if (terminal) {
                // terminal.writeln('连接到终端成功');
                toast.success('终端连接成功', {
                    description: '您可以开始输入命令。',
                    duration: 1000,
                });
            }
        };
        ws.onmessage = (event) => {
            if (terminal) {
                terminal.write(event.data);
            }
        };
        ws.onerror = () => {
            if (terminal) {
                terminal.writeln('\r\n终端连接错误');
                toast.error('终端连接错误，请重试', {
                    description: '请检查网络连接或容器状态。',
                });
            }
        };
        ws.onclose = () => {
            if (terminal) {
                terminal.writeln('\r\n终端连接已关闭');
                toast.info('终端连接已关闭', {
                    description: '请检查容器状态或重新连接。',
                });
            }
        };

        // 终端输入转发到 ws
        terminal.onData((data) => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(data);
            }
        });

        window.addEventListener('resize', handleResize);
    }
}

function disposeTerminal() {
    if (ws) {
        ws.close();
        ws = null;
    }
    if (terminal) {
        terminal.dispose();
        terminal = null;
    }
    fitAddon = null;

    window.removeEventListener('resize', handleResize);
}

function reconnectTerminal() {
    disposeTerminal();
    nextTick(() => {
        initTerminal();
    });
}

watch(open, (isOpen) => {
    if (isOpen) {
        nextTick(() => {
            initTerminal();
        });
    } else {
        disposeTerminal();
    }
});

watch(containerName, (val) => {
    if (terminal) {
        terminal.clear();
        terminal.writeln('当前容器: ' + (val || ''));
    }
});

function handleResize() {
    if (fitAddon) fitAddon.fit();
}

onBeforeUnmount(() => {
    disposeTerminal();
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
                            <span class="text-lg font-semibold text-primary">应用实例终端</span>
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
                                            <Box />
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
                        <div
                            class="absolute right-0 top-1/2 -translate-y-1/2 flex gap-4 items-center font-normal text-sm whitespace-nowrap pr-16">
                            <Button variant="outline" size="sm" class="p-2" @click="reconnectTerminal">
                                <Plug class="w-4 h-4" />
                                <span>重连</span>
                            </Button>
                        </div>
                    </div>
                </DialogTitle>
                <DialogDescription v-show="false">
                </DialogDescription>
            </DialogHeader>
            <!-- <ScrollArea class="w-full h-full text-gray-200 bg-gray-950 text-sm"> -->
            <div ref="terminalRef"
                class="w-full h-full flex-1 min-h-0 flex flex-col text-gray-200 bg-gray-950 text-sm pl-2 py-2 pr-0">
            </div>
            <!-- </ScrollArea> -->
        </DialogContent>
    </Dialog>
</template>
