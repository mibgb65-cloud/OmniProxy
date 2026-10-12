import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type { Item } from "../../bindings/omniproxy/internal/application/modelcatalog"

export type DiscoveredModel = Item
export const listModelCatalog = () => AppService.ModelCatalog()
export const syncAccountModels = (id: string) => AppService.ModelSync(id)
