import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
import type {
  DefaultRoute,
  Input,
  Item,
  Preview,
} from "../../bindings/omniproxy/internal/application/routing"
import type { Rule } from "../../bindings/omniproxy/internal/core/routing"

export type {
  DefaultRoute as DefaultModelRoute,
  Input as RouteInput,
  Item as RouteItem,
  Preview as RoutePreview,
  Rule as RoutingRule,
}
export async function listDefaultRoutes(): Promise<DefaultRoute[]> {
  return (await AppService.ListDefaultRoutes()) ?? []
}
export function setModelPreferredProvider(model: string, providerId: string): Promise<void> {
  return AppService.SetModelPreferredProvider(model, providerId)
}
export async function listRoutes(): Promise<Item[]> {
  return (await AppService.ListRoutes()) ?? []
}
export function createRoute(input: Input): Promise<Rule> {
  return AppService.CreateRoute(input)
}
export function updateRoute(id: string, input: Input): Promise<Rule> {
  return AppService.UpdateRoute(id, input)
}
export function setRouteEnabled(id: string, enabled: boolean): Promise<Rule> {
  return AppService.SetRouteEnabled(id, enabled)
}
export function deleteRoute(id: string): Promise<void> {
  return AppService.DeleteRoute(id)
}
export function previewRoute(model: string): Promise<Preview> {
  return AppService.PreviewRoute(model)
}
