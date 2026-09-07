Unicode true

####
## Please note: Template replacements don't work in this file. They are provided with default defines like
## mentioned underneath.
## If the keyword is not defined, "wails_tools.nsh" will populate them with the values from ProjectInfo.
## If they are defined here, "wails_tools.nsh" will not touch them. This allows to use this project.nsi manually
## from outside of Wails for debugging and development of the installer.
##
## For development first make a wails nsis build to populate the "wails_tools.nsh":
## > wails build --target windows/amd64 --nsis
## Then you can call makensis on this file with specifying the path to your binary:
## For a AMD64 only installer:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app.exe
## For a ARM64 only installer:
## > makensis -DARG_WAILS_ARM64_BINARY=..\..\bin\app.exe
## For a installer with both architectures:
## > makensis -DARG_WAILS_AMD64_BINARY=..\..\bin\app-amd64.exe -DARG_WAILS_ARM64_BINARY=..\..\bin\app-arm64.exe
####
## The following information is taken from the ProjectInfo file, but they can be overwritten here.
####
## !define INFO_PROJECTNAME    "MyProject" # Default "{{.Name}}"
## !define INFO_COMPANYNAME    "MyCompany" # Default "{{.Info.CompanyName}}"
## !define INFO_PRODUCTNAME    "MyProduct" # Default "{{.Info.ProductName}}"
## !define INFO_PRODUCTVERSION "1.0.0"     # Default "{{.Info.ProductVersion}}"
## !define INFO_COPYRIGHT      "Copyright" # Default "{{.Info.Copyright}}"
###
## !define PRODUCT_EXECUTABLE  "Application.exe"      # Default "${INFO_PROJECTNAME}.exe"
## !define UNINST_KEY_NAME     "UninstKeyInRegistry"  # Default "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
####
## !define REQUEST_EXECUTION_LEVEL "admin"            # Default "admin"  see also https://nsis.sourceforge.io/Docs/Chapter4.html
####
## Include the wails tools
####
!include "wails_tools.nsh"

!define WAILS_WIN10_REQUIRED "OmniProxy 仅支持 Windows 10（Server 2016）及更新版本。"
!define WAILS_ARCHITECTURE_NOT_SUPPORTED "OmniProxy 无法安装在当前 Windows 架构上。支持架构：${ARCH}"
!define WAILS_INSTALL_WEBVIEW_DETAILPRINT "正在安装：WebView2 Runtime"
!define OMNIPROXY_HOMEPAGE "https://github.com/mibgb65-cloud/OmniProxy"

# The version information for this two must consist of 4 parts
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} 安装程序"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

# Enable HiDPI support. https://nsis.sourceforge.io/Reference/ManifestDPIAware
ManifestDPIAware true

!include "MUI.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
# !define MUI_WELCOMEFINISHPAGE_BITMAP "resources\leftimage.bmp" #Include this to add a bitmap on the left side of the Welcome Page. Must be a size of 164x314
!define MUI_FINISHPAGE_NOAUTOCLOSE # Wait on the INSTFILES page so the user can take a look into the details of the installation steps
!define MUI_FINISHPAGE_RUN "$INSTDIR\${PRODUCT_EXECUTABLE}"
!define MUI_FINISHPAGE_RUN_TEXT "立即启动 OmniProxy"
!define MUI_ABORTWARNING # This will warn the user if they exit from the installer.

!insertmacro MUI_PAGE_WELCOME # Welcome to the installer page.
!insertmacro MUI_PAGE_LICENSE "..\..\..\LICENSE" # Display the project's MIT license before installation.
!insertmacro MUI_PAGE_DIRECTORY # In which folder install page.
!insertmacro MUI_PAGE_INSTFILES # Installing page.
!insertmacro MUI_PAGE_FINISH # Finished installation page.

!insertmacro MUI_UNPAGE_INSTFILES # Uinstalling page

!insertmacro MUI_LANGUAGE "SimpChinese" # Set the Language of the installer

## The following two statements can be used to sign the installer and the uninstaller. The path to the binaries are provided in %1
#!uninstfinalize 'signtool --file "%1"'
#!finalize 'signtool --file "%1"'

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe" # Name of the installer's file.
InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}" # Default installing folder ($PROGRAMFILES is Program Files folder).
InstallDirRegKey HKLM "${UNINST_KEY}" "InstallLocation"
ShowInstDetails show # This will always show the installation details.

Var RemoveUserData
Var AutoUpdate
Var ExistingInstall

Function .onInit
   !insertmacro wails.checkArchitecture
   SetRegView 64
   StrCpy $AutoUpdate "0"
   StrCpy $ExistingInstall "0"
   ${GetParameters} $0
   ClearErrors
   ${GetOptions} $0 "/OMNIPROXY_AUTOUPDATE=" $1
   IfErrors noAutoUpdate
   StrCmp $1 "1" 0 noAutoUpdate
   StrCpy $AutoUpdate "1"
   SetSilent silent
   SetAutoClose true

noAutoUpdate:

   ReadRegStr $0 HKLM "${UNINST_KEY}" "InstallLocation"
   ${If} $0 != ""
       StrCpy $INSTDIR "$0"
       StrCpy $ExistingInstall "1"
       Return
   ${EndIf}

   # Older installers did not persist InstallLocation. Reuse DisplayIcon to
   # recover the existing install directory for upgrades from those versions.
   ReadRegStr $0 HKLM "${UNINST_KEY}" "DisplayIcon"
   ${If} $0 != ""
       ${GetParent} "$0" $INSTDIR
       StrCpy $ExistingInstall "1"
   ${EndIf}
FunctionEnd

Function ensureAppNotRunning
    StrCmp $AutoUpdate "1" autoUpdateCheck interactiveCheck

autoUpdateCheck:
    StrCpy $2 0

autoUpdateWait:
    nsExec::ExecToStack 'cmd /C tasklist /FI $\"IMAGENAME eq ${PRODUCT_EXECUTABLE}$\" /NH | find /I $\"${PRODUCT_EXECUTABLE}$\" >NUL'
    Pop $0
    Pop $1
    IntCmp $0 0 autoUpdateRunning autoUpdateDone autoUpdateDone

autoUpdateRunning:
    IntCmp $2 30 autoUpdateForce autoUpdateSleep autoUpdateForce

autoUpdateSleep:
    DetailPrint "正在等待 OmniProxy 退出..."
    Sleep 1000
    IntOp $2 $2 + 1
    Goto autoUpdateWait

autoUpdateForce:
    DetailPrint "OmniProxy 未及时退出，正在强制关闭主进程..."
    nsExec::ExecToLog 'taskkill /F /IM "${PRODUCT_EXECUTABLE}"'
    Sleep 1000
    Goto autoUpdateWait

autoUpdateDone:
    Return

interactiveCheck:
check:
    nsExec::ExecToStack 'cmd /C tasklist /FI $\"IMAGENAME eq ${PRODUCT_EXECUTABLE}$\" /NH | find /I $\"${PRODUCT_EXECUTABLE}$\" >NUL'
    Pop $0
    Pop $1
    ${If} $0 != 0
        Return
    ${EndIf}

    MessageBox MB_YESNOCANCEL|MB_ICONEXCLAMATION "检测到 OmniProxy 正在运行。$\r$\n$\r$\n选择“是”将尝试自动关闭后继续安装；选择“否”可手动关闭后重试；选择“取消”退出安装。" IDYES close IDNO manual

cancel:
    Abort

close:
    DetailPrint "正在关闭 OmniProxy..."
    nsExec::ExecToLog 'taskkill /IM "${PRODUCT_EXECUTABLE}"'
    Sleep 2000
    nsExec::ExecToStack 'cmd /C tasklist /FI $\"IMAGENAME eq ${PRODUCT_EXECUTABLE}$\" /NH | find /I $\"${PRODUCT_EXECUTABLE}$\" >NUL'
    Pop $0
    Pop $1
    ${If} $0 != 0
        Return
    ${EndIf}
    MessageBox MB_YESNO|MB_ICONEXCLAMATION "OmniProxy 主进程仍在运行。$\r$\n$\r$\n是否强制关闭 OmniProxy.exe 后继续安装？" IDYES force IDNO manual

force:
    DetailPrint "正在强制关闭 OmniProxy 主进程..."
    nsExec::ExecToLog 'taskkill /F /IM "${PRODUCT_EXECUTABLE}"'
    Sleep 1000
    Goto check

manual:
    MessageBox MB_OK "请关闭 OmniProxy 后点击确定继续。"
    Goto check
FunctionEnd

Function un.onInit
    StrCpy $RemoveUserData "0"
    ${GetParameters} $0
    ClearErrors
    ${GetOptions} $0 "/OMNIPROXY_REMOVE_DATA=" $1
    IfErrors noRemoveOption
    StrCmp $1 "1" 0 noRemoveOption
    StrCpy $RemoveUserData "1"

noRemoveOption:
    IfSilent done ask

ask:
    MessageBox MB_YESNO|MB_ICONQUESTION "是否保留 OmniProxy 本地数据？$\r$\n$\r$\n选择“是”会保留账号、配置和请求历史；选择“否”会删除默认数据目录和数据目录配置。" IDYES keep IDNO remove
remove:
    StrCpy $RemoveUserData "1"
keep:
done:
FunctionEnd

Function createOrRefreshShortcuts
    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    ${If} $ExistingInstall == "1"
        IfFileExists "$DESKTOP\${INFO_PRODUCTNAME}.lnk" refreshDesktop done
refreshDesktop:
        CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
        Goto done
    ${EndIf}

    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

done:
FunctionEnd

Function un.ensureAppNotRunning
    IfSilent silentCheck interactiveCheck

silentCheck:
    nsExec::ExecToStack 'cmd /C tasklist /FI $\"IMAGENAME eq ${PRODUCT_EXECUTABLE}$\" /NH | find /I $\"${PRODUCT_EXECUTABLE}$\" >NUL'
    Pop $0
    Pop $1
    ${If} $0 == 0
        DetailPrint "正在关闭 OmniProxy..."
        nsExec::ExecToLog 'taskkill /F /IM "${PRODUCT_EXECUTABLE}"'
        Sleep 1000
    ${EndIf}
    Return

interactiveCheck:
check:
    nsExec::ExecToStack 'cmd /C tasklist /FI $\"IMAGENAME eq ${PRODUCT_EXECUTABLE}$\" /NH | find /I $\"${PRODUCT_EXECUTABLE}$\" >NUL'
    Pop $0
    Pop $1
    ${If} $0 != 0
        Return
    ${EndIf}

    MessageBox MB_YESNOCANCEL|MB_ICONEXCLAMATION "检测到 OmniProxy 正在运行。$\r$\n$\r$\n选择“是”将关闭后继续卸载；选择“否”可手动关闭后重试；选择“取消”退出卸载。" IDYES close IDNO manual

cancel:
    Abort

close:
    DetailPrint "正在关闭 OmniProxy..."
    nsExec::ExecToLog 'taskkill /IM "${PRODUCT_EXECUTABLE}"'
    Sleep 2000
    nsExec::ExecToStack 'cmd /C tasklist /FI $\"IMAGENAME eq ${PRODUCT_EXECUTABLE}$\" /NH | find /I $\"${PRODUCT_EXECUTABLE}$\" >NUL'
    Pop $0
    Pop $1
    ${If} $0 != 0
        Return
    ${EndIf}
    MessageBox MB_YESNO|MB_ICONEXCLAMATION "OmniProxy 主进程仍在运行。$\r$\n$\r$\n是否强制关闭 OmniProxy.exe 后继续卸载？" IDYES force IDNO manual

force:
    DetailPrint "正在强制关闭 OmniProxy 主进程..."
    nsExec::ExecToLog 'taskkill /F /IM "${PRODUCT_EXECUTABLE}"'
    Sleep 1000
    Goto check

manual:
    MessageBox MB_OK "请关闭 OmniProxy 后点击确定继续。"
    Goto check
FunctionEnd

Function un.deleteAppShortcuts
    SetShellVarContext all
    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"
    SetShellVarContext current
    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"
    !insertmacro wails.setShellContext
FunctionEnd

Function un.clearUserAutoStart
    DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "OmniProxy"
FunctionEnd

Section
    !insertmacro wails.setShellContext
    Call ensureAppNotRunning

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    !insertmacro wails.files

    Call createOrRefreshShortcuts

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    !insertmacro wails.writeUninstaller

    SetRegView 64
    WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
    WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1
    WriteRegStr HKLM "${UNINST_KEY}" "URLInfoAbout" "${OMNIPROXY_HOMEPAGE}"
    WriteRegStr HKLM "${UNINST_KEY}" "HelpLink" "${OMNIPROXY_HOMEPAGE}/issues"

    ${If} $AutoUpdate == "1"
        Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}"'
    ${EndIf}
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext
    Call un.ensureAppNotRunning
    Call un.clearUserAutoStart
    Call un.deleteAppShortcuts

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}" # Remove the WebView2 DataPath

    ${If} $RemoveUserData == "1"
        RMDir /r "$PROFILE\.omniproxy"
        RMDir /r "$APPDATA\OmniProxy"
        Delete "$PROFILE\.omniproxy-bootstrap.json"
    ${EndIf}

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    !insertmacro wails.deleteUninstaller
    RMDir /r "$INSTDIR"
SectionEnd
