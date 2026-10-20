import { useEffect } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"

export function useModelCatalogEvents() {
  const client = useQueryClient()
  useEffect(
    () =>
      Events.On("modelcatalog:updated", () => {
        // 不刷新 Google 的发现查询：该查询本身会触发同步，反向失效会形成循环。
        for (const key of [
          "model-catalog",
          "default-routes",
          "routes",
          "agent-clients",
          "agent-configured-models",
        ]) {
          void client.invalidateQueries({ queryKey: [key] })
        }
      }),
    [client],
  )
}
