import { zodResolver } from "@hookform/resolvers/zod"
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

  const browse = async () => {
    const path = await selectDirectory.mutateAsync(t("settings.storage.selectTitle"))
    if (path) {
      setValue("path", path, { shouldValidate: true })
    }
  }

  const onSubmit = (values: FormValues) => {
    setDataDir.mutate(values.path, {
      onError: (error) => setError("path", { message: error.message }),
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.storage.locationTitle")}</CardTitle>
        <CardDescription>{t("settings.storage.locationDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="flex flex-wrap items-end gap-2"
          onSubmit={handleSubmit(onSubmit)}
          noValidate
        >
          <div className="min-w-72 flex-1">
            <label className="text-muted-foreground mb-1 block text-sm" htmlFor="data-dir">
              {t("settings.storage.pathLabel")}
            </label>
            <Input
              id="data-dir"
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

          <Button
            type="button"
            variant="outline"
            onClick={browse}
            disabled={selectDirectory.isPending}
          >
            {t("settings.storage.browse")}
          </Button>
          <Button type="submit" disabled={setDataDir.isPending}>
            {setDataDir.isPending ? t("settings.storage.saving") : t("settings.storage.save")}
          </Button>
        </form>

        {setDataDir.isSuccess ? (
          <p className="bg-muted text-muted-foreground mt-3 rounded-md px-3 py-2 text-sm">
            {t("settings.storage.saved", {
              size: formatBytes(setDataDir.data.existingDataSize),
            })}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
