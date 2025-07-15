<script setup lang="ts">
import { checkClusterExtensionFeatureEnabled, enableClusterExtension, getClusterExtensionValues, installClusterExtension, listClusterExtensions, uninstallClusterExtension } from "@/api/cluster";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import type { clusterExtensionModel, clusterModel, installClusterExtensionModel } from "@/types/cluster";
import { Blocks, Download, Package, Plus, RefreshCcw, Search, Trash2 } from "lucide-vue-next";
import { computed, onMounted, ref, watch } from "vue";
import { toast } from "vue-sonner";
import ConfirmDialog from "../shared/ConfirmDialog.vue";

const props = defineProps<{
    cluster: clusterModel | null;
}>();

const loading = ref(false);
const featureEnabled = ref(false);
const extensions = ref<Record<string, clusterExtensionModel>>({});
const installDialogOpen = ref(false);
const selectedExtension = ref<clusterExtensionModel | null>(null);
const installForm = ref<installClusterExtensionModel>({
    extensionName: '',
    type: 'helm',
    version: '',
    namespace: 'ketches',
    createNamespace: true,
    values: ''
});
const searchQuery = ref('');

const filteredExtensionList = computed(() => {
    const list = Object.values(extensions.value);
    const sortedList = list.sort((a, b) => {
        if (a.installed && !b.installed) return -1;
        if (!a.installed && b.installed) return 1;
        return a.displayName.localeCompare(b.displayName);
    });

    if (!searchQuery.value) {
        return sortedList;
    }

    return sortedList.filter(extension =>
        extension.displayName.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
        (extension.description && extension.description.toLowerCase().includes(searchQuery.value.toLowerCase()))
    );
});

async function onEnableClusterExtension() {
    if (!props.cluster?.clusterID) return;

    try {
        loading.value = true;
        await enableClusterExtension(props.cluster.clusterID);
        featureEnabled.value = true;
        toast.success('集群扩展功能已成功启用');
        await fetchExtensions();
    } catch (error) {
        console.error('启用集群扩展功能失败:', error);
        toast.error('启用集群扩展功能失败');
    } finally {
        loading.value = false;
    }
}

async function checkFeatureEnabled() {
    if (!props.cluster?.clusterID) return;

    try {
        loading.value = true;
        featureEnabled.value = await checkClusterExtensionFeatureEnabled(props.cluster.clusterID);
    } catch (error) {
        console.error('检查扩展功能失败:', error);
        toast.error('检查扩展功能状态失败');
    } finally {
        loading.value = false;
    }
}

async function fetchExtensions() {
    if (!props.cluster?.clusterID || !featureEnabled.value) return;

    try {
        loading.value = true;
        extensions.value = await listClusterExtensions(props.cluster.clusterID);
    } catch (error) {
        console.error('获取扩展列表失败:', error);
        toast.error('获取扩展列表失败');
    } finally {
        loading.value = false;
    }
}

function openInstallDialog(extension: clusterExtensionModel) {
    selectedExtension.value = extension;
    const latestVersion = extension.versions && extension.versions.length > 0 ? extension.versions[0] : '';
    installForm.value = {
        extensionName: extension.slug,
        type: 'helm',
        version: latestVersion,
        namespace: 'ketches',
        createNamespace: true,
        values: ''
    };
    installDialogOpen.value = true;

    // Load default values for the latest version
    if (latestVersion && props.cluster?.clusterID) {
        loadDefaultValues(extension.slug, latestVersion);
    }
}

async function handleInstall() {
    if (!props.cluster?.clusterID || !selectedExtension.value) return;

    try {
        loading.value = true;
        await installClusterExtension(props.cluster.clusterID, installForm.value);
        toast.success('扩展安装成功');
        installDialogOpen.value = false;
        await fetchExtensions();
    } catch (error) {
        console.error('安装扩展失败:', error);
        toast.error('安装扩展失败');
    } finally {
        loading.value = false;
    }
}

async function handleUninstall(extensionName: string) {
    if (!props.cluster?.clusterID) return;

    if (!confirm('您确定要卸载此扩展吗？')) {
        return;
    }

    try {
        loading.value = true;
        await uninstallClusterExtension(props.cluster.clusterID, extensionName);
        toast.success('扩展卸载成功');
        await fetchExtensions();
    } catch (error) {
        console.error('卸载扩展失败:', error);
        toast.error('卸载扩展失败');
    } finally {
        loading.value = false;
    }
}

async function loadDefaultValues(extensionName: string, version: string) {
    if (!props.cluster?.clusterID || !version) return;

    try {
        const values = await getClusterExtensionValues(props.cluster.clusterID, extensionName, version);
        if (values) {
            installForm.value.values = values;
        }
    } catch (error) {
        console.error('获取默认配置失败:', error);
        // Don't show error toast as this is optional
    }
}

// Watch for version changes to load corresponding default values
watch(() => installForm.value.version, (newVersion) => {
    if (newVersion && selectedExtension.value && props.cluster?.clusterID) {
        loadDefaultValues(selectedExtension.value.slug, newVersion);
    }
});

onMounted(async () => {
    await checkFeatureEnabled();
    if (featureEnabled.value) {
        await fetchExtensions();
    }
});

const showEnableClusterExtensionDialog = ref(false);
</script>

<template>
    <div v-if="!loading && !featureEnabled" class="flex flex-1 flex-col gap-4 p-4 pt-0 h-full">
        <div
            class="flex flex-col flex-grow text-balance text-center text-sm text-muted-foreground justify-center items-center">
            <span class="block mb-2">集群启用扩展功能，会在集群中创建 <a
                    href="https://github.com/ketches/helm-operator">helm-operator</a>，并自动添加 <a
                    href="https://github.com/ketches/ketches-extension-charts">ketches-extension-charts</a>
                Helm 仓库，基于该仓库中的应用实现集群的扩展能力。</span>
            <Button variant="default" class="my-4" @click="showEnableClusterExtensionDialog = true">
                <Plus />
                启用集群扩展
            </Button>
            <ConfirmDialog v-if="showEnableClusterExtensionDialog" title="确定启用集群扩展功能吗"
                description="您确定要启用集群扩展功能吗？此操作将在集群 ketches 命名空间中创建额外资源。" @confirm="
                    onEnableClusterExtension()
                    " @cancel="showEnableClusterExtensionDialog = false" />
        </div>
    </div>
    <div v-else class="space-y-4">
        <!-- Loading state -->
        <div v-if="loading" class="space-y-4">
            <Skeleton class="h-12 w-full" />
            <Skeleton class="h-32 w-full" />
            <Skeleton class="h-32 w-full" />
        </div>

        <!-- Extensions list -->
        <div v-else-if="featureEnabled">
            <div class="flex items-center justify-between mb-4">
                <div class="relative w-full max-w-sm">
                    <Input v-model="searchQuery" placeholder="搜索扩展..." class="pl-10" />
                    <div class="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none">
                        <Search class="h-4 w-4 text-muted-foreground" />
                    </div>
                </div>
                <Button @click="fetchExtensions" variant="outline" size="sm">
                    <RefreshCcw />
                    刷新
                </Button>
            </div>

            <div v-if="filteredExtensionList.length === 0" class="text-center py-8">
                <Card>
                    <CardContent class="pt-6">
                        <Package class="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                        <h3 class="text-lg font-semibold mb-2">无可用扩展</h3>
                        <p class="text-muted-foreground">此集群无可用扩展</p>
                    </CardContent>
                </Card>
            </div>
            <div v-else class="space-y-4">
                <Card v-for="extension in filteredExtensionList" :key="extension.extensionID" class="py-4">
                    <CardContent class="px-4 relative">
                        <div class="flex items-center gap-4">
                            <div class="flex-shrink-0">
                                <Blocks class="w-10 h-10 text-muted-foreground" />
                            </div>
                            <div class="flex-grow space-y-1">
                                <div class="flex items-center justify-start gap-2">
                                    <h3 class="font-semibold">{{ extension.displayName }}</h3>
                                    <Badge v-if="extension.status === 'installed'"
                                        class="bg-green-500/20 text-green-500">
                                        已安装
                                    </Badge>
                                    <Badge v-else-if="extension.status === 'installing'"
                                        class="bg-yellow-500/20 text-yellow-500">
                                        安装中
                                    </Badge>
                                    <Badge v-else-if="extension.status === 'failed'" class="bg-red-500/20 text-red-500">
                                        失败
                                    </Badge>
                                    <Badge v-else class=" bg-amber-500/20 text-amber-500">
                                        可用
                                    </Badge>
                                </div>
                                <div class="text-xs text-muted-foreground">
                                    版本: {{ extension.version || '最新' }} | 安装方式: {{ extension.installMethod || 'helm' }}
                                </div>
                                <p class="text-sm text-muted-foreground">
                                    {{ extension.description || '暂无描述' }}
                                </p>
                            </div>
                        </div>
                        <div class="absolute top-0 right-4 flex gap-2">
                            <Button v-if="!extension.installed" @click="openInstallDialog(extension)" size="sm"
                                variant="default" :disabled="loading">
                                <Download class="h-4 w-4 mr-2" />
                                安装
                            </Button>
                            <template v-else>
                                <Button @click="handleUninstall(extension.slug)" size="sm" variant="destructive"
                                    :disabled="loading">
                                    <Trash2 class="h-4 w-4 mr-2" />
                                    卸载
                                </Button>
                            </template>
                        </div>
                    </CardContent>
                </Card>
            </div>
        </div>

        <!-- Install dialog -->
        <Dialog v-model:open="installDialogOpen">
            <DialogContent class="sm:max-w-[800px]">
                <DialogHeader>
                    <DialogTitle>安装扩展</DialogTitle>
                    <DialogDescription>
                        为 {{ selectedExtension?.displayName }} 配置安装参数
                    </DialogDescription>
                </DialogHeader>
                <div class="grid gap-4 py-4">
                    <div class="grid grid-cols-8 items-center gap-4">
                        <Label for="extension-name" class="text-right">
                            扩展名称
                        </Label>
                        <Input id="extension-name" v-model="installForm.extensionName" class="col-span-7" readonly />
                    </div>
                    <div class="grid grid-cols-8 items-center gap-4">
                        <Label for="version" class="text-right">
                            版本
                        </Label>
                        <Select v-model="installForm.version">
                            <SelectTrigger class="col-span-7">
                                <SelectValue placeholder="选择版本" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem v-for="version in selectedExtension?.versions" :key="version"
                                    :value="version">
                                    {{ version }}
                                </SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                    <div class="grid grid-cols-8 items-center gap-4">
                        <Label for="namespace" class="text-right">
                            命名空间
                        </Label>
                        <Input id="namespace" v-model="installForm.namespace" class="col-span-7" readonly />
                    </div>
                    <div class="grid grid-cols-8 items-center gap-4">
                        <Label for="values" class="text-right pt-0">
                            Values
                        </Label>
                        <Textarea id="values" v-model="installForm.values" placeholder="YAML 格式的 Helm values (可选)"
                            class="col-span-7 min-h-32 max-h-64 font-mono" />
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" @click="installDialogOpen = false">
                        取消
                    </Button>
                    <Button @click="handleInstall" :disabled="loading">
                        {{ loading ? '安装中...' : '安装' }}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    </div>
</template>
