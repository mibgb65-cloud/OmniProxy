import { createFileRoute } from "@tanstack/react-router"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { PagePlaceholder } from "../../components/common/page-placeholder"
import { useAppInfo } from "../../hooks/use-app-info"
import { usePing } from "../../hooks/use-ping"

export const Route = createFileRoute("/")({
  component: DashboardPage,
})

function DashboardPage() {
  const { data, isPending, isError, error } = useAppInfo()
  const pingMutation = usePing()

  return (
    <PagePlaceholder
      title="Dashboard"
      description="OmniProxy runtime overview. Feature panels arrive with their backends."
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Application</CardTitle>
            <CardDescription>Runtime information from the Go backend</CardDescription>
          </CardHeader>
          <CardContent className="text-sm">
            {isPending ? <p className="text-muted-foreground">Loading…</p> : null}
            {isError ? (
              <p className="text-destructive">Failed to load app info: {error.message}</p>
            ) : null}
            {data ? (
              <dl className="grid grid-cols-2 gap-y-2">
                <dt className="text-muted-foreground">Version</dt>
                <dd className="text-right">{data.version}</dd>
                <dt className="text-muted-foreground">Profile</dt>
                <dd className="text-right">{data.profile}</dd>
                <dt className="text-muted-foreground">Proxy port</dt>
                <dd className="text-right">{data.proxyPort}</dd>
                <dt className="text-muted-foreground">Control port</dt>
                <dd className="text-right">{data.controlPort}</dd>
                <dt className="text-muted-foreground">Data directory</dt>
                <dd className="truncate text-right" title={data.dataDir}>
                  {data.dataDir}
                </dd>
              </dl>
            ) : null}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Binding check</CardTitle>
            <CardDescription>Wails typed binding round-trip</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <Button
              size="sm"
              onClick={() => pingMutation.mutate()}
              disabled={pingMutation.isPending}
            >
              {pingMutation.isPending ? "Pinging…" : "Ping backend"}
            </Button>
            {pingMutation.isError ? (
              <p className="text-destructive">Ping failed: {pingMutation.error.message}</p>
            ) : null}
            {pingMutation.isSuccess ? (
              <p className="text-muted-foreground">
                Backend replied:{" "}
                <span className="text-foreground font-medium">{pingMutation.data}</span>
              </p>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </PagePlaceholder>
  )
}
