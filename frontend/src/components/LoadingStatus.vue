<script setup lang="ts">
withDefaults(defineProps<{
  label: string
  variant?: 'inline' | 'panel' | 'stage'
}>(), { variant: 'inline' })
</script>

<template>
  <div class="loading-status" :class="`loading-status--${variant}`">
    <div class="loading-status__signal" role="status">
      <span class="loading-status__mark" aria-hidden="true"><i /></span>
      <span>{{ label }}</span>
    </div>
    <div v-if="$slots.preview" class="loading-status__preview" aria-hidden="true">
      <slot name="preview" />
    </div>
  </div>
</template>

<style scoped>
.loading-status {
  --signal-color: var(--loading-accent, var(--blue));
  color: var(--loading-ink, var(--ink));
  font-size: .8125rem;
  font-weight: 500;
  line-height: 1.5;
}
.loading-status__signal {
  display: flex;
  align-items: center;
  gap: .7rem;
  min-width: 0;
}
.loading-status__mark {
  position: relative;
  display: grid;
  place-items: center;
  flex: none;
  width: 1.75rem;
  aspect-ratio: 1;
  color: var(--signal-color);
}
.loading-status__mark::before {
  content: '';
  width: .22rem;
  aspect-ratio: 1;
  border-radius: 50%;
  background: currentColor;
  opacity: .56;
}
.loading-status__mark i {
  position: absolute;
  inset: 8%;
  border: 1px solid color-mix(in srgb, currentColor 48%, transparent);
  border-radius: 50%;
  transform: rotate(-32deg) scaleY(.52);
}
.loading-status__mark i::after {
  content: '';
  position: absolute;
  inset: -25%;
  background: radial-gradient(circle .16rem at 50% 10%, currentColor 87%, transparent);
  animation: loading-orbit 2.5s .25s linear infinite;
}
@keyframes loading-orbit {
  to { transform: rotate(1turn); }
}
.loading-status--panel {
  padding: 1rem 0 1.15rem;
  border-block: 1px solid var(--loading-line, var(--line));
}
.loading-status--panel .loading-status__preview { margin-top: .95rem; }
.loading-status--stage {
  display: grid;
  place-items: center;
  min-height: 100dvh;
  padding: 5rem 1.5rem 2rem;
  font-size: 1rem;
  background: radial-gradient(ellipse at center, var(--surface) 0%, var(--bg) 65%);
}
.loading-status--stage .loading-status__signal {
  flex-direction: column;
  gap: 1.05rem;
  text-align: center;
}
.loading-status--stage .loading-status__mark { width: 3.75rem; }
@media (prefers-reduced-motion: reduce) {
  .loading-status__mark i::after { animation: none; }
}
</style>
