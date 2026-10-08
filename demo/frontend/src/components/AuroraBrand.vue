<script setup lang="ts">
withDefaults(defineProps<{
  subtitle: string
  variant?: 'deep-space' | 'sky'
  leaving?: boolean
}>(), {
  variant: 'deep-space',
  leaving: false,
})
</script>

<template>
  <a
    class="aurora-brand brand"
    :class="[`is-${variant}`, { 'is-leaving': leaving }]"
    href="#home"
    aria-label="返回 AURORA 封面"
  >
    <span class="aurora-brand-mark brand-mark" aria-hidden="true"><i /><i /><i /></span>
    <span class="aurora-brand-copy"><strong>AURORA</strong><small>{{ subtitle }}</small></span>
  </a>
</template>

<style>
.aurora-brand {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  width: max-content;
  color: inherit;
  text-decoration: none;
}
.aurora-brand-mark {
  position: relative;
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  border: 1px solid rgba(114, 215, 255, .48);
  border-radius: 50%;
  transition: transform .26s cubic-bezier(.22, 1, .36, 1), border-color .22s, box-shadow .22s;
}
.aurora-brand-mark i {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--blue, #72d7ff);
  transform: translate(-50%, -50%);
  transition: transform .32s cubic-bezier(.22, 1, .36, 1), background .22s, border-color .22s, opacity .22s;
}
.aurora-brand-mark i:nth-child(2) {
  width: 25px;
  height: 1px;
  border-radius: 0;
  background: rgba(114, 215, 255, .42);
  transform: translate(-50%, -50%) rotate(-38deg);
}
.aurora-brand-mark i:nth-child(3) {
  left: -4px;
  top: 7px;
  width: 36px;
  height: 13px;
  border: 1px solid rgba(114, 215, 255, .28);
  background: none;
  transform: rotate(18deg);
}
.aurora-brand-copy {
  min-width: 0;
  transition: opacity .24s, transform .32s cubic-bezier(.22, 1, .36, 1);
}
.aurora-brand strong,
.aurora-brand small { display: block; }
.aurora-brand strong {
  font-size: 13px;
  font-weight: 600;
  letter-spacing: .2em;
  transition: color .22s, text-shadow .22s;
}
.aurora-brand small {
  margin-top: 3px;
  color: var(--muted, #7f98a7);
  font: 400 8px var(--font-mono);
  letter-spacing: .14em;
  white-space: nowrap;
  transition: color .22s;
}
.aurora-brand:hover .aurora-brand-mark,
.aurora-brand:focus-visible .aurora-brand-mark {
  border-color: rgba(114, 215, 255, .9);
  box-shadow: 0 0 0 1px rgba(114, 215, 255, .22), 0 6px 20px rgba(0, 8, 18, .42), inset 0 0 12px rgba(114, 215, 255, .2);
  transform: translateY(-1px) scale(1.05);
}
.aurora-brand:hover .aurora-brand-mark i,
.aurora-brand:focus-visible .aurora-brand-mark i { background: #72d7ff; }
.aurora-brand:hover .aurora-brand-mark i:nth-child(2),
.aurora-brand:focus-visible .aurora-brand-mark i:nth-child(2) { background: rgba(114, 215, 255, .95); }
.aurora-brand:hover .aurora-brand-mark i:nth-child(3),
.aurora-brand:focus-visible .aurora-brand-mark i:nth-child(3) { border-color: rgba(114, 215, 255, .72); background: var(--blue, #72d7ff); }
.aurora-brand:hover strong,
.aurora-brand:focus-visible strong { color: #edf7fb; text-shadow: 0 2px 14px rgba(0, 8, 18, .72); }
.aurora-brand:hover small,
.aurora-brand:focus-visible small { color: var(--blue, #72d7ff); }
.aurora-brand:focus-visible {
  border-radius: 4px;
  outline: 1px solid rgba(114, 215, 255, .76);
  outline-offset: 5px;
}

/* 天文观测返回：轨道向中心收拢，像关闭观测光阑；深空页仍使用外层空间缩放。 */
.aurora-brand.is-sky.is-leaving .aurora-brand-mark {
  border-color: color-mix(in srgb, var(--sky-lunar) 72%, transparent);
  box-shadow: 0 6px 18px rgba(0, 8, 18, .38);
  transform: scale(.9) rotate(-7deg);
}
.aurora-brand.is-sky.is-leaving .aurora-brand-mark i:first-child {
  background: #edf5f7;
  transform: translate(-50%, -50%) scale(1.5);
}
.aurora-brand.is-sky.is-leaving .aurora-brand-mark i:nth-child(2) {
  transform: translate(-50%, -50%) rotate(-12deg) scaleX(.58);
}
.aurora-brand.is-sky.is-leaving .aurora-brand-mark i:nth-child(3) {
  opacity: .58;
  transform: rotate(-10deg) scale(.72);
}
.aurora-brand.is-sky.is-leaving .aurora-brand-copy {
  opacity: .46;
  transform: translateX(-3px);
}

@media (prefers-reduced-motion: reduce) {
  .aurora-brand-mark,
  .aurora-brand-mark i,
  .aurora-brand-copy { transition-duration: .01ms; }
}
</style>
