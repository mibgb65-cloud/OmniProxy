import { useIsMutating, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { getMiMoSettings, setMiMoPreferredBillingMode } from "../../services/mimo-settings"
import { getStepFunSettings, setStepFunPreferredBillingMode } from "../../services/stepfun-settings"
import { getKimiSettings, setKimiPreferredBillingMode } from "../../services/kimi-settings"

export type BillingOrderProvider = "mimo" | "stepfun" | "kimi"

const services = {
  kimi: { read: getKimiSettings, save: setKimiPreferredBillingMode },
  mimo: { read: getMiMoSettings, save: setMiMoPreferredBillingMode },
  stepfun: { read: getStepFunSettings, save: setStepFunPreferredBillingMode },
}

export function useBillingOrder(provider: BillingOrderProvider) {
  const client = useQueryClient()
  const key = [`${provider}-settings`]
  const saving = useIsMutating({ mutationKey: key }) > 0
  const settings = useQuery({ queryKey: key, queryFn: services[provider].read })
  const save = useMutation({
    mutationKey: key,
    mutationFn: services[provider].save,
    onSuccess: async (data) => {
      await client.cancelQueries({ queryKey: key })
      client.setQueryData(key, data)
      await Promise.all(
        ["routes", "default-routes", "gateway-runtime", "agent-clients"].map((key) =>
          client.invalidateQueries({ queryKey: [key] }),
        ),
      )
    },
  })
  return { settings, save, saving }
}
