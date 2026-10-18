import { useMutation, useQueryClient } from "@tanstack/react-query"
import { importLegacyData, previewLegacyImport } from "../../services/legacy-import"

export function useLegacyImport() {
  const client = useQueryClient()
  const preview = useMutation({ mutationFn: previewLegacyImport })
  const apply = useMutation({
    mutationFn: ({ directory, snapshot }: { directory: string; snapshot: string }) =>
      importLegacyData(directory, snapshot),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: ["accounts"] }),
        client.invalidateQueries({ queryKey: ["storage-info"] }),
        client.invalidateQueries({ queryKey: ["providers"] }),
      ])
    },
  })
  return { preview, apply }
}
