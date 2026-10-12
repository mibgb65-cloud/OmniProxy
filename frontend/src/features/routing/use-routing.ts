import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { getGatewayInfo } from "../../services/gateway"
import { listModelCatalog } from "../../services/models"
import { listAccounts, listProviders } from "../../services/omniproxy"
import {
  createRoute,
  deleteRoute,
  listRoutes,
  listDefaultRoutes,
  setModelPreferredProvider,
  previewRoute,
  setRouteEnabled,
  updateRoute,
  type RouteInput,
} from "../../services/routing"

export function useRouting() {
  const client = useQueryClient()
  const routes = useQuery({ queryKey: ["routes"], queryFn: listRoutes })
  const providers = useQuery({ queryKey: ["providers"], queryFn: listProviders })
  const accounts = useQuery({ queryKey: ["accounts"], queryFn: listAccounts })
  const gateway = useQuery({ queryKey: ["gateway-runtime"], queryFn: getGatewayInfo })
  const catalog = useQuery({ queryKey: ["model-catalog"], queryFn: listModelCatalog })
  const defaults = useQuery({
    queryKey: [
      "default-routes",
      providers.dataUpdatedAt,
      accounts.dataUpdatedAt,
      catalog.dataUpdatedAt,
    ],
    queryFn: listDefaultRoutes,
    enabled: providers.isSuccess && accounts.isSuccess && catalog.isSuccess,
    placeholderData: keepPreviousData,
  })
  async function invalidate() {
    await Promise.all([
      client.invalidateQueries({ queryKey: ["routes"] }),
      client.invalidateQueries({ queryKey: ["default-routes"] }),
      client.invalidateQueries({ queryKey: ["gateway-runtime"] }),
      client.invalidateQueries({ queryKey: ["agent-clients"] }),
    ])
  }
  const create = useMutation({
    mutationFn: (input: RouteInput) => createRoute(input),
    onSuccess: invalidate,
  })
  const update = useMutation({
    mutationFn: ({ id, input }: { id: string; input: RouteInput }) => updateRoute(id, input),
    onSuccess: invalidate,
  })
  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => setRouteEnabled(id, enabled),
    onSuccess: invalidate,
  })
  const remove = useMutation({ mutationFn: (id: string) => deleteRoute(id), onSuccess: invalidate })
  const preview = useMutation({ mutationFn: (model: string) => previewRoute(model) })
  const preference = useMutation({
    mutationFn: ({ model, providerId }: { model: string; providerId: string }) =>
      setModelPreferredProvider(model, providerId),
    onMutate: () => preview.reset(),
    onSuccess: invalidate,
  })
  async function refresh() {
    preview.reset()
    await Promise.all([
      routes.refetch(),
      providers.refetch(),
      accounts.refetch(),
      gateway.refetch(),
      catalog.refetch(),
      defaults.refetch(),
    ])
  }
  return {
    routes,
    providers,
    accounts,
    gateway,
    catalog,
    defaults,
    preference,
    create,
    update,
    toggle,
    remove,
    preview,
    refresh,
    isFetching:
      routes.isFetching ||
      providers.isFetching ||
      accounts.isFetching ||
      gateway.isFetching ||
      catalog.isFetching ||
      defaults.isFetching,
  }
}
export type RoutingData = ReturnType<typeof useRouting>
