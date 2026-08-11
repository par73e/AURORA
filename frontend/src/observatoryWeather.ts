/** Open-Meteo 使用的 WMO weather interpretation code（WMO 4677）中文展示。 */
export function conditionDescription(code: number) {
  switch (code) {
    case 0: return '晴朗'
    case 1: return '大致晴朗'
    case 2: return '局部多云'
    case 3: return '阴天'
    case 45:
    case 48: return '雾'
    case 51:
    case 53:
    case 55: return '毛毛雨'
    case 56:
    case 57: return '冻毛毛雨'
    case 61:
    case 63:
    case 65: return '降雨'
    case 66:
    case 67: return '冻雨'
    case 71:
    case 73:
    case 75: return '降雪'
    case 77: return '雪粒'
    case 80:
    case 81:
    case 82: return '阵雨'
    case 85:
    case 86: return '阵雪'
    case 95: return '雷暴'
    case 96:
    case 99: return '雷暴伴冰雹'
    default: return '天气状况未知'
  }
}

export function weatherGlyph(code: number) {
  if (code === 0 || code === 1) return '☼'
  if (code === 2 || code === 3) return '☁'
  if (code === 45 || code === 48) return '≋'
  if ([71, 73, 75, 77, 85, 86].includes(code)) return '❄'
  if ([95, 96, 99].includes(code)) return 'ϟ'
  if ([51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82].includes(code)) return '☂'
  return '·'
}
