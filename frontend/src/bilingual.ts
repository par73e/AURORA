/** 中英文名称统一逻辑（用户要求，应用于所有中英文并存的文本）：
 *  中国对象 → 中文为主：中文（英文）
 *  外国对象 → 英文为主：English（中文）
 *  中性对象（行星/太阳/月球等通用天体）→ 中文为主（应用以中文为第一语言） */

/** 中国主体特征：中国机构 / 中国发射场 / 中文任务名 */
const CHINA_HINTS =
  /中国|CNSA|CAS|中国科学院|国家航天局|中国空间|CLTC|CAST|航天科技|航天科工|酒泉|西昌|太原|文昌|长征|神舟|天舟|天宫|问天|梦天|天问|嫦娥|鹊桥|祝融|羲和|夸父|北斗|风云|高分|遥感|实践|吉林|长光|蓝箭|星际荣耀/

export function isChineseOrigin(text: string): boolean {
  return CHINA_HINTS.test(text ?? '')
}

/** 双语名称：primary 主名 + secondary 括号备注。
 *  chinese 三种形态：
 *    boolean → 强制（true=中文主；false=英文主，中性天体传 true）；
 *    string  → 按运营方/名称判断（中国机构/发射场/任务名 → 中文主）；
 *    缺省    → 按名称本身判断。 */
export function bilingualName(zh: string, en: string, chinese?: boolean | string): { primary: string; secondary: string } {
  // 去重/缺省保护：同名或单语 → 只显示主名（避免 "CAPSTONE（CAPSTONE）" 类重复）
  if (!en || en === zh) return { primary: zh, secondary: '' }
  if (!zh) return { primary: en, secondary: '' }
  const zhPrimary = typeof chinese === 'boolean' ? chinese : isChineseOrigin(chinese || zh)
  if (zhPrimary) return { primary: zh, secondary: en }
  return { primary: en, secondary: zh }
}
