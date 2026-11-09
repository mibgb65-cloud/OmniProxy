import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { Item, SyncStatus } from "../../bindings/omniproxy/internal/application/modelcatalog"

export type DiscoveredModel = Item
export type ModelCatalogSyncStatus = SyncStatus
export const listModelCatalog = () => AppService.ModelCatalog()
export const syncAccountModels = (id: string) => AppService.ModelSync(id)
export const getModelCatalogSyncStatus = () => AppService.ModelCatalogSyncStatus()
