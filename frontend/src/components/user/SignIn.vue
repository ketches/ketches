<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import SignInForm from './SignInForm.vue'

const route = useRoute()

const redirectUrl = computed(() => {
  return route.query.redirectUrl?.toString()
})
</script>
<template>
  <div class="flex min-h-svh flex-col items-center justify-center bg-muted p-6 md:p-10">
    <div class="w-full max-w-sm md:max-w-3xl">
      <SignInForm :redirectUrl="redirectUrl" />
    </div>
  </div>
</template>