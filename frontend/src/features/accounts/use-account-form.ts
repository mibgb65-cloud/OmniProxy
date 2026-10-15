import type { MiniMaxOptions } from "../../services/minimax"
import { validMiniMaxOptions } from "../../services/minimax"
import type { KimiOptions } from "../../services/kimi"
import type { StepFunOptions as Options } from "../../services/stepfun"
import { useMemo, useRef, useState } from "react"
import type { FormEvent } from "react"
import type { CreateAccountInput, CreateAccountsInput } from "../../services/omniproxy"
import {
  MAX_BATCH_KEYS,
  MAX_BATCH_TEXT_BYTES,
  splitAPIKeys,
  validAccountName,
  validAPIKey,
} from "./account-input"

export function useAccountForm({
  providerId,
  stepEnabled = false,
  kimiEnabled = false,
  minimaxEnabled = false,
  onSubmit,
  onBatchSubmit,
}: {
  providerId: string
  stepEnabled?: boolean
  kimiEnabled?: boolean
  minimaxEnabled?: boolean
  onSubmit: (input: CreateAccountInput) => Promise<void>
  onBatchSubmit: (input: CreateAccountsInput) => Promise<void>
}) {
  const [minimax, setMiniMax] = useState<MiniMaxOptions>({
    region: "cn",
    billingMode: "payg",
    version: 0,
  })
  const [invalidMiniMax, setInvalidMiniMax] = useState(false)
  const [kimi, setKimi] = useState<KimiOptions>({ region: "cn", version: 0 })
  const [stepfun, setStepFun] = useState<Options>({ region: "cn", billingMode: "payg", version: 0 })
  const [mode, setMode] = useState<"single" | "batch">("single")
  const [name, setName] = useState("")
  const [apiKey, setAPIKey] = useState("")
  const [batchText, setBatchText] = useState("")
  const [invalidName, setInvalidName] = useState(false)
  const [invalidKey, setInvalidKey] = useState(false)
  const [saveError, setSaveError] = useState(false)
  const [saving, setSaving] = useState(false)
  const savingRef = useRef(false)
  const batchKeys = useMemo(() => splitAPIKeys(batchText), [batchText])
  const changeMode = (value: string) => {
    if (savingRef.current || (value !== "single" && value !== "batch")) return
    setMode(value)
    setInvalidName(false)
    setInvalidKey(false)
    setSaveError(false)
  }
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (savingRef.current) return
    const nameError = mode === "single" && !validAccountName(name)
    const keyError =
      mode === "single"
        ? !validAPIKey(apiKey)
        : batchKeys.length === 0 ||
          batchKeys.length > MAX_BATCH_KEYS ||
          new TextEncoder().encode(batchText).length > MAX_BATCH_TEXT_BYTES ||
          batchKeys.some((key) => !validAPIKey(key))
    setInvalidName(nameError)
    setInvalidKey(keyError)
    const minimaxError = minimaxEnabled && !validMiniMaxOptions(minimax)
    setInvalidMiniMax(minimaxError)
    if (nameError || keyError || minimaxError) return
    savingRef.current = true
    setSaving(true)
    setSaveError(false)
    try {
      const minimaxOptions = minimaxEnabled ? { minimax } : {}
      const stepOptions = stepEnabled ? { stepfun } : {}
      const kimiOptions = kimiEnabled ? { kimi } : {}
      if (mode === "batch")
        await onBatchSubmit({
          providerId,
          apiKeys: batchKeys,
          ...stepOptions,
          ...kimiOptions,
          ...minimaxOptions,
        })
      else
        await onSubmit({
          providerId,
          name: name.trim(),
          apiKey: apiKey.trim(),
          ...stepOptions,
          ...kimiOptions,
          ...minimaxOptions,
        })
    } catch {
      setSaveError(true)
    } finally {
      savingRef.current = false
      setSaving(false)
    }
  }
  return {
    minimax,
    setMiniMax,
    invalidMiniMax,
    kimi,
    setKimi,
    stepfun,
    setStepFun,
    mode,
    changeMode,
    name,
    setName,
    apiKey,
    setAPIKey,
    batchText,
    setBatchText,
    batchCount: batchKeys.length,
    invalidName,
    invalidKey,
    saveError,
    saving,
    submit,
  }
}
