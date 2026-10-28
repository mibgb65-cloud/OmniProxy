import {
  AppService,
  type AppInfo,
  type CleanResult,
  type DesktopSettings,
  type ProxyPortState,
  type ProxyPortUpdate,
  type RuntimeInfo,
  type SetDataDirResult,
  type StorageInfo,
  type UpdateStatus,
} from "../../bindings/omniproxy/internal/transport/wails"
import type {
  BatchCreateInput as CreateAccountsInput,
  CreateInput as CreateAccountInput,
  Item as AccountItem,
  UpdateInput as UpdateAccountInput,
  UpdateResult as UpdateAccountResult,
} from "../../bindings/omniproxy/internal/application/account"
import type {
  Configuration as ProviderConfiguration,
  CreateInput as CreateProviderInput,
  Definition as ProviderDefinition,
  ProbeResult as ProviderProbeResult,
  UpdateInput as UpdateProviderInput,
} from "../../bindings/omniproxy/internal/application/provider"
import type {
  Discovered as ProxyDiscovered,
  SaveInput as ProxySaveInput,
  Settings as ProxySettings,
  TestResult as ProxyTestResult,
} from "../../bindings/omniproxy/internal/application/proxysettings"

export type {
  AppInfo,
  CleanResult,
  DesktopSettings,
  ProxyPortState,
  ProxyPortUpdate,
  RuntimeInfo,
  SetDataDirResult,
  StorageInfo,
  UpdateStatus,
}
export type {
  AccountItem,
  CreateAccountInput,
  CreateAccountsInput,
  UpdateAccountInput,
  UpdateAccountResult,
}
export type {
  ProviderConfiguration,
  CreateProviderInput,
  ProviderDefinition,
  ProviderProbeResult,
  UpdateProviderInput,
}
export type { ProxyDiscovered, ProxySaveInput, ProxySettings, ProxyTestResult }

/** getAppInfo returns the application runtime information from the Go backend. */
export function getAppInfo(): Promise<AppInfo> {
  return AppService.AppInfo()
}

/** ping is a liveness probe against the Go backend. */
export function ping(): Promise<string> {
  return AppService.Ping()
}

/** getStorageInfo returns data storage locations and their sizes. */
export function getStorageInfo(): Promise<StorageInfo> {
  return AppService.StorageInfo()
}

/**
 * setDataDir persists a new data directory. The change takes effect on the
 * next start; existing data is moved when the app shuts down.
 */
export function setDataDir(path: string): Promise<SetDataDirResult> {
  return AppService.SetDataDir(path)
}

/** selectDirectory opens the native folder picker; empty string = cancelled. */
export function selectDirectory(title: string): Promise<string> {
  return AppService.SelectDirectory(title)
}

/**
 * cleanStorage runs a cleanup operation: "logs" removes rotated log backups,
 * "updates" removes the helper copies left by install attempts, and "database"
 * compacts the SQLite database.
 */
export function cleanStorage(target: "logs" | "updates" | "database"): Promise<CleanResult> {
  return AppService.CleanStorage(target)
}

/** getDesktopSettings reads the current OS autostart registration and close preference. */
export function getDesktopSettings(): Promise<DesktopSettings> {
  return AppService.DesktopSettings()
}

/** setAutoStart registers or removes this profile from the user's login startup. */
export function setAutoStart(enabled: boolean): Promise<boolean> {
  return AppService.SetAutoStart(enabled)
}

export function setGatewayAutoStart(enabled: boolean): Promise<boolean> {
  return AppService.SetGatewayAutoStart(enabled)
}

/** getProxyPort 读取本地网关端口、默认端口与占用状态。 */
export function getProxyPort(): Promise<ProxyPortState> {
  return AppService.ProxyPort()
}

/**
 * setProxyPort 保存并立即应用网关端口；端口被占用时后端自动避让到同区间可用端口，
 * 传 0 表示恢复默认端口。
 */
export function setProxyPort(port: number): Promise<ProxyPortUpdate> {
  return AppService.SetProxyPort(port)
}

/** setCloseBehavior persists how closing the main window should behave. */
export function setCloseBehavior(value: "exit" | "tray"): Promise<string> {
  return AppService.SetCloseBehavior(value)
}

/** getRuntimeInfo returns application and runtime details for the About page. */
export function getRuntimeInfo(): Promise<RuntimeInfo> {
  return AppService.RuntimeInfo()
}

/** checkForUpdate compares the current version against the latest release. */
export function checkForUpdate(): Promise<UpdateStatus> {
  return AppService.CheckForUpdate()
}

/** getProviderCatalog returns the built-in provider kinds and auth options. */
export async function getProviderCatalog(): Promise<ProviderDefinition[]> {
  return (await AppService.ProviderCatalog()) ?? []
}

/** listProviders returns the user's saved provider configurations. */
export async function listProviders(): Promise<ProviderConfiguration[]> {
  return (await AppService.ListProviders()) ?? []
}

/** createProvider saves one provider configuration. */
export function createProvider(input: CreateProviderInput): Promise<ProviderConfiguration> {
  return AppService.CreateProvider(input)
}

/** updateProvider changes the editable fields of one configuration. */
export function updateProvider(
  id: string,
  input: UpdateProviderInput,
): Promise<ProviderConfiguration> {
  return AppService.UpdateProvider(id, input)
}

/** setProviderEnabled switches a built-in or custom provider on or off. */
export function setProviderEnabled(id: string, enabled: boolean): Promise<ProviderConfiguration> {
  return AppService.SetProviderEnabled(id, enabled)
}

/** probeProvider checks endpoint reachability and header latency without credentials. */
export function probeProvider(id: string): Promise<ProviderProbeResult> {
  return AppService.ProbeProvider(id)
}

/** deleteProvider removes an unreferenced configuration. */
export function deleteProvider(id: string): Promise<void> {
  return AppService.DeleteProvider(id)
}

/** setProviderProxyMode 设置该提供商的出站策略：inherit | direct | proxy。 */
export function setProviderProxyMode(id: string, mode: string): Promise<ProviderConfiguration> {
  return AppService.SetProviderProxyMode(id, mode)
}

/** 只返回脱敏后的代理配置，密码留在凭据库中。 */
export function getProxySettings(): Promise<ProxySettings> {
  return AppService.GetProxySettings()
}

/** saveProxySettings 保存代理连接参数；密码只写入凭据库。 */
export function saveProxySettings(input: ProxySaveInput): Promise<ProxySettings> {
  return AppService.SaveProxySettings(input)
}

/** testProxySettings 探测端点可用的协议，不改变已保存设置。 */
export function testProxySettings(input: ProxySaveInput): Promise<ProxyTestResult> {
  return AppService.TestProxySettings(input)
}

/** discoverProxyPorts 探测本机常见代理端口。 */
export async function discoverProxyPorts(): Promise<ProxyDiscovered[]> {
  return (await AppService.DiscoverProxyPorts()) ?? []
}

/** 返回账号元数据；自动名称可能含 Key 前缀，密钥字段与凭证引用不返回。 */
export async function listAccounts(): Promise<AccountItem[]> {
  return (await AppService.ListAccounts()) ?? []
}

/** createAccount stores an API Key in protected local storage. */
export function createAccount(input: CreateAccountInput): Promise<AccountItem> {
  return AppService.CreateAccount(input)
}

/** 一次保存多个自动命名的账号，完整密钥留在受保护存储中。 */
export async function createAccounts(input: CreateAccountsInput): Promise<AccountItem[]> {
  return (await AppService.CreateAccounts(input)) ?? []
}

/** 修改名称及可选的新密钥；空密钥字段保持原凭证。 */
export function updateAccount(input: UpdateAccountInput): Promise<UpdateAccountResult> {
  return AppService.UpdateAccount(input)
}

/** setAccountEnabled switches one account on or off. */
export function setAccountEnabled(id: string, enabled: boolean): Promise<AccountItem> {
  return AppService.SetAccountEnabled(id, enabled)
}

/** deleteAccount removes account metadata and the protected credential. */
export function deleteAccount(id: string): Promise<void> {
  return AppService.DeleteAccount(id)
}

/** 只返回后端生成的密钥脱敏片段。 */
export function getAccountKeyPreview(id: string): Promise<string> {
  return AppService.AccountKeyPreview(id)
}

/** 桌面端直接复制；完整密钥不返回浏览器。 */
export function copyAccountKey(id: string): Promise<void> {
  return AppService.CopyAccountKey(id)
}
