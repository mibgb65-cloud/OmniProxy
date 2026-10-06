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
  onSubmit,
  onBatchSubmit,
}: {
  providerId: string
  onSubmit: (input: CreateAccountInput) => Promise<void>
  onBatchSubmit: (input: CreateAccountsInput) => Promise<void>
}) {
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
    if (nameError || keyError) return
    savingRef.current = true
    setSaving(true)
    setSaveError(false)
    try {
      if (mode === "batch") await onBatchSubmit({ providerId, apiKeys: batchKeys })
      else await onSubmit({ providerId, name: name.trim(), apiKey: apiKey.trim() })
    } catch {
      setSaveError(true)
    } finally {
      savingRef.current = false
      setSaving(false)
    }
  }
  return {
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
