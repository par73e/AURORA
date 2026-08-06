/** 中英文名称统一逻辑（用户最终决定，应用于所有中英文并存的文本）：
 *  全部中文为主，外国对象附英文括号注释：中文（English）
 *  （先前的"外国=英文主"方案用户试看后否决——恢复中文第一语言） */

/** 双语名称：primary=中文主名 + secondary=英文括号注释。
 *  同名/单语去重：中文与英文相同或缺失时只显示主名（避免 "CAPSTONE（CAPSTONE）" 类重复）。 */
export function bilingualName(zh: string, en: string): { primary: string; secondary: string } {
  if (!en || en === zh) return { primary: zh, secondary: '' }
  if (!zh) return { primary: en, secondary: '' }
  return { primary: zh, secondary: en }
}
