import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { getUpdatePreferences, setUpdateChannel } from "../../services/updates"

export function useUpdateChannel(onChanged: () => void) {
  const client = useQueryClient()
  const preferences = useQuery({ queryKey: ["update-preferences"], queryFn: getUpdatePreferences })
  const save = useMutation({
    mutationFn: setUpdateChannel,
    onSuccess: (data) => {
      client.setQueryData(["update-preferences"], data)
      onChanged()
    },
  })
  return { preferences, save }
}
