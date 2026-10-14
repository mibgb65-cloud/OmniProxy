import { AppService } from "../../bindings/omniproxy/internal/transport/wails"
export type {
  Options as StepFunOptions,
  Balance as StepFunBalance,
} from "../../bindings/omniproxy/internal/provider/stepfun"

export function openStepConsole(accountId: string) {
  return AppService.OpenStepConsole(accountId)
}
