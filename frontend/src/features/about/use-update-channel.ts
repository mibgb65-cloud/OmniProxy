import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  getUpdatePreferences,
  setUpdateChannel,
  setAutomaticUpdateCheck,
  setAutomaticUpdateDownload,
} from "../../services/updates"

export function useUpdateChannel(onChanged: () => void) {
  const client = useQueryClient()
  const preferences = useQuery({ queryKey: ["update-preferences"], queryFn: getUpdatePreferences })
  const save = useMutation({
    mutationFn: setUpdateChannel,
    onSuccess: (data) => {
      client.setQueryData(["update-preferences"], data)
      void client.invalidateQueries({ queryKey: ["application-update"] })
      onChanged()
    },
  })
  const automatic = useMutation({
    mutationFn: setAutomaticUpdateCheck,
    onSuccess: (data) => client.setQueryData(["update-preferences"], data),
  })
  const download = useMutation({
    mutationFn: setAutomaticUpdateDownload,
    onSuccess: (data) => client.setQueryData(["update-preferences"], data),
  })
  return { preferences, save, automatic, download }
}
