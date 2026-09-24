import { useMutation } from "@tanstack/react-query"
import { ping } from "../services/omniproxy"

export function usePing() {
  return useMutation({ mutationFn: ping })
}
