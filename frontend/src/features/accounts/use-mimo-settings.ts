import { useIsMutating, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { getMiMoSettings, setMiMoPreferredBillingMode } from "../../services/mimo-settings"

export function useMiMoSettings() {
  const client = useQueryClient()
  const saving = useIsMutating({ mutationKey: ["mimo-settings"] }) > 0
  const settings = useQuery({ queryKey: ["mimo-settings"], queryFn: getMiMoSettings })
  const save = useMutation({
    mutationKey: ["mimo-settings"],
    mutationFn: setMiMoPreferredBillingMode,
    onSuccess: async (data) => {
      await client.cancelQueries({ queryKey: ["mimo-settings"] })
      client.setQueryData(["mimo-settings"], data)
      await Promise.all(
        ["routes", "default-routes", "gateway-runtime", "agent-clients"].map((key) =>
          client.invalidateQueries({ queryKey: [key] }),
        ),
      )
    },
  })
  return { settings, save, saving }
}
