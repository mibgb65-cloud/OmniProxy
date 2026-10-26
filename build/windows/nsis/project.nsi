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
!define MUI_FINISHPAGE_RUN_TEXT "$(OMNI_RUN_TEXT)"
!define MUI_ABORTWARNING
; 语言选择在 .onInit 里弹出（静默升级自动跳过）；选择记录在 HKCU，卸载器复用。
!define MUI_LANGDLL_REGISTRY_ROOT "HKCU"
!define MUI_LANGDLL_REGISTRY_KEY "Software\OmniProxy"
!define MUI_LANGDLL_REGISTRY_VALUENAME "InstallerLanguage"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
Page custom DataDirPageCreate DataDirPageLeave
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"
!insertmacro MUI_LANGUAGE "English"
!include "strings.nsh"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
!if "${WAILS_INSTALL_SCOPE}" == "user"
    InstallDir "$LOCALAPPDATA\Programs\${INFO_PRODUCTNAME}"
!else
    InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
!endif
ShowInstDetails show
!include "installer_runtime.nsh"
!include "data_dir_page.nsh"

Section
    !insertmacro wails.setShellContext
    !insertmacro ExtractHelper
    !insertmacro StopTarget
    !insertmacro wails.webview2runtime
    SetOutPath $INSTDIR
    !insertmacro wails.files
    File "/oname=app-icon-v2.ico" "..\icon.ico"
    File "/oname=OmniProxy-LICENSE.txt" "..\..\..\LICENSE"
    ; 只分发公开证书，不修改用户证书库或系统信任设置。
    File "/oname=OmniProxy-CodeSigning.cer" "..\codesign.cer"
    !insertmacro WriteDataDirChoice
    Call CreateAppShortcuts
    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols
    !insertmacro wails.writeUninstaller
    !insertmacro WriteInstallLocation
    ${IfNot} ${Silent}
        !insertmacro MUI_LANGDLL_SAVELANGUAGE
    ${EndIf}
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
    Delete "$INSTDIR\app-icon-v2.ico"
    Delete "$INSTDIR\OmniProxy-LICENSE.txt"
    Delete "$INSTDIR\OmniProxy-CodeSigning.cer"
    Delete "$INSTDIR\data-dir.txt"
    !insertmacro wails.deleteUninstaller
    ; 只删除程序自有文件；未知文件及所有账号／数据库始终保留。
    RMDir "$INSTDIR"
    DetailPrint "$(OMNI_DATA_KEPT)"
SectionEnd
