<template>
  <button
    class="solar-system-item"
    :class="{
      'is-active': active,
      'is-animated': animated
    }"
    :style="componentStyle"
    type="button"
    @click="$emit('click')"
  >
    <span class="solar-icon" aria-hidden="true">
      <!-- 轨道 -->
      <span class="orbit orbit-outer">
        <span class="planet planet-blue"></span>
      </span>

      <span class="orbit orbit-middle">
        <span class="planet planet-cyan"></span>
      </span>

      <span class="orbit orbit-inner">
        <span class="planet planet-orange"></span>
      </span>

      <!-- 中心太阳 -->
      <span class="sun">
        <span class="sun-core"></span>
      </span>

      <!-- 装饰光点 -->
      <span class="star-dot star-dot-1"></span>
      <span class="star-dot star-dot-2"></span>
      <span class="star-dot star-dot-3"></span>
    </span>

    <span class="solar-title">
      {{ title }}
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps({
  title: {
    type: String,
    default: '太阳系总览'
  },

  /**
   * 整个图标区域的宽度
   * 推荐范围：52 ~ 90
   */
  iconSize: {
    type: Number,
    default: 68
  },

  active: {
    type: Boolean,
    default: false
  },

  animated: {
    type: Boolean,
    default: true
  }
})

defineEmits(['click'])

const componentStyle = computed(() => ({
  '--icon-size': `${props.iconSize}px`
}))
</script>

<style scoped>
.solar-system-item {
  --icon-size: 68px;

  --text-color: #b4c6ce;

  --sun-core: #fff4bd;
  --sun-main: #ffc75f;
  --sun-edge: #e7932e;

  --orbit-blue: rgba(74, 188, 255, 0.9);
  --orbit-cyan: rgba(83, 225, 255, 0.58);
  --orbit-gold: rgba(255, 189, 81, 0.42);

  position: relative;

  display: inline-flex;
  align-items: center;
  gap: 10px;

  min-height: 0;
  padding: 6px 10px;

  color: var(--text-color);
  font: inherit;

  border: 0;
  border-radius: 0;

  /* 与页头背景严格一致 rgba(3, 7, 12, .9) */
  background: rgba(3, 7, 12, 0.9);

  cursor: pointer;
  overflow: hidden;

  transition:
    background 240ms ease,
    transform 240ms ease;
}

/* 鼠标经过时扫过的科技光线 */
.solar-system-item::after {
  content: "";

  position: absolute;
  top: -100%;
  left: -35%;

  width: 24%;
  height: 300%;

  background: linear-gradient(
    90deg,
    transparent,
    rgba(125, 220, 255, 0.11),
    transparent
  );

  transform: rotate(18deg);
  transition: left 600ms ease;
  pointer-events: none;
}

.solar-system-item:not(.is-active):hover {
  /* 与页头背景严格一致，不再叠加渐变光晕 */
  background: rgba(3, 7, 12, 0.9);

  transform: translateY(-1px);
}

.solar-system-item:not(.is-active):hover::after {
  left: 125%;
}

.solar-system-item:not(.is-active):active {
  transform: translateY(0) scale(0.985);
}

/* 激活状态：纯背景，无任何光晕 */
.solar-system-item.is-active {
  background: rgba(3, 7, 12, 0.9);
}

/* =========================
   图标主体
   ========================= */

.solar-icon {
  position: relative;

  display: block;
  flex: 0 0 auto;

  width: var(--icon-size);
  height: calc(var(--icon-size) * 0.72);
}

/* 图标背后的蓝金色光晕（已移除，避免与页头背景不一致的渐变色块） */

/* =========================
   太阳
   ========================= */

.sun {
  position: absolute;
  left: 37%;
  top: 42%;

  width: 22%;
  aspect-ratio: 1;

  border-radius: 50%;

  transform: translate(-50%, -50%);

  background:
    radial-gradient(
      circle at 34% 30%,
      var(--sun-core) 0%,
      var(--sun-main) 42%,
      var(--sun-edge) 100%
    );

  box-shadow:
    0 0 4px rgba(255, 235, 153, 0.55),
    0 0 10px rgba(255, 193, 73, 0.45),
    0 0 14px rgba(255, 163, 47, 0.22);

  z-index: 8;
}

/* 太阳外层扩散光（收窄到图标内部，避免被按钮边界裁切出色块） */
.sun::before {
  content: "";

  position: absolute;
  inset: -40%;

  border-radius: 50%;

  background: radial-gradient(
    circle,
    rgba(255, 199, 83, 0.13),
    rgba(255, 160, 45, 0.04) 43%,
    transparent 68%
  );

  filter: blur(2px);
}

.sun-core {
  position: absolute;
  left: 27%;
  top: 22%;

  width: 25%;
  height: 25%;

  border-radius: 50%;

  background: rgba(255, 255, 255, 0.55);
  filter: blur(0.5px);
}

/* =========================
   轨道
   ========================= */

.orbit {
  position: absolute;
  left: 50%;
  top: 50%;

  border-radius: 50%;

  transform:
    translate(-50%, -50%)
    rotate(-18deg);

  transform-origin: center;
  z-index: 3;
}

.orbit-outer {
  width: 94%;
  height: 70%;

  border: 1.4px solid var(--orbit-blue);

  box-shadow:
    0 0 4px rgba(67, 187, 255, 0.12),
    inset 0 0 4px rgba(67, 187, 255, 0.05);
}

.orbit-middle {
  width: 72%;
  height: 51%;

  border: 1px solid var(--orbit-cyan);

  box-shadow: 0 0 5px rgba(83, 225, 255, 0.05);
}

.orbit-inner {
  width: 49%;
  height: 34%;

  border: 1px dashed var(--orbit-gold);

  box-shadow: 0 0 5px rgba(255, 189, 81, 0.05);
}

/* 让轨道部分区域变亮，增加空间层次 */
.orbit-outer::after,
.orbit-middle::after {
  content: "";

  position: absolute;
  inset: -1px;

  border-radius: inherit;
  border-top: 1px solid rgba(141, 224, 255, 0.5);
  border-left: 1px solid transparent;
  border-right: 1px solid transparent;
  border-bottom: 1px solid transparent;

  filter: drop-shadow(0 0 3px rgba(77, 193, 255, 0.2));
}

/* =========================
   行星
   ========================= */

.planet {
  position: absolute;

  display: block;

  border-radius: 50%;

  z-index: 9;
}

/* 外轨道右上行星 */
.planet-blue {
  top: -8%;
  right: 12%;

  width: 12%;
  aspect-ratio: 1;

  background: radial-gradient(
    circle at 32% 28%,
    #cbf1ff,
    #59bdff 45%,
    #247ed1 100%
  );

  box-shadow:
    0 0 3px rgba(91, 193, 255, 0.55),
    0 0 5px rgba(63, 169, 255, 0.28);
}

/* 中轨道右下行星 */
.planet-cyan {
  right: -5%;
  bottom: 13%;

  width: 15%;
  aspect-ratio: 1;

  background: radial-gradient(
    circle at 32% 28%,
    #d6fcff,
    #6fe2f4 48%,
    #249cc1 100%
  );

  box-shadow:
    0 0 4px rgba(107, 228, 249, 0.5),
    0 0 8px rgba(60, 204, 235, 0.22);
}

/* 内轨道左下小行星 */
.planet-orange {
  left: -7%;
  bottom: 5%;

  width: 18%;
  aspect-ratio: 1;

  background: radial-gradient(
    circle at 32% 28%,
    #fff0c2,
    #ffbd63 48%,
    #d77a29 100%
  );

  box-shadow:
    0 0 3px rgba(255, 194, 99, 0.5),
    0 0 6px rgba(255, 153, 44, 0.22);
}

/* =========================
   背景小星点
   ========================= */

.star-dot {
  position: absolute;

  width: 2px;
  height: 2px;

  border-radius: 50%;

  background: #8bdcff;

  box-shadow: 0 0 5px rgba(93, 208, 255, 0.4);

  opacity: 0.55;
}

.star-dot-1 {
  left: 8%;
  top: 24%;
}

.star-dot-2 {
  right: 3%;
  top: 49%;

  width: 1px;
  height: 1px;
}

.star-dot-3 {
  left: 55%;
  bottom: 5%;

  background: #ffd184;
  box-shadow: 0 0 5px rgba(255, 190, 87, 0.4);
}

/* =========================
   标题
   ========================= */

.solar-title {
  position: relative;

  color: var(--text-color);

  font-size: 12px;
  font-weight: 500;
  line-height: 1;
  letter-spacing: 0.08em;
  white-space: nowrap;

  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.4);

  transition:
    color 220ms ease,
    text-shadow 220ms ease;
}

.solar-system-item:not(.is-active):hover .solar-title,
.solar-system-item:not(.is-active):focus-visible .solar-title {
  color: #eef8ff;

  text-shadow:
    0 0 10px rgba(100, 202, 255, 0.2),
    0 0 18px rgba(65, 160, 255, 0.1);
}

/* =========================
   动画
   ========================= */

.is-animated .orbit-outer {
  animation: orbitRotateOuter 9s linear infinite;
}

.is-animated .orbit-middle {
  animation: orbitRotateMiddle 7s linear infinite reverse;
}

.is-animated .orbit-inner {
  animation: orbitRotateInner 5s linear infinite;
}

.is-animated .sun {
  animation: sunPulse 2.8s ease-in-out infinite;
}

.is-animated .star-dot {
  animation: starBlink 2.4s ease-in-out infinite;
}

.is-animated .star-dot-2 {
  animation-delay: 0.7s;
}

.is-animated .star-dot-3 {
  animation-delay: 1.3s;
}

@keyframes orbitRotateOuter {
  from {
    transform:
      translate(-50%, -50%)
      rotate(-18deg);
  }

  to {
    transform:
      translate(-50%, -50%)
      rotate(342deg);
  }
}

@keyframes orbitRotateMiddle {
  from {
    transform:
      translate(-50%, -50%)
      rotate(-18deg);
  }

  to {
    transform:
      translate(-50%, -50%)
      rotate(342deg);
  }
}

@keyframes orbitRotateInner {
  from {
    transform:
      translate(-50%, -50%)
      rotate(-18deg);
  }

  to {
    transform:
      translate(-50%, -50%)
      rotate(342deg);
  }
}

@keyframes sunPulse {
  0%,
  100% {
    filter: brightness(1);
  }

  50% {
    filter: brightness(1.13);
  }
}

@keyframes starBlink {
  0%,
  100% {
    opacity: 0.3;
    transform: scale(0.8);
  }

  50% {
    opacity: 1;
    transform: scale(1.3);
  }
}

/* 用户系统开启"减少动态效果"时停止动画 */
@media (prefers-reduced-motion: reduce) {
  .solar-system-item *,
  .solar-system-item *::before,
  .solar-system-item *::after {
    animation: none !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
