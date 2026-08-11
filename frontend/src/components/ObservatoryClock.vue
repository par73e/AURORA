<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps<{ timezone: string }>()

const current = ref(new Date())
let clock: number | undefined

const label = computed(() => new Intl.DateTimeFormat('zh-CN', {
  timeZone: props.timezone,
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
}).format(current.value))

onMounted(() => {
  clock = window.setInterval(() => {
    current.value = new Date()
  }, 1000)
})

onBeforeUnmount(() => {
  if (clock !== undefined) window.clearInterval(clock)
})
</script>

<template>
  <time>{{ label }} <small>LOCAL</small></time>
</template>
