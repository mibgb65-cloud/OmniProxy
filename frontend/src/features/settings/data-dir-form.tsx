import { zodResolver } from "@hookform/resolvers/zod"
import { FolderOpen, Info } from "lucide-react"
import { useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { z } from "zod"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { formatBytes } from "../../lib/format"
import { useSelectDirectory, useSetDataDir, useStorageInfo } from "./use-storage"

// The backend performs the authoritative validation (absolute path,
// writability, nesting); this schema only guards the obvious cases.
const schema = z.object({
  path: z.string().trim().min(1, "required"),
})

type FormValues = z.infer<typeof schema>

export function DataDirForm() {
  const { t } = useTranslation()
  const { data: info } = useStorageInfo()
  const setDataDir = useSetDataDir()
  const selectDirectory = useSelectDirectory()

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { path: info?.dataDir ?? "" },
  })

  const browse = () => {
    selectDirectory.mutate(t("settings.storage.selectTitle"), {
      onSuccess: (path) => {
        if (path) {
          setValue("path", path, { shouldValidate: true })
        }
      },
    })
  }

  const onSubmit = (values: FormValues) => {
    setDataDir.mutate(values.path, {
      onError: (error) => setError("path", { message: error.message }),
    })
  }

  return (
    <Card className="gap-0 rounded-2xl py-4 shadow-none">
      <CardHeader className="px-5 pb-4">
        <CardTitle className="text-base">{t("settings.storage.locationTitle")}</CardTitle>
        <CardDescription>{t("settings.storage.locationDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="@container space-y-4 px-5">
        <form
          className="flex flex-col gap-2 @lg:flex-row @lg:items-start"
          onSubmit={handleSubmit(onSubmit)}
          noValidate
        >
          <div className="min-w-0 flex-1">
            <label className="sr-only" htmlFor="data-dir">
              {t("settings.storage.pathLabel")}
            </label>
            <Input
              id="data-dir"
              className="font-mono text-xs"
              placeholder={t("settings.storage.pathPlaceholder")}
              aria-invalid={errors.path ? true : undefined}
              {...register("path")}
            />
            {errors.path ? (
              <p className="text-destructive mt-1 text-xs">
                {errors.path.message === "required"
                  ? t("settings.storage.error.required")
                  : errors.path.message}
              </p>
            ) : null}
          </div>

          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={browse}
              disabled={selectDirectory.isPending}
            >
              <FolderOpen aria-hidden="true" />
              {t("settings.storage.browse")}
            </Button>
            <Button
              type="submit"
              className="bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-600 dark:hover:bg-blue-700"
              disabled={setDataDir.isPending}
            >
              {setDataDir.isPending ? t("settings.storage.saving") : t("settings.storage.save")}
            </Button>
          </div>
        </form>

        {selectDirectory.isError ? (
          <p className="text-destructive text-xs" role="alert">
            {t("settings.storage.browseFailed", { error: selectDirectory.error.message })}
          </p>
        ) : null}

        {setDataDir.isSuccess ? (
          <p className="bg-muted text-muted-foreground rounded-md px-3 py-2 text-sm" role="status">
            {t("settings.storage.saved", {
              size: formatBytes(setDataDir.data.existingDataSize),
            })}
          </p>
        ) : null}

        <p className="flex gap-2 rounded-md border border-blue-200 bg-blue-50 px-3 py-2.5 text-xs text-blue-700 dark:border-blue-500/25 dark:bg-blue-500/5 dark:text-blue-300">
          <Info className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <span>{t("settings.storage.migrationHint")}</span>
        </p>
      </CardContent>
    </Card>
  )
}
