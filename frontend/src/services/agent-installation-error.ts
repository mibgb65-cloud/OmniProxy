export function agentInstallationError(error: unknown) {
  const code = String(error instanceof Error ? error.message : error).replace("agentinstall: ", "")
  switch (code) {
    case "installer_unavailable":
    case "detection_failed":
    case "version_unavailable":
    case "install_failed":
    case "install_interrupted":
    case "verification_failed":
    case "busy":
    case "unsupported_platform":
    case "official_unavailable":
    case "launch_failed":
    case "activation_required":
    case "update_check_failed":
    case "update_unsupported":
    case "update_failed":
    case "update_unavailable":
    case "update_interrupted":
      return code
    default:
      return "failed"
  }
}
