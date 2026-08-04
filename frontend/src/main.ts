import { createApp } from 'vue'
import App from './App.vue'
import './style.css'

// 路径归一化：无论以什么路径打开（旧大写 /AURORA/、拼凑的 /aurora/AURORA/、无尾斜杠等
// 脏路径，Vite SPA fallback 都会返回应用），统一替换为 base 路径，仅保留 hash 路由，
// 避免地址栏出现错误的 /aurora/AURORA/#... 形式
{
  // 部分沙箱/iframe 环境禁止 JS 导航 API（replaceState 会抛 SecurityError）——
  // 归一化失败时静默降级，应用照常按 hash 路由运行
  try {
    const base = import.meta.env.BASE_URL // 以 / 结尾，如 '/aurora/'
    const path = window.location.pathname.endsWith('/')
      ? window.location.pathname
      : window.location.pathname + '/'
    if (path !== base) {
      window.history.replaceState(null, '', base + window.location.hash)
    }
  } catch {
    /* 归一化不可用时忽略：hash 路由仍可正常工作 */
  }
}

createApp(App).mount('#app')
