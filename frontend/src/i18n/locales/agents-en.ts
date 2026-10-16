export const agentsEn = {
  "agents.title": "Local agent connections",
  "agents.codex": "Codex CLI / Desktop",
  "agents.claude": "Claude Code CLI",
  "agents.claude-desktop": "Claude Code Desktop",
  "agents.selectClient": "Choose a local agent, then open its configuration page to connect it.",
  "agents.codexDescription": "Shared connection settings for the CLI and desktop app.",
  "agents.claudeDescription": "Connect Claude Code in your terminal.",
  "agents.desktopDescription": "Connect the Code feature in the Claude desktop app.",
  "agents.open": "Open configuration",
  "agents.openClient": "Configure {{client}}",
  "agents.back": "Back to local agents",
  "agents.connectionSettings": "Connection settings",
  "agents.configPath": "Configuration file path",
  "agents.configFile": "Configuration file",
  "agents.fileExists": "Configuration file exists",
  "agents.fileMissing": "Created when configuration is applied",
  "agents.disabled": "Access key disabled",
  "agents.needsAttention": "Configuration needs attention",
  "agents.loading": "Loading…",
  "agents.unavailable": "Client information unavailable",
  "agents.loadFailed": "Unable to read client configuration. Refresh and try again.",
  "agents.working": "Working…",
  "agents.install.title": "Software installation",
  "agents.install.description":
    "Detect local installations and install the clients you need through Windows.",
  "agents.install.codexCli": "Codex CLI",
  "agents.install.codexDesktop": "Codex Desktop",
  "agents.install.claudeCode": "Claude Code CLI",
  "agents.install.claudeDesktop": "Claude Code Desktop",
  "agents.install.desktopCodeHint":
    "Official Claude Code Desktop uses the Code tab in the Claude desktop app and shares its installer with Chat. One-click configuration activates the local gateway. Quit Claude completely and restart, then open Code.",
  "agents.install.openCode": "Open Claude Code",
  "agents.install.checking": "Checking…",
  "agents.install.installed": "Installed",
  "agents.install.notInstalled": "Not installed",
  "agents.install.unknown": "Installation status unknown",
  "agents.install.installing": "Downloading and installing…",
  "agents.install.preparing": "Preparing installation…",
  "agents.install.downloading": "Downloading…",
  "agents.install.installingStage": "Installing…",
  "agents.install.progressLabel": "{{client}} installation progress",
  "agents.install.waitingProgress": "Waiting for installer progress",
  "agents.install.downloaded": "Current download {{downloaded}} / {{total}}",
  "agents.install.elapsed": "Elapsed {{time}}",
  "agents.install.phaseHint": "Shows the current stage when a percentage is unavailable.",
  "agents.install.verifying": "Verifying installation…",
  "agents.install.refresh": "Check again",
  "agents.install.install": "Install",
  "agents.install.installClient": "Install {{client}}",
  "agents.install.official": "Official guide",
  "agents.install.officialClient": "{{client}} official installation guide",
  "agents.install.agreements":
    "Installation uses Windows WinGet. Installing accepts the software and source license terms. Complete any Windows authorization prompts.",
  "agents.install.loadFailed": "Unable to read installation status. Check again.",
  "agents.install.error.installer_unavailable":
    "WinGet was not found. Install Windows App Installer or use the official guide to install manually.",
  "agents.install.error.detection_failed": "Detection failed. Check again before installing.",
  "agents.install.error.version_unavailable":
    "The program was found, but its version could not be confirmed. Check the client and try detection again.",
  "agents.install.error.install_failed":
    "Installation did not finish. Check your network or Windows prompts, then retry.",
  "agents.install.error.install_interrupted": "The installation result is uncertain. Check again.",
  "agents.install.error.verification_failed":
    "Installation ended, but a usable program has not been detected. Check again or consult the official guide.",
  "agents.install.error.busy": "Another installation is in progress. Try again later.",
  "agents.install.error.unsupported_platform":
    "Local detection and installation currently support Windows only.",
  "agents.install.error.official_unavailable": "Unable to open the official guide. Try again.",
  "agents.install.error.launch_failed":
    "Unable to open Claude Code. Check installation again or open Code from the client.",
  "agents.install.error.activation_required":
    "Apply one-click configuration to activate the local gateway first, then quit Claude completely and restart.",
  "agents.install.error.failed": "Operation failed. Check installation status and retry.",
  "agents.configure": "Configure",
  "agents.reread": "Reload configuration",
  "agents.rereadSuccess": "Current configuration reloaded. Preview again before applying.",
  "agents.rereadChangesNotice":
    "The current file was reloaded. Applying preserves its settings, merges gateway configuration, and creates a new encrypted backup. The previous backup is archived. Preview again before applying.",
  "agents.restore": "Restore configuration",
  "agents.copyKey": "Copy access key",
  "agents.keyCopied": "Access key copied.",
  "agents.disable": "Disable access key",
  "agents.enable": "Enable access key",
  "agents.configured": "Access key enabled",
  "agents.idle": "Not configured",
  "agents.provider": "Preserved provider ID",
  "agents.model": "Default model or route",
  "agents.choose": "Choose a model or route",
  "agents.defaultModel": "Default model",
  "agents.catalogMode": "Model catalog",
  "agents.catalogCount": "Provides {{count}} models / routes",
  "agents.mode.local": "Application-managed local catalog",
  "agents.mode.remote": "Remote gateway catalog",
  "agents.mode.desktop": "Desktop client model list",
  "agents.mode.cli": "CLI model picker",
  "agents.mode.default": "Client default model",
  "agents.catalogRefreshHint":
    "Restart the client after catalog updates. An existing available model is preserved; unknown capabilities remain unconfirmed.",
  "agents.preview": "Configuration preview",
  "agents.apply": "Apply configuration",
  "agents.applying": "Applying…",
  "agents.close": "Close",
  "agents.restart": "Restart the client after saving. Existing sessions are preserved.",
  "agents.desktopHint":
    "Applying activates third-party gateway mode and selects the OmniProxy local gateway profile. Quit Claude completely and restart, then open Code to use the local gateway without signing in to an Anthropic account. Existing settings and sessions are preserved. Restore configuration also restores the original startup mode.",
  "agents.stopped":
    "The gateway is stopped. You can save the connection configuration now; start the gateway before making model requests from the client.",
  "agents.noModels":
    "Sync account models or configure an available route on the Routing page first.",
  "agents.success": "Configuration saved with an encrypted backup.",
  "agents.successNotification": "Configuration saved notification",
  "agents.dismissSuccess": "Dismiss configuration saved notification",
  "agents.restored": "Original configuration restored.",
  "agents.error.config_changed":
    "Another tool modified the file. Reload configuration, preview, and apply again. Current edits are preserved.",
  "agents.error.invalid_config": "Fix the original configuration format first.",
  "agents.error.auth_conflict": "Resolve the provider credential helper conflict first.",
  "agents.error.profile_conflict":
    "Check the active configuration profile’s provider overrides first.",
  "agents.error.key_disabled": "Enable this client’s access key first.",
  "agents.error.desktop_profile_active":
    "Switch Claude Desktop back to its original profile before removing the active gateway profile.",
  "agents.error.failed": "Operation failed. Check the client configuration and gateway.",
  "agents.error.write_failed":
    "Unable to save the client configuration. Check the configuration directory’s write permissions and retry.",
  "agents.error.read_failed":
    "Unable to read the client configuration. Check file permissions and reload.",
  "agents.error.unsafe_config":
    "The configuration path or file is unsafe. Check its file type, size, and links.",
  "agents.error.config_locked":
    "The client is using the configuration file. Quit the client completely, then preview and apply again.",
  "agents.error.backup_failed":
    "Unable to save the encrypted backup. Check the app data directory and retry.",
  "agents.error.backup_unavailable":
    "The original configuration backup is unavailable. Check the app data directory.",
  "agents.error.rollback_failed":
    "Configuration writing and rollback failed. Check directory permissions and restore the encrypted backup.",
  "agents.error.no_models":
    "No available models or routes. Sync account models or configure a route first.",
  "agents.error.models_unavailable": "Unable to read the model catalog. Refresh and try again.",
  "agents.error.catalog_changed": "The model catalog changed. Preview again before applying.",
  "agents.error.catalog_conflict":
    "Another file exists at the managed catalog location. Resolve the file conflict first.",
  "agents.error.catalog_too_large":
    "The model catalog exceeds the client connection limit. Reduce available models or routes.",
  "agents.cliModelHint":
    "Adds the model list to the /model picker. You can also switch with /model <model-id>. Update older Claude Code clients if the list does not appear.",
  "models.title": "Official model catalog",
  "models.account": "Account to sync",
  "models.choose": "Choose an account",
  "models.sync": "Sync official models",
  "models.syncing": "Syncing…",
  "models.failed":
    "Sync failed. The last successful catalog is preserved. Check authentication and network.",
  "models.empty":
    "No models yet. Sync a signed-in ChatGPT or a DeepSeek, Gemini, or MiMo API key account; River, Atria and MiMo presets appear when an API key account is added.",
  "models.synced": "Synced {{count}} account models",
  "models.catalogCount": "{{count}} account models",
  "models.presetSource": "Official preset · Account access unverified",
  "models.metadataUnknown": "Context capability unconfirmed",
  "models.contextWindow": "Context: {{count}} tokens",
  "models.reasoningEfforts": "Reasoning effort: {{values}}",
} as const
