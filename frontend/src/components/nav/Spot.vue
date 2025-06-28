<script setup lang="ts">
import { Icon } from "@iconify/vue";
import { useMagicKeys, whenever } from "@vueuse/core";
import { Search } from "lucide-vue-next";
import {
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxLabel,
  ComboboxRoot,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  DialogTrigger,
  VisuallyHidden,
} from "reka-ui";
import { ref } from "vue";
import { commandMenuItems } from "../data/spotList.ts";
import ScrollArea from "../ui/scroll-area/ScrollArea.vue";
import SidebarMenuButton from "../ui/sidebar/SidebarMenuButton.vue";

const open = ref(false);

const { meta_j, ctrl_j } = useMagicKeys();
whenever([meta_j, ctrl_j], (n) => {
  if (n[0] || n[1]) {
    open.value = true;
  }
});

function handleSelect(ev: CustomEvent) {
  ev.preventDefault();
  open.value = false;
  // eslint-disable-next-line no-console
  console.log("Selected: ", ev.detail.value);
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogTrigger class="w-full">
      <SidebarMenuButton class="dark:text-white border flex items-center justify-between text-muted-foreground">
        <Search />
        <span class="justify-center flex-1">聚焦</span>
        <span
          class="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[12px] font-medium text-muted-foreground opacity-100"><kbd><span
              class="text-sm">⌘</span> J</kbd></span>
      </SidebarMenuButton>
    </DialogTrigger>

    <DialogPortal>
      <DialogOverlay class="bg-background/80 fixed inset-0 z-30" />
      <DialogContent
        class="fixed top-[15%] left-[50%] max-h-[85vh] w-[90vw] max-w-[24rem] translate-x-[-50%] text-sm rounded-xl overflow-hidden border border-muted-foreground/30 bg-card focus:outline-none z-[100]">
        <VisuallyHidden>
          <DialogTitle>Command Menu</DialogTitle>
          <DialogDescription>Search for command</DialogDescription>
        </VisuallyHidden>

        <ComboboxRoot :open="true">
          <ComboboxInput placeholder="输入搜索 ..."
            class="bg-transparent w-full px-4 py-3 outline-none placeholder-muted-foreground" @keydown.enter.prevent />

          <ComboboxContent class="border-t border-muted-foreground/30 p-2 max-h-[20rem] overflow-y-auto"
            @escape-key-down="open = false">
            <ComboboxEmpty class="text-center text-muted-foreground p-4">
              No results
            </ComboboxEmpty>
            <ScrollArea>
              <ComboboxGroup v-for="group in commandMenuItems" :key="group.group">
                <ComboboxLabel
                  class="px-4 inline-flex w-full items-center gap-4 text-muted-foreground font-semibold mt-3 mb-3">
                  <Icon :icon="group.icon" />
                  {{ group.group }}
                </ComboboxLabel>
                <ComboboxItem v-for="item in group.items" :key="item.id" :value="item"
                  class="cursor-default pl-12 py-2 rounded-md data-[highlighted]:bg-muted inline-flex w-full items-center gap-4"
                  @select="handleSelect">
                  <span>{{ item.name }}</span>
                </ComboboxItem>
              </ComboboxGroup>
            </ScrollArea>
          </ComboboxContent>
        </ComboboxRoot>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>

</template>
