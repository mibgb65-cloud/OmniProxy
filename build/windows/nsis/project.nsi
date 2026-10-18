Unicode true
!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "CompanyName" "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} 安装程序"
VIAddVersionKey "ProductVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion" "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright" "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName" "${INFO_PRODUCTNAME}"
ManifestDPIAware true

!include "MUI.nsh"
!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_FUNCTION StartInstalledApp
!define MUI_FINISHPAGE_RUN_TEXT "立即启动 OmniProxy"
!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
!else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif
ShowInstDetails show
!include "installer_runtime.nsh"

Section
    !insertmacro wails.setShellContext
    !insertmacro ExtractHelper
    !insertmacro StopTarget
    !insertmacro wails.webview2runtime
    SetOutPath $INSTDIR
    !insertmacro wails.files
    Call CreateAppShortcuts
    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols
    !insertmacro wails.writeUninstaller
    !insertmacro WriteInstallLocation
    ${If} $AutoUpdate == "1"
    ${AndIf} $NoRestart != "1"
        Call StartInstalledApp
    ${EndIf}
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext
    !insertmacro ExtractHelper
    !insertmacro StopTarget
    nsExec::ExecToStack '"$PLUGINSDIR\omniproxy-installer-helper.exe" --omniproxy-installer-cleanup "$INSTDIR\${PRODUCT_EXECUTABLE}"'
    Pop $0
    Pop $1
    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"
    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols
    Delete "$INSTDIR\${PRODUCT_EXECUTABLE}"
    !insertmacro wails.deleteUninstaller
    ; 只删除程序自有文件；未知文件及所有账号／数据库始终保留。
    RMDir "$INSTDIR"
    DetailPrint "OmniProxy 本地数据已保留。"
SectionEnd
