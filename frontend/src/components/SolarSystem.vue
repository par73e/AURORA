<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { SolarSystemScene, type SolarLabel } from '../solar/scene'
import { planets, SUN, type PlanetSpec } from '../solar/data'

const props = defineProps<{ enterFromOrbit?: boolean }>()

const emit = defineEmits<{
  'select-earth': []
  'earth-fly-start': []
  'earth-fly-zoom': []
}>()

const canvasHost = ref<HTMLDivElement | null>(null)
const activeId = ref('earth')
const labels = ref<SolarLabel[]>([])
let scene: SolarSystemScene | undefined

const activePlanet = computed<PlanetSpec>(() => planets.find((p) => p.id === activeId.value) ?? planets[2])
const activeName = computed(() => (activeId.value === 'sun' ? SUN.name : activePlanet.value.name))
const activeNameEn = computed(() => (activeId.value === 'sun' ? SUN.nameEn : activePlanet.value.nameEn))

const planetLabels = computed(() => labels.value.filter((l) => l.kind === 'planet'))
const sunLabel = computed(() => labels.value.find((l) => l.kind === 'sun') ?? null)
const beltLabels = computed(() => labels.value.filter((l) => l.kind === 'belt'))
const earthLabel = computed(() => labels.value.find((l) => l.kind === 'planet' && l.id === 'earth') ?? null)

const nameById = new Map(planets.map((p) => [p.id, p.name]))
const nameEnById = new Map(planets.map((p) => [p.id, p.nameEn]))

function choosePlanet(id: string) {
  activeId.value = id
  if (id === 'earth') {
    // 地球：先在太阳系场景内放大地球，飞行到位后再由 App 切换页面
    scene?.flyToEarth()
    emit('earth-fly-start')
  }
}

// 排列线（屏幕方向：右上 → 左下）的单位法向（右下），标签沿该方向放在球体轮廓之外，避免压到相邻行星；
// 数值按 28° 俯视、35° 方位角下的投影方向近似
const LABEL_OFFSET_X = 0.5
const LABEL_OFFSET_Y = 0.87

/** 标签放在球体轮廓之外：沿排列线法向（右下）偏移 */
function planetLabelStyle(label: SolarLabel) {
  const offset = label.radiusPx + 14
  return { transform: `translate(calc(${label.x + offset * LABEL_OFFSET_X}px - 50%), ${label.y + offset * LABEL_OFFSET_Y}px)` }
}

function sunLabelStyle(label: SolarLabel) {
  const offset = label.radiusPx + 16
  return { transform: `translate(calc(${label.x + offset * LABEL_OFFSET_X}px - 50%), ${label.y + offset * LABEL_OFFSET_Y}px)` }
}

function beltLabelStyle(label: SolarLabel) {
  return { transform: `translate(calc(${label.x}px - 50%), ${label.y - 12}px)` }
}

/** YOU ARE HERE 标记：位于地球正上方，底部小箭头指向地球顶端 */
function youMarkerStyle(label: SolarLabel) {
  return { transform: `translate(calc(${label.x}px - 50%), ${label.y - label.radiusPx - 27}px)` }
}

onMounted(() => {
  if (!canvasHost.value) return
  scene = new SolarSystemScene(
    canvasHost.value,
    {
      onHover: (id) => {
        if (id) activeId.value = id
      },
      onSelect: choosePlanet,
      onFlyZoom: () => emit('earth-fly-zoom'),
      onFlyComplete: () => emit('select-earth'),
    },
    (next) => {
      labels.value = next
    },
  )
  // 从 ORBIT 返回：镜头从地球近景拉回默认构图（地球缩回太阳系）
  if (props.enterFromOrbit) scene.flyFromEarth()
})

onBeforeUnmount(() => {
  scene?.dispose()
})
</script>

<template>
  <section class="solar-system" aria-labelledby="solar-system-title">
    <header class="solar-intro">
      <p>SOLAR SYSTEM · OBSERVATORY</p>
      <h1 id="solar-system-title">太阳系</h1>
    </header>

    <div ref="canvasHost" class="solar-scene-host" role="group" aria-label="太阳系示意排布场景，点击地球可进入 ORBIT 观测界面">
      <button
        v-for="label in planetLabels"
        v-show="label.visible"
        :key="label.id"
        class="solar-label planet-label"
        :class="{ earth: label.id === 'earth', active: activeId === label.id }"
        :style="planetLabelStyle(label)"
        :aria-label="label.id === 'earth' ? '进入地球 ORBIT 观测界面' : `${nameById.get(label.id)}（${nameEnById.get(label.id)}）`"
        @mouseenter="activeId = label.id"
        @focus="activeId = label.id"
        @click="choosePlanet(label.id)"
      >
        <strong>{{ nameById.get(label.id) }}</strong>
        <small>{{ nameEnById.get(label.id) }}</small>
        <i v-if="label.id === 'earth'" class="earth-entry">进入 ORBIT ↗</i>
      </button>

      <div
        v-if="sunLabel"
        v-show="sunLabel.visible"
        class="solar-label sun-label"
        :style="sunLabelStyle(sunLabel)"
        aria-hidden="true"
      >
        <strong>太阳</strong>
        <small>SUN</small>
      </div>

      <div
        v-for="belt in beltLabels"
        v-show="belt.visible"
        :key="belt.id"
        class="belt-label"
        :style="beltLabelStyle(belt)"
        aria-hidden="true"
      >
        {{ belt.id === 'asteroid-belt' ? '小行星带' : '柯伊伯带' }}
        <small>{{ belt.id === 'asteroid-belt' ? 'ASTEROID BELT' : 'KUIPER BELT' }}</small>
      </div>

      <div
        v-if="earthLabel"
        v-show="earthLabel.visible"
        class="you-marker"
        :style="youMarkerStyle(earthLabel)"
        aria-hidden="true"
      >
        <strong>YOU ARE HERE</strong>
        <svg viewBox="0 0 24 14" role="presentation">
          <path d="M12 1 V8" />
          <path d="M8 6 L12 11 L16 6" />
        </svg>
      </div>
    </div>

    <div class="solar-credits" aria-hidden="true">
      <span>行星纹理 Solar System Scope · CC BY 4.0</span>
    </div>

    <footer class="solar-readout" aria-live="polite">
      <span>{{ activeNameEn }}</span>
      <strong>{{ activeName }}</strong>
    </footer>
  </section>
</template>

<style scoped>
.solar-system {
  position: relative;
  height: 100dvh;
  min-height: 680px;
  overflow: hidden;
  isolation: isolate;
  padding-top: 72px;
  color: #e8f2f5;
  background:
    radial-gradient(ellipse at 82% 10%, rgba(255, 169, 70, .085), transparent 16%),
    radial-gradient(ellipse at 45% 68%, rgba(24, 94, 133, .12), transparent 42%),
    linear-gradient(134deg, #010408 0%, #020812 50%, #010408 100%);
}

.solar-system::before {
  content: '';
  position: absolute;
  z-index: -2;
  inset: 72px 0 0;
  pointer-events: none;
  background: linear-gradient(156deg, transparent 36%, rgba(32, 96, 128, .035) 59%, transparent 60%);
}

.solar-intro { position: absolute; z-index: 4; top: 110px; left: 32px; }
.solar-intro p { margin: 0 0 9px; color: rgba(112, 179, 209, .72); font: 500 8px var(--font-mono); letter-spacing: .2em; }
.solar-intro h1 { margin: 0; font-size: 34px; font-weight: 400; letter-spacing: .08em; }

.solar-scene-host {
  position: absolute;
  inset: 72px 0 0;
  overflow: hidden;
  cursor: default;
}
.solar-scene-host canvas { display: block; }

.solar-label {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 4;
  display: grid;
  justify-items: center;
  gap: 3px;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  white-space: nowrap;
  pointer-events: none;
  appearance: none;
  text-shadow: 0 2px 8px rgba(0, 0, 0, .85);
}
.planet-label { pointer-events: auto; cursor: pointer; }
.planet-label strong { color: rgba(224, 235, 240, .88); font-size: 10px; font-weight: 500; letter-spacing: .08em; }
.planet-label small { color: rgba(104, 138, 153, .74); font: 400 7px var(--font-mono); letter-spacing: .12em; }
.planet-label:focus-visible { outline: 1px dashed rgba(115, 223, 255, .6); outline-offset: 4px; border-radius: 2px; }

.sun-label strong { color: rgba(255, 218, 159, .9); font-size: 10px; font-weight: 500; letter-spacing: .08em; }
.sun-label small { color: rgba(213, 147, 75, .7); font: 400 7px var(--font-mono); letter-spacing: .12em; }

.earth-entry {
  display: flex;
  gap: 5px;
  align-items: center;
  margin-top: 2px;
  color: rgba(126, 222, 252, .92);
  font: 500 7px var(--font-mono);
  letter-spacing: .1em;
  opacity: 0;
  transform: translateY(3px);
  transition: opacity .25s, transform .4s cubic-bezier(.16, 1, .3, 1);
}
.planet-label.earth:hover .earth-entry,
.planet-label.earth:focus-visible .earth-entry,
.planet-label.earth.active .earth-entry { opacity: 1; transform: translateY(0); }

.you-marker {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 6;
  display: grid;
  justify-items: center;
  gap: 2px;
  pointer-events: none;
}
.you-marker strong {
  color: rgba(190, 232, 247, .82);
  font: 500 7px var(--font-mono);
  letter-spacing: .14em;
  text-transform: uppercase;
  text-shadow: 0 1px 6px rgba(0, 0, 0, .85);
}
.you-marker svg { width: 22px; height: 13px; overflow: visible; }
.you-marker path {
  fill: none;
  stroke: rgba(149, 211, 235, .6);
  stroke-width: 0.9;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.belt-label {
  position: absolute;
  left: 0;
  top: 0;
  z-index: 3;
  color: rgba(143, 163, 170, .64);
  font-size: 8px;
  letter-spacing: .08em;
  text-align: center;
  pointer-events: none;
  text-shadow: 0 1px 6px rgba(0, 0, 0, .8);
}
.belt-label small { display: block; margin-top: 4px; color: rgba(76, 109, 123, .65); font: 400 6px var(--font-mono); letter-spacing: .1em; }

.solar-credits {
  position: absolute;
  z-index: 3;
  right: 34px;
  bottom: 30px;
  color: rgba(105, 146, 165, .54);
  font: 400 7px var(--font-mono);
  letter-spacing: .08em;
  text-align: right;
  pointer-events: none;
}

.solar-readout { position: absolute; z-index: 4; left: 32px; bottom: 28px; display: grid; grid-template-columns: 100px auto; align-items: end; column-gap: 18px; }
.solar-readout > span { grid-row: 1; color: rgba(92, 153, 179, .64); font: 500 8px var(--font-mono); letter-spacing: .15em; }
.solar-readout strong { grid-column: 1; grid-row: 2; margin-top: 5px; color: rgba(235, 242, 245, .9); font-size: 17px; font-weight: 500; }

@media (prefers-reduced-motion: no-preference) {
  .solar-intro, .solar-readout { animation: copy-arrive .7s .12s cubic-bezier(.16, 1, .3, 1) both; }
}

@keyframes copy-arrive { from { opacity: 0; filter: blur(5px); transform: translateY(12px); } }

@media (max-width: 1120px) {
  .solar-intro { left: 24px; }
  .solar-readout { left: 24px; }
  .solar-credits { right: 24px; }
}
</style>
