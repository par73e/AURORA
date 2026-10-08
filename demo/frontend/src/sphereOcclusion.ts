import * as THREE from 'three'

/**
 * 视线-球体遮挡判定：相机到目标点的线段是否被以 center 为球心、radius 为半径的球体挡住。
 *
 * 全站统一的球面遮挡判据（地球 ORBIT、月球、火星、行星特写、土卫六共用），
 * 用来决定球面标记与标签在背面时是否隐藏。
 *
 * 不要退化成「法线与相机方向的点积 > 固定阈值」。真实的可见边界是 n·û = R/d
 * （û 为球心指向相机的单位向量，d 为相机距离，R 为球体半径），阈值随距离变化；
 * 写死常数只在某一个特定距离上成立，拉近镜头后会把球体背面的点误判为可见，
 * 表现为「圆点被球体挡住、标签却还浮在上面」。
 *
 * @param margin 可选的掠射余量：把判定球半径放大，使球外标记在贴近轮廓时
 *   更早隐藏，避免圆点与球面重叠。火星沿用 0.04 的圆点余量。
 */
export function isOccludedBySphere(
  worldPoint: THREE.Vector3,
  cameraPosition: THREE.Vector3,
  center: THREE.Vector3,
  radius: number,
  margin = 0,
): boolean {
  // 点在球内时射线-球体判定会漏判（最近点越过目标点），必须单独处理。
  // 用 radius 而非 radius+margin：表面点（r == radius）不能被误判为球内，
  // 故留 1e-3 浮点余量（cos²+sin² 计算的表面点可能略小于 radius）。
  if (worldPoint.distanceTo(center) < radius - 1e-3) return true

  const toTarget = worldPoint.clone().sub(cameraPosition)
  const targetDistance = toTarget.length()
  if (targetDistance === 0) return false
  toTarget.divideScalar(targetDistance)
  // 球心在视线上的投影参数；落在相机与目标点之外时，球体不在这段视线上
  const t = center.clone().sub(cameraPosition).dot(toTarget)
  if (t <= 0 || t >= targetDistance) return false
  const closest = cameraPosition.clone().addScaledVector(toTarget, t)
  return closest.distanceTo(center) < radius + margin
}
