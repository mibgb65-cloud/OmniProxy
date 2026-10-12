import type { ComponentProps } from "react"
import { Check, ChevronDown, ChevronUp } from "lucide-react"
import { Select } from "radix-ui"
import { cn } from "../../lib/utils"

export type SelectOption = { value: string; label: string; disabled?: boolean }
type SelectFieldProps = Omit<ComponentProps<typeof Select.Trigger>, "value" | "onChange"> & {
  value: string
  onValueChange: (value: string) => void
  options: SelectOption[]
  name?: string
  required?: boolean
  placeholder?: string
}

// 每个业务值都加前缀，保留空选项且不会与模型／账号 ID 中的任意字符串冲突。
const encode = (value: string) => `value:${value}`

export function SelectField({
  value,
  onValueChange,
  options,
  name,
  required,
  disabled,
  placeholder,
  className,
  ...props
}: SelectFieldProps) {
  return (
    <Select.Root
      value={encode(value)}
      onValueChange={(next) => {
        if (next.startsWith("value:")) onValueChange(next.slice(6))
      }}
      name={name}
      required={required}
      disabled={disabled}
    >
      <Select.Trigger
        data-slot="select-trigger"
        {...props}
        className={cn(
          "group border-input bg-background text-foreground hover:bg-accent/30 focus-visible:border-ring focus-visible:ring-ring/40 aria-invalid:border-destructive aria-invalid:focus-visible:ring-destructive/20 dark:bg-input/20 flex h-9 w-full min-w-0 cursor-pointer items-center justify-between gap-3 rounded-lg border px-3 text-left text-sm shadow-xs transition-colors outline-none focus-visible:ring-3 disabled:cursor-not-allowed disabled:opacity-50 motion-reduce:transition-none [&>[data-slot=select-value]]:min-w-0 [&>[data-slot=select-value]]:flex-1 [&>[data-slot=select-value]]:truncate",
          className,
        )}
      >
        <Select.Value data-slot="select-value" placeholder={placeholder}>
          <span data-slot="select-label" className="block truncate">
            {options.find((option) => option.value === value)?.label ?? placeholder ?? value}
          </span>
        </Select.Value>
        <Select.Icon asChild>
          <ChevronDown
            aria-hidden="true"
            className="text-muted-foreground size-3.5 shrink-0 transition-transform duration-150 group-data-[state=open]:rotate-180 motion-reduce:transition-none"
          />
        </Select.Icon>
      </Select.Trigger>
      <Select.Portal>
        <Select.Content
          position="popper"
          sideOffset={6}
          collisionPadding={12}
          data-slot="select-content"
          className="bg-popover/95 text-popover-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 relative z-80 max-h-[min(20rem,var(--radix-select-content-available-height))] max-w-[calc(100vw-1.5rem)] min-w-[var(--radix-select-trigger-width)] overflow-hidden rounded-xl border p-1.5 shadow-[0_12px_40px_-12px_rgba(0,0,0,0.28)] backdrop-blur-xl motion-safe:duration-150"
        >
          <Select.ScrollUpButton className="text-muted-foreground flex h-6 items-center justify-center">
            <ChevronUp className="size-3.5" aria-hidden="true" />
          </Select.ScrollUpButton>
          <Select.Viewport className="max-h-[inherit] min-w-0 space-y-0.5">
            {options.map((option) => (
              <Select.Item
                key={option.value}
                value={encode(option.value)}
                disabled={option.disabled}
                data-value={option.value}
                className="text-foreground data-[highlighted]:bg-accent/80 relative flex min-h-9 cursor-pointer items-center rounded-lg py-2 pr-9 pl-3 text-sm transition-colors outline-none select-none data-[disabled]:cursor-not-allowed data-[disabled]:opacity-35 data-[state=checked]:bg-blue-500/8 data-[state=checked]:text-blue-700 motion-reduce:transition-none dark:data-[state=checked]:text-blue-300"
              >
                <Select.ItemText>
                  <span className="block wrap-anywhere whitespace-normal">{option.label}</span>
                </Select.ItemText>
                <Select.ItemIndicator className="absolute right-3 flex size-4 items-center justify-center">
                  <Check className="size-3.5" aria-hidden="true" />
                </Select.ItemIndicator>
              </Select.Item>
            ))}
          </Select.Viewport>
          <Select.ScrollDownButton className="text-muted-foreground flex h-6 items-center justify-center">
            <ChevronDown className="size-3.5" aria-hidden="true" />
          </Select.ScrollDownButton>
        </Select.Content>
      </Select.Portal>
    </Select.Root>
  )
}
