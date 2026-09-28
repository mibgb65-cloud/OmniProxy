import { useTranslation } from "react-i18next"

export function ProviderSwitch({
  name,
  checked,
  disabled,
  onChange,
}: {
  name: string
  checked: boolean
  disabled: boolean
  onChange: (enabled: boolean) => void
}) {
  const { t } = useTranslation()
  return (
    <label className="flex cursor-pointer items-center has-disabled:cursor-default">
      <input
        type="checkbox"
        role="switch"
        aria-label={t("providers.toggleLabel", { name })}
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.currentTarget.checked)}
        className="peer sr-only"
      />
      <span
        className={`peer-focus-visible:ring-ring/50 relative h-6 w-11 shrink-0 rounded-full peer-focus-visible:ring-4 peer-disabled:opacity-50 ${checked ? "bg-blue-600" : "bg-input"}`}
        aria-hidden="true"
      >
        <span
          className={`absolute top-0.5 left-0.5 size-5 rounded-full bg-white shadow-sm motion-safe:transition-transform ${checked ? "translate-x-5" : "translate-x-0"}`}
        />
      </span>
    </label>
  )
}
