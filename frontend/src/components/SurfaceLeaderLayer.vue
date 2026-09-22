<script setup lang="ts">
import type { SurfaceAnnotationLayout } from '../surfaceAnnotations'

defineProps<{
  labels: SurfaceAnnotationLayout[]
}>()
</script>

<template>
  <svg class="surface-leader-layer" aria-hidden="true">
    <line
      v-for="label in labels"
      v-show="label.visible"
      :key="label.id"
      :class="{ selected: label.selected, observer: label.variant === 'observer', inactive: label.inactive }"
      :x1="label.anchorX"
      :y1="label.anchorY"
      :x2="label.leaderX"
      :y2="label.leaderY"
    />
  </svg>
</template>

<style scoped>
.surface-leader-layer {
  position: absolute;
  inset: 0;
  z-index: 4;
  width: 100%;
  height: 100%;
  overflow: visible;
  pointer-events: none;
}
.surface-leader-layer line {
  stroke: var(--mission-accent-dim, rgba(114, 215, 255, .38));
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}
.surface-leader-layer line.selected {
  stroke: var(--mission-accent, #72d7ff);
  stroke-width: 1.25;
}
.surface-leader-layer line.observer {
  stroke: rgba(121, 227, 189, .48);
}
.surface-leader-layer line.observer.selected {
  stroke: #79e3bd;
}
.surface-leader-layer line.inactive {
  opacity: .34;
}
</style>
