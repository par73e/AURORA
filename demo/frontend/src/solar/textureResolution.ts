export function wantsDetail(pixels: number, current: boolean) {
  return pixels > (current ? 1400 : 1800)
}
