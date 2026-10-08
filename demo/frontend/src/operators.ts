/** 运营方主机构（筛选/下拉用）：取联合署名第一个机构，并去掉中文括号注释；
 *  详情面板仍显示完整的 operatorName（联合署名）。 */
export function primaryOperator(operatorName: string): string {
  const first = operatorName.split(' / ')[0].trim()
  const paren = first.indexOf('（')
  const halfParen = first.indexOf('(')
  const cut = paren > 0 ? paren : halfParen > 0 ? halfParen : -1
  return cut > 0 ? first.slice(0, cut) : first
}
