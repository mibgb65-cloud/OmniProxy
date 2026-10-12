import { SelectField } from "../../components/ui/select-field"
import { Search } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import { logLevels, type LogLevel } from "./log-stream"

export function LogFilters({
  levels,
  onLevels,
  module,
  onModule,
  modules,
  keyword,
  onKeyword,
}: {
  levels: LogLevel[]
  onLevels: (levels: LogLevel[]) => void
  module: string
  onModule: (module: string) => void
  modules: string[]
  keyword: string
  onKeyword: (keyword: string) => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap items-center gap-3">
      <div role="group" aria-label={t("logs.levelFilter")} className="flex gap-1">
        {logLevels.map((level) => (
          <Button
            key={level}
            type="button"
            variant={levels.includes(level) ? "secondary" : "outline"}
            size="sm"
            aria-pressed={levels.includes(level)}
            onClick={() =>
              onLevels(
                levels.includes(level)
                  ? levels.filter((item) => item !== level)
                  : [...levels, level],
              )
            }
          >
            {level}
          </Button>
        ))}
      </div>
      <SelectField
        className="w-auto min-w-36"
        aria-label={t("logs.moduleFilter")}
        value={module}
        onValueChange={onModule}
        options={[
          { value: "", label: t("logs.allModules") },
          ...[...new Set([...modules, ...(module ? [module] : [])])]
            .sort()
            .map((value) => ({ value, label: value })),
        ]}
      />
      <div className="relative min-w-40 flex-1">
        <Search
          className="text-muted-foreground absolute top-2.5 left-3 size-4"
          aria-hidden="true"
        />
        <Input
          type="search"
          className="pl-9"
          maxLength={256}
          value={keyword}
          onChange={(event) => onKeyword(event.target.value)}
          aria-label={t("logs.search")}
          placeholder={t("logs.searchPlaceholder")}
        />
      </div>
    </div>
  )
}
