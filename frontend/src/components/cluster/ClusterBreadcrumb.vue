<script lang="ts" setup>
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
} from "@/components/ui/breadcrumb";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Boxes, ChevronDown } from "lucide-vue-next";
import { ref } from "vue";
import Button from "../ui/button/Button.vue";
const envHover = ref(false);
</script>

<template>
  <Breadcrumb>
    <BreadcrumbList>
      <BreadcrumbItem @mouseenter="envHover = true" @mouseleave="envHover = false">
        <DropdownMenu>
          <DropdownMenuTrigger class="flex items-center gap-1">
            <Button variant="ghost">
              <Boxes class="h-4 w-4" />
              阿里云集群
              <ChevronDown v-if="envHover" class="h-4 w-4" />
              <!-- <ChevronRight v-else class="h-4 w-4" /> -->
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuItem :disabled="true">
              <Boxes />
              阿里云环境
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Boxes />
              <RouterLink to="/console/cluster/tencentcloud">腾讯云集群</RouterLink>
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Boxes />
              <RouterLink to="/console/cluster/huaweicloud">华为云集群</RouterLink>
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Boxes />
              <RouterLink to="/console/cluster/local">本地集群</RouterLink>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </BreadcrumbItem>
    </BreadcrumbList>
  </Breadcrumb>
</template>
