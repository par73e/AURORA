<!--
THESIS: AURORA begins as a horizon ignition, not a conventional navigation page.
OWN-WORLD: diagonal night Earth, near-black space, frost-white lettering, one warm solar filament.
STORY: the sunrise draws a filament into the final A, reveals the name, then offers two paths.
FIRST VIEWPORT: wordmark and actions anchor the quiet left field while Earth rises across the right half.
FORM: an orbital title sequence; the filament is both logo stroke and navigation feedback.
-->
<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import coverEarth from '../assets/aurora-cover-earth.png'

const emit = defineEmits<{
  explore: []
}>()

const cover = ref<HTMLElement | null>(null)
const launching = ref(false)
const astronomyNotice = ref(false)
let pointerFrame = 0
let astronomyTimer: number | undefined
let launchTimer: number | undefined

function updateParallax(event: PointerEvent) {
  if (!cover.value || launching.value || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  const bounds = cover.value.getBoundingClientRect()
  const x = ((event.clientX - bounds.left) / bounds.width - 0.5) * 2
  const y = ((event.clientY - bounds.top) / bounds.height - 0.5) * 2
  window.cancelAnimationFrame(pointerFrame)
  pointerFrame = window.requestAnimationFrame(() => {
    cover.value?.style.setProperty('--pointer-x', x.toFixed(3))
    cover.value?.style.setProperty('--pointer-y', y.toFixed(3))
  })
}

function resetParallax() {
  cover.value?.style.setProperty('--pointer-x', '0')
  cover.value?.style.setProperty('--pointer-y', '0')
}

function enterDeepSpace() {
  if (launching.value) return
  launching.value = true
  launchTimer = window.setTimeout(() => emit('explore'), 720)
}

function previewAstronomy() {
  astronomyNotice.value = true
  if (astronomyTimer) window.clearTimeout(astronomyTimer)
  astronomyTimer = window.setTimeout(() => { astronomyNotice.value = false }, 3200)
}

onBeforeUnmount(() => {
  window.cancelAnimationFrame(pointerFrame)
  if (astronomyTimer) window.clearTimeout(astronomyTimer)
  if (launchTimer) window.clearTimeout(launchTimer)
})
</script>

<template>
  <section
    ref="cover"
    class="aurora-cover"
    :class="{ 'is-launching': launching, 'astronomy-active': astronomyNotice }"
    aria-labelledby="aurora-cover-title"
    @pointermove="updateParallax"
    @pointerleave="resetParallax"
  >
    <img class="cover-earth" :src="coverEarth" alt="" aria-hidden="true">
    <div class="cover-vignette" aria-hidden="true" />

    <div class="cover-content">
      <div class="cover-wordmark-wrap">
        <div class="cover-logo-stage">
          <h1 id="aurora-cover-title" class="cover-wordmark">AURORA</h1>
        </div>
        <p class="cover-expansion" aria-label="A Universe of Rockets Orbits Research and Astronomy">
          <span><strong>A</strong> Universe of</span>
          <span><strong>R</strong>ockets</span>
          <span><strong>O</strong>rbits</span>
          <span><strong>R</strong>esearch and</span>
          <span><strong>A</strong>stronomy</span>
        </p>
      </div>

      <nav class="cover-paths" aria-label="AURORA 探索路径">
        <button class="cover-path cover-path-primary" type="button" @click="enterDeepSpace">
          <i aria-hidden="true" />
          <span><strong>深空探索</strong><small>DEEP SPACE</small></span>
        </button>
        <button class="cover-path" type="button" @click="previewAstronomy">
          <span><strong>天文观测</strong><small>ASTRONOMY</small></span>
        </button>
      </nav>

      <p class="cover-notice" :class="{ visible: astronomyNotice }" aria-live="polite">
        天文观测模块正在建设中
      </p>
    </div>

    <div class="cover-coordinate" aria-hidden="true">
      <span>EARTH LIMB</span>
      <i />
      <span>ORBITAL ENTRY</span>
    </div>
  </section>
</template>

<style scoped>
.aurora-cover {
  --pointer-x: 0;
  --pointer-y: 0;
  position: relative;
  width: 100%;
  height: 100dvh;
  min-height: 660px;
  overflow: hidden;
  isolation: isolate;
  background: #010307;
  color: #f2f7fa;
}

.cover-earth {
  position: absolute;
  z-index: -4;
  inset: -2.5%;
  width: 105%;
  height: 105%;
  object-fit: cover;
  transform: translate3d(calc(var(--pointer-x) * -7px), calc(var(--pointer-y) * -5px), 0) scale(1.025);
  transition: transform 1.1s cubic-bezier(.16, 1, .3, 1), filter .7s ease, opacity .6s ease;
}

.cover-vignette {
  position: absolute;
  z-index: -3;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(0, 2, 6, .38) 0%, rgba(0, 2, 6, .06) 58%, rgba(0, 2, 6, .12) 100%),
    linear-gradient(180deg, rgba(0, 2, 6, .1), transparent 54%, rgba(0, 2, 6, .28));
}

.cover-content {
  --cover-content-inset: 24px;
  position: absolute;
  z-index: 3;
  left: clamp(72px, 7vw, 128px);
  top: 38%;
  width: min(770px, 56vw);
  transform: translateY(-50%);
}

.cover-wordmark-wrap {
  width: min(720px, 56vw);
  max-width: 100%;
}

.cover-logo-stage {
  position: relative;
  width: 100%;
  height: 110px;
  filter: drop-shadow(0 10px 24px rgba(0, 7, 15, .34));
}

.cover-wordmark {
  position: absolute;
  left: var(--cover-content-inset);
  top: 0;
  margin: 0;
  color: #f5f7f8;
  font-family: Montserrat, Manrope, sans-serif;
  font-size: clamp(76px, 7.6vw, 102px);
  font-weight: 200;
  line-height: .92;
  letter-spacing: .17em;
  white-space: nowrap;
  text-shadow: 0 1px 18px rgba(220, 240, 248, .08);
}

.cover-expansion {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin: 8px 0 0 var(--cover-content-inset);
  color: rgba(224, 234, 239, .84);
  font-family: var(--font-sans);
  font-size: clamp(12px, 1.12vw, 16px);
  font-weight: 400;
  line-height: 1.5;
  letter-spacing: .11em;
  white-space: nowrap;
}

.cover-expansion span {
  display: inline-flex;
  align-items: baseline;
}

.cover-expansion strong {
  margin-right: .12em;
  color: #f7fbfd;
  font-size: 1.34em;
  font-weight: 500;
}

.cover-paths {
  display: flex;
  align-items: center;
  gap: clamp(42px, 5vw, 84px);
  margin: 48px 0 0 var(--cover-content-inset);
}

.cover-path {
  position: relative;
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 132px;
  min-height: 54px;
  padding: 4px 0;
  border: 0;
  background: transparent;
  color: rgba(211, 224, 231, .74);
  text-align: left;
}

.cover-path::after {
  content: '';
  position: absolute;
  left: 0;
  bottom: 0;
  width: 100%;
  height: 1px;
  background: currentColor;
  transform: scaleX(.18);
  transform-origin: left;
  opacity: .38;
  transition: transform .42s cubic-bezier(.16, 1, .3, 1), opacity .2s;
}

.cover-path:hover,
.cover-path:focus-visible {
  color: #f4f9fb;
  outline: 0;
}

.cover-path:hover::after,
.cover-path:focus-visible::after {
  transform: scaleX(1);
  opacity: .72;
}

.cover-path i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #ffbf74;
  box-shadow: 3px 5px 14px 4px rgba(255, 176, 88, .18);
  transition: transform .3s cubic-bezier(.16, 1, .3, 1), background .2s;
}

.cover-path-primary:hover i,
.cover-path-primary:focus-visible i {
  background: #fff0d8;
  transform: scale(1.35);
}

.cover-path span,
.cover-path strong,
.cover-path small {
  display: block;
}

.cover-path strong {
  font-size: 17px;
  font-weight: 400;
  letter-spacing: .14em;
}

.cover-path small {
  margin-top: 6px;
  color: rgba(118, 143, 156, .68);
  font: 400 8px/1 'IBM Plex Mono', monospace;
  letter-spacing: .18em;
}

.cover-notice {
  min-height: 18px;
  margin: 20px 0 0 2px;
  color: rgba(145, 166, 177, .72);
  font-size: 11px;
  letter-spacing: .08em;
  opacity: 0;
  transform: translateY(6px);
  transition: opacity .25s ease, transform .35s cubic-bezier(.16, 1, .3, 1);
}

.cover-notice.visible {
  opacity: 1;
  transform: translateY(0);
}

.cover-coordinate {
  position: absolute;
  z-index: 3;
  right: 42px;
  bottom: 30px;
  display: flex;
  align-items: center;
  gap: 12px;
  color: rgba(172, 194, 204, .42);
  font: 400 8px/1 'IBM Plex Mono', monospace;
  letter-spacing: .16em;
}

.cover-coordinate i {
  width: 30px;
  height: 1px;
  background: rgba(172, 194, 204, .32);
}

.aurora-cover.is-launching .cover-content,
.aurora-cover.is-launching .cover-coordinate {
  opacity: 0;
  transform: translateY(-50%) translateX(-20px);
  filter: blur(7px);
  transition: opacity .32s ease, transform .58s cubic-bezier(.16, 1, .3, 1), filter .42s ease;
}

.aurora-cover.is-launching .cover-coordinate {
  transform: translateX(20px);
}

.aurora-cover.is-launching .cover-earth {
  filter: brightness(1.16);
  opacity: .64;
  transform: translate3d(-12px, -5px, 0) scale(1.075);
}

.aurora-cover.is-launching::after {
  content: '';
  position: absolute;
  z-index: 8;
  inset: 0;
  background: #02070d;
  opacity: 1;
  animation: cover-exit-veil .72s cubic-bezier(.16, 1, .3, 1) both;
}

@media (prefers-reduced-motion: no-preference) {
  .cover-earth {
    animation: cover-earth-arrive 1.05s cubic-bezier(.16, 1, .3, 1) both;
  }

  .cover-wordmark {
    animation: logo-wordmark-arrive .64s .68s cubic-bezier(.16, 1, .3, 1) both;
  }

  .cover-expansion {
    animation: cover-copy-arrive .48s 1.5s cubic-bezier(.16, 1, .3, 1) both;
  }

  .cover-paths {
    animation: cover-copy-arrive .5s 1.72s cubic-bezier(.16, 1, .3, 1) both;
  }

  .cover-coordinate {
    animation: cover-copy-arrive .5s 1.88s cubic-bezier(.16, 1, .3, 1) both;
  }

  .aurora-cover.astronomy-active .cover-earth {
    filter: brightness(1.05) saturate(1.06);
  }

}

@keyframes cover-earth-arrive {
  from { opacity: .2; filter: brightness(.55) blur(5px); transform: scale(1.065); }
}

@keyframes logo-wordmark-arrive {
  from { opacity: 0; filter: blur(6px); clip-path: inset(0 100% 0 0); transform: translateX(-8px); }
  to { opacity: 1; filter: blur(0); clip-path: inset(0 0 0 0); transform: translateX(0); }
}

@keyframes cover-copy-arrive {
  from { opacity: 0; filter: blur(5px); transform: translateY(10px); }
}

@keyframes cover-exit-veil {
  from { opacity: 0; }
  55% { opacity: .18; }
  to { opacity: 1; }
}

@media (max-width: 1240px) {
  .cover-content { left: 64px; width: 700px; }
  .cover-wordmark-wrap { width: 640px; }
  .cover-wordmark { font-size: 84px; }
  .cover-expansion { font-size: 12px; }
}

@media (prefers-reduced-motion: reduce) {
  .cover-earth { transform: none; transition: none; }
}
</style>
