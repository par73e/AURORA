<script setup lang="ts">
import type { MissionDetail } from '../missionPresentation'
defineProps<{ detail: MissionDetail }>()
defineEmits<{ close: [] }>()
</script>

<template>
  <aside class="mission-detail-panel" :class="`is-${detail.kind}`" :aria-label="`${detail.nameZh}详情`">
    <button class="mission-detail-close" type="button" aria-label="关闭详情" @click="$emit('close')">关闭</button>
    <header class="mission-detail-head">
      <span v-if="detail.iconHtml" class="mission-detail-icon" aria-hidden="true" v-html="detail.iconHtml" />
      <div>
        <p class="mission-detail-type">
          <span>{{ detail.typeZh }}</span>
          <small>{{ detail.typeEn }}</small>
          <em v-if="detail.status">{{ detail.status }}</em>
        </p>
        <h2>{{ detail.nameZh }}</h2>
        <p v-if="detail.nameEn && detail.nameEn !== detail.nameZh" class="mission-detail-name-en">（{{ detail.nameEn }}）</p>
      </div>
    </header>

    <p v-if="detail.description" class="mission-detail-description">{{ detail.description }}</p>

    <dl v-if="detail.fields.length" class="mission-detail-fields">
      <div v-for="field in detail.fields" :key="field.label">
        <dt>{{ field.label }}</dt>
        <dd>{{ field.value }}</dd>
      </div>
    </dl>

    <section v-if="detail.hardware?.length" class="mission-detail-hardware">
      <h3>任务设施 <small>MISSION HARDWARE</small></h3>
      <ul><li v-for="item in detail.hardware" :key="item">{{ item }}</li></ul>
    </section>

    <footer v-if="detail.source" class="mission-detail-footer">
      <small>数据来源：{{ detail.source }}</small>
    </footer>
  </aside>
</template>

<style scoped>
.mission-detail-panel {
  position: absolute;
  z-index: 8;
  top: 18px;
  right: 32px;
  width: clamp(368px, 25vw, 460px);
  max-height: calc(100% - 36px - var(--header-overlay-offset, 0px));
  overflow-y: auto;
  padding: 26px 28px 24px;
  border-radius: 12px;
  background: var(--mission-panel-surface, rgba(5, 14, 22, .94));
  color: var(--mission-text, #ecf5f9);
  box-shadow: 0 24px 70px var(--mission-shadow, rgba(0, 8, 18, .5));
  backdrop-filter: blur(20px);
  scrollbar-width: thin;
  animation: mission-panel-enter .34s cubic-bezier(.16, 1, .3, 1) both;
}
.mission-detail-close {
  position: absolute;
  top: 17px;
  right: 17px;
  padding: 5px 7px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--mission-quiet, #7f98a7);
  font: 500 10px var(--font-sans);
  cursor: pointer;
}
.mission-detail-close:hover,
.mission-detail-close:focus-visible { color: var(--mission-text, #ecf5f9); }
.mission-detail-close:focus-visible { outline: 1px solid var(--mission-accent, #72d7ff); outline-offset: 2px; }
.mission-detail-head { display: flex; gap: 12px; padding-right: 48px; }
.mission-detail-icon { display: inline-flex; margin-top: 3px; color: var(--mission-accent, #72d7ff); }
.mission-detail-icon :deep(svg) { width: 18px; height: 18px; }
.mission-detail-type {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 9px;
  margin: 0 0 10px;
  color: var(--mission-accent, #72d7ff);
  font: 500 10px/1.4 var(--font-mono);
  letter-spacing: .08em;
}
.mission-detail-type small { color: var(--mission-quiet, #7f98a7); font: inherit; }
.mission-detail-type em {
  padding-inline-start: 9px;
  border-inline-start: 1px solid var(--mission-line, rgba(139, 180, 202, .18));
  color: var(--mission-text, #ecf5f9);
  font-style: normal;
}
.mission-detail-head h2 { max-width: 320px; margin: 0; color: inherit; font-size: 24px; font-weight: 500; line-height: 1.18; letter-spacing: -.025em; }
.mission-detail-name-en { margin: 6px 0 0; color: var(--mission-quiet, #7f98a7); font: 400 11px/1.5 var(--font-mono); }
.mission-detail-description { max-width: 68ch; margin: 22px 0; color: var(--mission-body, #a2b4bd); font-size: 12px; line-height: 1.75; }
.mission-detail-fields { margin: 0; }
.mission-detail-fields > div { display: grid; grid-template-columns: 94px minmax(0, 1fr); gap: 18px; padding: 11px 0; border-top: 1px solid var(--mission-line, rgba(139, 180, 202, .18)); font-size: 10px; }
.mission-detail-fields dt { color: var(--mission-quiet, #7f98a7); }
.mission-detail-fields dd { margin: 0; color: inherit; text-align: right; line-height: 1.55; overflow-wrap: anywhere; }
.mission-detail-hardware { padding-top: 16px; border-top: 1px solid var(--mission-line, rgba(139, 180, 202, .18)); }
.mission-detail-hardware h3 { margin: 0 0 10px; color: var(--mission-quiet, #7f98a7); font-size: 10px; font-weight: 500; }
.mission-detail-hardware h3 small { margin-inline-start: 6px; font: 400 9px var(--font-mono); }
.mission-detail-hardware ul { display: grid; gap: 6px; margin: 0; padding-inline-start: 17px; }
.mission-detail-hardware li { color: var(--mission-body, #a2b4bd); font-size: 11px; line-height: 1.55; }
.mission-detail-footer { display: grid; gap: 6px; margin-top: 18px; padding-top: 14px; border-top: 1px solid var(--mission-line, rgba(139, 180, 202, .18)); }
.mission-detail-footer small { color: var(--mission-quiet, #7f98a7); font: 400 9px/1.6 var(--font-mono); overflow-wrap: anywhere; }
@keyframes mission-panel-enter {
  from { opacity: .2; filter: blur(5px); transform: translateY(10px); }
  to { opacity: 1; filter: blur(0); transform: translateY(var(--header-overlay-offset, 0px)); }
}
@media (max-width: 900px) {
  .mission-detail-panel { right: 20px; width: min(420px, calc(100vw - 40px)); }
}
@media (prefers-reduced-motion: reduce) {
  .mission-detail-panel { animation-duration: .01ms; }
}
</style>
