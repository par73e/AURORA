<script setup lang="ts">
defineProps<{
  kind: 'spacecraft' | 'surface'
  nameZh: string
  nameEn?: string
  selected?: boolean
  iconHtml?: string
  compact?: boolean
}>()
</script>

<template>
  <button class="mission-scene-label" :class="[`is-${kind}`, { selected, 'is-compact': compact }]" type="button">
    <span v-if="iconHtml" class="mission-scene-label-icon" aria-hidden="true" v-html="iconHtml" />
    <i v-else class="mission-scene-label-dot" aria-hidden="true" />
    <span class="mission-scene-label-copy">
      <strong>{{ nameZh }}</strong>
      <small v-if="nameEn && nameEn !== nameZh">（{{ nameEn }}）</small>
    </span>
  </button>
</template>

<style scoped>
.mission-scene-label {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 5;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 30px;
  padding: 5px 8px;
  border: 1px solid var(--mission-line, rgba(139, 180, 202, .22));
  border-radius: 4px;
  background: var(--mission-label-surface, rgba(3, 10, 17, .78));
  color: var(--mission-text, #ecf5f9);
  font: inherit;
  text-align: left;
  cursor: pointer;
  backdrop-filter: blur(8px);
  transition: border-color .2s, background .2s, color .2s, opacity .3s;
}
.mission-scene-label::before {
  content: '';
  position: absolute;
  right: 100%;
  top: 50%;
  width: 10px;
  height: 1px;
  background: var(--mission-accent-dim, rgba(114, 215, 255, .38));
}
.mission-scene-label:hover,
.mission-scene-label:focus-visible,
.mission-scene-label.selected {
  border-color: var(--mission-accent, #72d7ff);
  background: var(--mission-label-surface-active, rgba(6, 17, 26, .9));
}
.mission-scene-label:focus-visible {
  outline: 1px solid var(--mission-accent, #72d7ff);
  outline-offset: 3px;
}
.mission-scene-label-dot {
  width: 5px;
  height: 5px;
  flex: 0 0 5px;
  border-radius: 50%;
  background: var(--mission-accent, #72d7ff);
  box-shadow: 0 0 8px color-mix(in srgb, var(--mission-accent, #72d7ff) 68%, transparent);
}
.mission-scene-label-icon {
  display: inline-flex;
  flex: 0 0 auto;
  color: var(--mission-accent, #72d7ff);
}
.mission-scene-label-icon :deep(svg) { width: 12px; height: 12px; }
.mission-scene-label-copy {
  display: grid;
  gap: 2px;
  min-width: 0;
}
.mission-scene-label strong {
  color: inherit;
  font-size: 10px;
  font-weight: 500;
  line-height: 1.2;
  letter-spacing: .04em;
  white-space: nowrap;
}
.mission-scene-label small {
  color: var(--mission-quiet, #7f98a7);
  font: 400 8px/1.3 var(--font-mono);
  letter-spacing: .08em;
  white-space: nowrap;
}
.mission-scene-label.is-compact {
  gap: 4px;
  min-height: 18px;
  max-width: 112px;
  padding: 2px 4px;
  border-color: transparent;
  background: transparent;
  backdrop-filter: none;
}
.mission-scene-label.is-compact::before { width: 6px; opacity: .72; }
.mission-scene-label.is-compact .mission-scene-label-dot {
  width: 4px;
  height: 4px;
  flex-basis: 4px;
  box-shadow: none;
}
.mission-scene-label.is-compact strong {
  overflow: hidden;
  max-width: 88px;
  font-size: 8px;
  font-weight: 500;
  letter-spacing: .06em;
  text-overflow: ellipsis;
}
.mission-scene-label.is-compact small { display: none; }
.mission-scene-label.is-compact:hover,
.mission-scene-label.is-compact:focus-visible,
.mission-scene-label.is-compact.selected {
  gap: 7px;
  max-width: none;
  min-height: 28px;
  padding: 4px 7px;
  border-color: var(--mission-accent, #72d7ff);
  background: var(--mission-label-surface-active, rgba(6, 17, 26, .9));
  backdrop-filter: blur(8px);
}
.mission-scene-label.is-compact:hover strong,
.mission-scene-label.is-compact:focus-visible strong,
.mission-scene-label.is-compact.selected strong {
  overflow: visible;
  max-width: none;
  font-size: 10px;
  text-overflow: clip;
}
.mission-scene-label.is-compact:hover small,
.mission-scene-label.is-compact:focus-visible small,
.mission-scene-label.is-compact.selected small { display: block; }
@media (prefers-reduced-motion: reduce) {
  .mission-scene-label { transition-duration: .01ms; }
}
</style>
