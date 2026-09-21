<!--
THESIS: AURORA begins as a horizon ignition, not a conventional navigation page.
OWN-WORLD: diagonal night Earth, near-black space, frost-white lettering, one warm solar filament.
STORY: the sunrise draws a filament into the final A, reveals the name, then offers two paths.
FIRST VIEWPORT: wordmark and actions anchor the quiet left field while Earth rises across the right half.
FORM: an orbital title sequence; the filament is both logo stroke and navigation feedback.
-->
<script lang="ts">
// 模块级标记：封面入场动画只在整页首次挂载时播放一次；
// 返回首页等重新挂载直接显示静止封面，避免"回去以后又重演一遍入场"。
let coverEntrancePlayed = false
</script>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import coverEarth from '../assets/aurora-cover-earth.png'

const emit = defineEmits<{
  explore: []
  astronomy: []
  deepTransitionEnd: []
}>()

const props = withDefaults(defineProps<{ activeHome?: boolean }>(), { activeHome: false })

const settled = ref(coverEntrancePlayed)
coverEntrancePlayed = true

const cover = ref<HTMLElement | null>(null)
const launching = ref(false)
const astronomyPending = ref(false)
let pointerFrame = 0

// 封面实例现在跨“返回首页”存活（standby 待命，不再卸载重挂），
// 瞬态标志必须随封面重新成为可交互首页而复位，否则再次点击天文观测会被守卫拦下。
watch(() => props.activeHome, (active) => {
  if (!active) return
  astronomyPending.value = false
  launching.value = false
})

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
  if (launching.value || astronomyPending.value) return
  launching.value = true
  // 与 launching 的 DOM 更新保持在同一个 Vue flush 中。若延后一轮 event loop，浏览器可能先
  // 绘制一帧旧的 is-launching 黑场，再被 deep-preparing 覆盖，看起来像首页重新闪出。
  emit('explore')
}

function handleAnimationEnd(event: AnimationEvent) {
  if (event.target !== event.currentTarget) return
  // scoped CSS 会给 keyframes 添加哈希后缀，因此只匹配稳定前缀。
  if (event.animationName.startsWith('cover-to-deep-space')) emit('deepTransitionEnd')
}

function enterAstronomy() {
  if (launching.value || astronomyPending.value) return
  // 天文观测是同级工作台，切换时保持封面亮度，不复用深空探索的起飞遮罩。
  astronomyPending.value = true
  emit('astronomy')
}

onBeforeUnmount(() => {
  window.cancelAnimationFrame(pointerFrame)
})
</script>

<template>
  <section
    ref="cover"
    class="aurora-cover"
    :class="{ 'is-launching': launching, 'play-entrance': !settled }"
    :aria-busy="astronomyPending"
    aria-labelledby="aurora-cover-title"
    @pointermove="updateParallax"
    @pointerleave="resetParallax"
    @animationend="handleAnimationEnd"
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
          <svg class="cover-path-icon cover-path-icon-helmet" viewBox="0 0 32 32" aria-hidden="true">
            <path d="M7.5 21v-6.2a8.5 8.5 0 0 1 17 0V21" />
            <path d="M10 13.5c1.6-2.2 3.6-3.3 6-3.3s4.4 1.1 6 3.3v4.2c-1.7 1.3-3.7 2-6 2s-4.3-.7-6-2z" />
            <path d="M8.5 21.2v4h15v-4" />
            <path d="M12 25.2v1.5M20 25.2v1.5" />
          </svg>
          <span><strong>深空探索</strong><small>DEEP SPACE</small></span>
        </button>
        <button class="cover-path" type="button" @click="enterAstronomy">
          <svg class="cover-path-icon cover-path-icon-sky" viewBox="0 0 32 32" aria-hidden="true">
            <path d="M4.5 24c6.5-2.1 16.5-2.1 23 0" />
            <path d="M11.5 7.2a6.6 6.6 0 0 0 5.8 10.5A7.2 7.2 0 1 1 11.5 7.2Z" />
            <path d="M23 8v4M21 10h4" />
            <path d="M7 18h3" />
          </svg>
          <span><strong>天文观测</strong><small>ASTRONOMY</small></span>
        </button>
      </nav>
    </div>

    <div class="cover-coordinate" aria-hidden="true">
      <span><strong>太阳系探索</strong><small>SOLAR SYSTEM EXPLORATION</small></span>
      <i />
      <span><strong>本地天空观测</strong><small>LOCAL SKY OBSERVATION</small></span>
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
  /* 点击深空探索后（换页提前到点击瞬间）：封面继续覆盖新场景，直到黑幕结束 */
  z-index: 0;
}
/* 覆盖期间提到所有页面元素之上（页头 z-40 之上、遮罩 z-60 之下） */
.aurora-cover.lingering { z-index: 50; }

/* SKY 已在封面后方完成挂载；一条柔和的夜幕边界从左向右退场，连续揭示观测界面。 */
.aurora-cover.sky-transitioning {
  position: fixed;
  inset: 0;
  pointer-events: none;
  -webkit-mask-image: linear-gradient(90deg, transparent 0 30%, var(--bg) 48% 100%);
  mask-image: linear-gradient(90deg, transparent 0 30%, var(--bg) 48% 100%);
  -webkit-mask-size: 300% 100%;
  mask-size: 300% 100%;
  -webkit-mask-position: 100% 0;
  mask-position: 100% 0;
  animation: cover-to-sky .96s cubic-bezier(.16, 1, .3, 1) both;
}

/* 返回首页 = 入场动画的镜像反向：同一蒙版动画反向播放，封面从右向左扫回盖住观测界面。 */
.aurora-cover.sky-returning {
  position: fixed;
  inset: 0;
  pointer-events: none;
  -webkit-mask-image: linear-gradient(90deg, transparent 0 30%, var(--bg) 48% 100%);
  mask-image: linear-gradient(90deg, transparent 0 30%, var(--bg) 48% 100%);
  -webkit-mask-size: 300% 100%;
  mask-size: 300% 100%;
  -webkit-mask-position: 0 0;
  mask-position: 0 0;
  animation: cover-to-sky .96s cubic-bezier(.16, 1, .3, 1) reverse both;
}
/* SKY 页面期间封面保持挂载但隐藏待命（fixed 出文档流 + visibility 隐藏，不参与绘制）：
   返回首页时只做揭示与蒙版扫回，避免点击瞬间同步挂载整块封面导致的顿挫。 */
.aurora-cover.standby {
  position: fixed;
  inset: 0;
  visibility: hidden;
  pointer-events: none;
}

/* 深空探索使用与 SKY 同族、但更有纵深感的斜向晨昏线：首页不是淡黑消失，
   而是像近景舷窗一样退开，让已经在后方运行的太阳系自然接管画面。 */
@property --deep-wipe {
  syntax: '<percentage>';
  inherits: false;
  initial-value: -24%;
}

.aurora-cover.deep-transitioning,
.aurora-cover.deep-returning {
  --deep-wipe: -24%;
  position: fixed;
  inset: 0;
  pointer-events: none;
  /* 直接移动渐变分界，而不是移动一张 270% 宽的蒙版图。后者的 0/100% position
     并不等于“完全离场”，会在定时器撤类时残留半张封面并闪切。 */
  -webkit-mask-image: linear-gradient(
    112deg,
    transparent 0 var(--deep-wipe),
    rgba(0, 0, 0, .72) calc(var(--deep-wipe) + 9%),
    #000 calc(var(--deep-wipe) + 18%) 100%
  );
  mask-image: linear-gradient(
    112deg,
    transparent 0 var(--deep-wipe),
    rgba(0, 0, 0, .72) calc(var(--deep-wipe) + 9%),
    #000 calc(var(--deep-wipe) + 18%) 100%
  );
  animation: cover-to-deep-space 1.04s cubic-bezier(.16, 1, .3, 1) both;
}

.aurora-cover.deep-returning {
  --deep-wipe: 118%;
  animation-direction: reverse;
}

/* 准备阶段只收拢首页信息，不允许旧的黑幕遮住即将接入的太阳系。 */
.aurora-cover.deep-preparing.is-launching::after,
.aurora-cover.deep-transitioning.is-launching::after,
.aurora-cover.deep-returning.is-launching::after {
  display: none;
}

.aurora-cover.deep-preparing.is-launching .cover-earth,
.aurora-cover.deep-transitioning.is-launching .cover-earth,
.aurora-cover.deep-returning .cover-earth {
  animation: none;
  opacity: 1;
  filter: none;
  transform: translate3d(10px, -2px, 0) scale(1.035);
  transition: transform 1.04s cubic-bezier(.16, 1, .3, 1);
}

/* 返回首页时，蒙版先把地球带回，再让标题和路径从各自锚点重新就位。 */
.aurora-cover.deep-returning .cover-content {
  animation: deep-cover-content-return .62s .22s cubic-bezier(.16, 1, .3, 1) both;
}

.aurora-cover.deep-returning .cover-coordinate {
  animation: deep-cover-coordinate-return .5s .34s cubic-bezier(.16, 1, .3, 1) both;
}

/* 返回时封面是"重新挂载"的：入场动画已由 .play-entrance 门控（仅首次挂载播放），
   保持离开首页时的静止画面，只让蒙版扫回。 */
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

.cover-path-icon {
  flex: 0 0 28px;
  width: 28px;
  height: 28px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.45;
  stroke-linecap: round;
  stroke-linejoin: round;
  transition: color .2s ease, filter .3s ease, transform .3s cubic-bezier(.16, 1, .3, 1);
}

.cover-path-icon-helmet {
  color: rgba(255, 191, 116, .86);
  filter: drop-shadow(3px 5px 9px rgba(255, 176, 88, .12));
}

.cover-path-icon-sky {
  color: rgba(114, 215, 255, .84);
  filter: drop-shadow(3px 5px 9px rgba(72, 183, 232, .14));
}

.cover-path:hover .cover-path-icon,
.cover-path:focus-visible .cover-path-icon {
  transform: translateY(-1px);
}

.cover-path-primary:hover .cover-path-icon,
.cover-path-primary:focus-visible .cover-path-icon {
  color: var(--amber);
  filter: drop-shadow(3px 5px 10px rgba(255, 176, 88, .28));
}

.cover-path:hover .cover-path-icon-sky,
.cover-path:focus-visible .cover-path-icon-sky {
  color: var(--blue);
  filter: drop-shadow(3px 5px 10px rgba(72, 183, 232, .3));
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

.cover-coordinate span,
.cover-coordinate strong,
.cover-coordinate small { display: block; }

.cover-coordinate span { min-width: 118px; }

.cover-coordinate strong {
  color: inherit;
  font: 400 1em/1 var(--font-sans);
  letter-spacing: .14em;
  opacity: .9;
}

.cover-coordinate small {
  margin-top: 5px;
  color: inherit;
  font: 400 .75em/1 var(--font-mono);
  letter-spacing: .13em;
  opacity: .62;
}

.cover-coordinate i {
  width: 24px;
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
  /* 移除入场动画的填充值（fill 模式会压制过渡，导致地球卡在原地不变暗）；
     与遮罩（0.72s）同步淡出到全暗，不再放大，避免"放大+卡一下" */
  animation: none;
  opacity: 0;
  transition: opacity .72s ease;
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
  /* 入场动画仅在整页首次挂载且非起飞时生效（.play-entrance）；返回首页等重新挂载为静止封面 */
  .aurora-cover.play-entrance:not(.is-launching) .cover-earth {
    animation: cover-earth-arrive 1.05s cubic-bezier(.16, 1, .3, 1) both;
  }

  .aurora-cover.play-entrance:not(.is-launching) .cover-wordmark {
    animation: logo-wordmark-arrive .64s .68s cubic-bezier(.16, 1, .3, 1) both;
  }

  .aurora-cover.play-entrance:not(.is-launching) .cover-expansion {
    animation: cover-copy-arrive .48s 1.5s cubic-bezier(.16, 1, .3, 1) both;
  }

  .aurora-cover.play-entrance:not(.is-launching) .cover-paths {
    animation: cover-copy-arrive .5s 1.72s cubic-bezier(.16, 1, .3, 1) both;
  }

  .aurora-cover.play-entrance:not(.is-launching) .cover-coordinate {
    animation: cover-copy-arrive .5s 1.88s cubic-bezier(.16, 1, .3, 1) both;
  }

}

@keyframes cover-to-sky {
  from {
    -webkit-mask-position: 100% 0;
    mask-position: 100% 0;
  }
  to {
    -webkit-mask-position: 0 0;
    mask-position: 0 0;
  }
}

@keyframes cover-to-deep-space {
  from { --deep-wipe: -24%; }
  to { --deep-wipe: 118%; }
}

@keyframes deep-cover-content-return {
  from { opacity: 0; filter: blur(7px); transform: translateY(-50%) translateX(-20px); }
  to { opacity: 1; filter: blur(0); transform: translateY(-50%) translateX(0); }
}

@keyframes deep-cover-coordinate-return {
  from { opacity: 0; filter: blur(5px); transform: translateX(18px); }
  to { opacity: 1; filter: blur(0); transform: translateX(0); }
}

@media (prefers-reduced-motion: reduce) {
  .aurora-cover.sky-transitioning,
  .aurora-cover.sky-returning,
  .aurora-cover.deep-transitioning,
  .aurora-cover.deep-returning {
    animation-duration: .04s;
  }

  .aurora-cover.deep-returning .cover-content,
  .aurora-cover.deep-returning .cover-coordinate { animation-duration: .04s; animation-delay: 0s; }
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
