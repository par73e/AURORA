<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useAttrs, watch } from 'vue'

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()

const props = withDefaults(defineProps<{
  kind: 'spacecraft' | 'surface'
  nameZh: string
  nameEn?: string
  selected?: boolean
  iconHtml?: string
  compact?: boolean
  mode?: 'full' | 'compact' | 'cluster'
  clusterCount?: number
  connector?: boolean
  side?: 'left' | 'right'
  clusterItems?: Array<{ id: string; name: string }>
}>(), {
  mode: 'full',
  clusterCount: 1,
  connector: true,
  side: 'right',
  clusterItems: () => [],
})

const emit = defineEmits<{
  click: [event: MouseEvent]
  'select-member': [id: string]
}>()
const trigger = ref<HTMLButtonElement | null>(null)
const menu = ref<HTMLDivElement | null>(null)
const menuOpen = ref(false)
const menuStyle = ref<Record<string, string>>({})
let menuFrame = 0

function closeMenu() {
  menuOpen.value = false
  cancelAnimationFrame(menuFrame)
  document.removeEventListener('pointerdown', onOutsidePointer)
  document.removeEventListener('keydown', onMenuKeydown)
}

function onOutsidePointer(event: PointerEvent) {
  if (!trigger.value?.contains(event.target as Node) && !menu.value?.contains(event.target as Node)) closeMenu()
}

function onMenuKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.stopPropagation()
    closeMenu()
    trigger.value?.focus({ preventScroll: true })
  }
}

function positionMenu() {
  if (!menuOpen.value || !trigger.value) return
  const bounds = trigger.value.getBoundingClientRect()
  if (!bounds.width || !bounds.height) return closeMenu()
  const width = Math.min(280, window.innerWidth - 16)
  const height = menu.value?.offsetHeight ?? 240
  const left = Math.max(8, Math.min(bounds.left, window.innerWidth - width - 8))
  const top = bounds.bottom + height + 8 <= window.innerHeight
    ? bounds.bottom + 6
    : Math.max(8, bounds.top - height - 6)
  const theme = getComputedStyle(trigger.value)
  menuStyle.value = {
    left: `${left}px`, top: `${top}px`, width: `${width}px`,
    '--mission-accent': theme.getPropertyValue('--mission-accent'),
    '--mission-text': theme.getPropertyValue('--mission-text'),
    '--mission-quiet': theme.getPropertyValue('--mission-quiet'),
  }
  menuFrame = requestAnimationFrame(positionMenu)
}

async function onClick(event: MouseEvent) {
  if (props.mode !== 'cluster' || !props.clusterItems.length) return emit('click', event)
  event.stopPropagation()
  if (menuOpen.value) return closeMenu()
  menuOpen.value = true
  positionMenu()
  document.addEventListener('pointerdown', onOutsidePointer)
  document.addEventListener('keydown', onMenuKeydown)
  await nextTick()
  menu.value?.querySelector('button')?.focus({ preventScroll: true })
}

function selectMember(id: string) {
  closeMenu()
  emit('select-member', id)
}

watch(() => `${props.mode}:${props.clusterItems.map((item) => item.id).join(',')}`, closeMenu)
onBeforeUnmount(closeMenu)

function isCompact() {
  return props.compact || props.mode === 'compact'
}
</script>

<template>
  <button
    v-bind="attrs"
    ref="trigger"
    class="mission-scene-label"
    :class="[
      `is-${kind}`,
      `is-${mode}`,
      `on-${side}`,
      { selected, 'is-compact': isCompact(), 'has-connector': connector },
    ]"
    type="button"
    :aria-expanded="mode === 'cluster' ? menuOpen : undefined"
    :aria-label="mode === 'cluster' ? `查看 ${clusterCount} 个相近任务` : (attrs['aria-label'] as string | undefined)"
    @click="onClick"
  >
    <span v-if="mode === 'cluster'" class="mission-scene-label-cluster" aria-hidden="true">
      <i /><i /><i />
      <b>×{{ clusterCount }}</b>
    </span>
    <template v-else>
      <span v-if="iconHtml" class="mission-scene-label-icon" aria-hidden="true" v-html="iconHtml" />
      <i v-else class="mission-scene-label-dot" aria-hidden="true" />
      <span class="mission-scene-label-copy">
        <strong>{{ nameZh }}</strong>
        <small v-if="nameEn && nameEn !== nameZh">（{{ nameEn }}）</small>
      </span>
    </template>
    <Teleport to="body">
      <div v-if="menuOpen" ref="menu" class="mission-cluster-menu" :style="menuStyle" role="group" aria-label="此处的任务">
        <p>此处有 {{ clusterItems.length }} 个任务</p>
        <div class="mission-cluster-menu-items">
          <button v-for="item in clusterItems" :key="item.id" type="button" @click.stop="selectMember(item.id)">{{ item.name }}</button>
        </div>
      </div>
    </Teleport>
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
.mission-scene-label.has-connector::before {
  content: '';
  position: absolute;
  right: 100%;
  top: 50%;
  width: 10px;
  height: 1px;
  background: var(--mission-accent-dim, rgba(114, 215, 255, .38));
  transform: translateY(-50%);
}
.mission-scene-label.has-connector.on-left::before { right: auto; left: 100%; }
.mission-scene-label.selected { z-index: 6; }
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
  text-shadow: var(--mission-label-text-shadow, none);
  -webkit-text-stroke: var(--mission-label-text-stroke, 0 transparent);
  paint-order: stroke fill;
}
.mission-scene-label small {
  color: var(--mission-quiet, #7f98a7);
  font: 400 8px/1.3 var(--font-mono);
  letter-spacing: .08em;
  white-space: nowrap;
  text-shadow: var(--mission-label-text-shadow, none);
  -webkit-text-stroke: var(--mission-label-text-stroke, 0 transparent);
  paint-order: stroke fill;
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
.mission-scene-label.is-compact .mission-scene-label-dot {
  display: none;
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
.mission-scene-label.is-compact:focus-visible {
  border-color: color-mix(in srgb, var(--mission-accent, #72d7ff) 54%, transparent);
  background: rgba(6, 17, 26, .62);
  backdrop-filter: blur(4px);
}
.mission-scene-label.is-compact.selected {
  border-color: var(--mission-accent, #72d7ff);
  background: var(--mission-label-surface-active, rgba(6, 17, 26, .86));
  backdrop-filter: blur(5px);
}
.mission-scene-label.is-cluster {
  min-width: 42px;
  min-height: 22px;
  padding: 3px 5px;
  border-color: color-mix(in srgb, var(--mission-accent, #72d7ff) 42%, transparent);
  background: var(--mission-label-surface, rgba(3, 10, 17, .78));
}
.mission-scene-label-cluster {
  position: relative;
  display: inline-flex;
  align-items: center;
  min-width: 30px;
  height: 14px;
}
.mission-scene-label-cluster i {
  position: absolute;
  left: 1px;
  width: 7px;
  height: 7px;
  border: 1px solid color-mix(in srgb, var(--mission-accent, #72d7ff) 72%, transparent);
  border-radius: 2px;
  background: var(--mission-label-surface-active, rgba(6, 17, 26, .9));
}
.mission-scene-label-cluster i:nth-child(2) { left: 4px; top: 2px; }
.mission-scene-label-cluster i:nth-child(3) { left: 7px; top: 4px; }
.mission-scene-label-cluster b {
  margin-left: 17px;
  color: var(--mission-text, #ecf5f9);
  font: 500 8px/1 var(--font-mono);
  letter-spacing: .04em;
}
.mission-cluster-menu {
  position: fixed;
  z-index: 60;
  padding: 10px;
  border: 1px solid var(--mission-accent, #72d7ff);
  border-radius: 4px;
  background: #07111b;
  color: var(--mission-text, #ecf5f9);
  text-align: left;
}
.mission-cluster-menu p { margin: 0 0 6px; color: var(--mission-quiet, #7f98a7); font: 11px/1.5 var(--font-sans); }
.mission-cluster-menu-items { max-height: 192px; overflow-y: auto; }
.mission-cluster-menu-items button {
  display: block;
  width: 100%;
  padding: 7px 6px;
  border: 0;
  background: transparent;
  color: inherit;
  font: 12px/1.5 var(--font-sans);
  text-align: left;
  overflow-wrap: anywhere;
  cursor: pointer;
}
.mission-cluster-menu-items button:hover,
.mission-cluster-menu-items button:focus-visible { background: #132533; outline: 1px solid var(--mission-accent, #72d7ff); outline-offset: -1px; }
@media (prefers-reduced-motion: reduce) {
  .mission-scene-label { transition-duration: .01ms; }
}
</style>
