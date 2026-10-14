import { useRef, useState } from "react"
import type { FormEvent } from "react"
import type { AccountItem, UpdateAccountInput } from "../../services/omniproxy"
import type { KimiOptions } from "../../services/kimi"
import type { StepFunOptions as Options } from "../../services/stepfun"
import { validAccountName, validAPIKey } from "./account-input"
import { validStepOptions } from "./stepfun-input"

export function useAccountEditForm(
  account: AccountItem,
  onSubmit: (input: UpdateAccountInput) => Promise<void>,
) {
  const kimiEnabled = Boolean(account.kimi || account.providerId === "builtin_kimi")
  const [kimi, setKimi] = useState<KimiOptions>(account.kimi ?? { region: "", version: 0 })
  const [invalidKimi, setInvalidKimi] = useState(false)
  const stepEnabled = Boolean(account.stepfun || account.providerId === "builtin_stepfun")
  const [stepfun, setStepFun] = useState<Options>(
    account.stepfun ?? { region: "", billingMode: "", version: 0 },
  )
  const [invalidStep, setInvalidStep] = useState(false)
  const [name, setName] = useState(account.name)
  const [apiKey, setAPIKey] = useState("")
  const [invalidName, setInvalidName] = useState(false)
  const [invalidKey, setInvalidKey] = useState(false)
  const [saveError, setSaveError] = useState(false)
  const [saving, setSaving] = useState(false)
  const savingRef = useRef(false)
  const nameRef = useRef<HTMLInputElement>(null)
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (savingRef.current) return
    const nameError = !validAccountName(name)
    const keyError = apiKey.trim() !== "" && !validAPIKey(apiKey)
    setInvalidName(nameError)
    setInvalidKey(keyError)
    const stepError = stepEnabled && !validStepOptions(stepfun)
    setInvalidStep(stepError)
    const kimiError = kimiEnabled && !["cn", "global"].includes(kimi.region)
    setInvalidKimi(kimiError)
    if (nameError || keyError || stepError || kimiError) return
    savingRef.current = true
    setSaving(true)
    setSaveError(false)
    try {
      await onSubmit({
        id: account.id,
        name: name.trim(),
        apiKey: apiKey.trim(),
        ...(stepEnabled ? { stepfun } : {}),
        ...(kimiEnabled ? { kimi } : {}),
      })
    } catch {
      setSaveError(true)
    } finally {
      savingRef.current = false
      setSaving(false)
    }
  }

  return {
    name,
    setName,
    apiKey,
    setAPIKey,
    invalidName,
    invalidKey,
    saveError,
    saving,
    nameRef,
    submit,
    kimiEnabled,
    kimi,
    setKimi,
    invalidKimi,
    stepEnabled,
    stepfun,
    setStepFun,
    invalidStep,
  }
}
