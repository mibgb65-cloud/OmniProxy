import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { Info } from "../../bindings/omniproxy/internal/application/gateway"

export type { Info as GatewayInfo }

/** 读取内存运行快照，不调用模型。 */
export function getGatewayInfo(): Promise<Info> {
  return AppService.GatewayInfo()
}
/** 启动当前环境的本机代理。 */
export function startGateway(): Promise<Info> {
  return AppService.GatewayStart()
}
/** 停止代理并取消在途请求。 */
export function stopGateway(): Promise<Info> {
  return AppService.GatewayStop()
}
