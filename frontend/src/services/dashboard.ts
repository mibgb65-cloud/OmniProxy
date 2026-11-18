import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type {
  DashboardInfo,
  DashboardModel,
  DashboardPoint,
  DashboardQuery,
  DashboardSummary,
  DashboardProvider,
} from "../../bindings/omniproxy/internal/application/activity"

export type {
  DashboardInfo,
  DashboardModel,
  DashboardPoint,
  DashboardQuery,
  DashboardSummary,
  DashboardProvider,
}
export const readDashboard = (query: DashboardQuery): Promise<DashboardInfo> =>
  AppService.DashboardRead(query)
