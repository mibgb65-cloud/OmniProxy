import type { StepFunOptions } from "../../services/stepfun"

export function validStepOptions(value: StepFunOptions) {
  return (
    ["cn", "global"].includes(value.region) && ["payg", "step_plan"].includes(value.billingMode)
  )
}
