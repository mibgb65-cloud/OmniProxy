!include "LogicLib.nsh"

!macro ExtractHelper
    InitPluginsDir
    ; 数据位置页需要在向导阶段调用辅助程序读取当前目录，因此 .onInit 里先解压一次。
    IfFileExists "$PLUGINSDIR\omniproxy-installer-helper.exe" +2
    !ifdef ARG_WAILS_AMD64_BINARY
        File "/oname=$PLUGINSDIR\omniproxy-installer-helper.exe" "${ARG_WAILS_AMD64_BINARY}"
    !else
        !error "OmniProxy 当前发布仅支持 Windows amd64"
    !endif
!macroend

Var AutoUpdate
Var NoRestart
Var ExistingInstall
Var DataDirSourceDir

Function .onInit
    !insertmacro wails.checkArchitecture
    SetRegView 64
    StrCpy $AutoUpdate "0"
    StrCpy $NoRestart "0"
    StrCpy $ExistingInstall "0"
    ${GetParameters} $0
    ClearErrors
    ${GetOptions} $0 "/OMNIPROXY_AUTOUPDATE=" $1
    ${IfNot} ${Errors}
        ${If} $1 == "1"
            StrCpy $AutoUpdate "1"
            SetSilent silent
            SetAutoClose true
        ${EndIf}
    ${EndIf}
    ClearErrors
    ${GetOptions} $0 "/OMNIPROXY_NORESTART=" $1
    ${IfNot} ${Errors}
        ${If} $1 == "1"
            StrCpy $NoRestart "1"
        ${EndIf}
    ${EndIf}
    ; 语言选择：静默升级已在上面 SetSilent，此处自动跳过；交互安装先选语言。
    !insertmacro MUI_LANGDLL_DISPLAY
    !insertmacro ExtractHelper
    !if "${WAILS_INSTALL_SCOPE}" == "user"
        ReadRegStr $0 HKCU "${UNINST_KEY}" "InstallLocation"
    !else
        ReadRegStr $0 HKLM "${UNINST_KEY}" "InstallLocation"
    !endif
    ${If} $0 == ""
        !if "${WAILS_INSTALL_SCOPE}" == "user"
            ReadRegStr $0 HKCU "${UNINST_KEY}" "DisplayIcon"
        !else
            ReadRegStr $0 HKLM "${UNINST_KEY}" "DisplayIcon"
        !endif
        ${If} $0 != ""
            StrCpy $1 $0 1
            ${If} $1 == '$\"'
                StrCpy $0 $0 "" 1
            ${EndIf}
            ${GetParent} "$0" $0
        ${EndIf}
    ${EndIf}
    ${If} $0 != ""
        StrCpy $INSTDIR "$0"
        StrCpy $ExistingInstall "1"
    ${EndIf}
    ; 目录选择页可以修改 INSTDIR，但历史数据标记始终来自原安装位置。
    StrCpy $DataDirSourceDir "$INSTDIR"
    Call .onVerifyInstDir
FunctionEnd

Function .onVerifyInstDir
    ${GetRoot} "$INSTDIR" $0
    ${If} $INSTDIR == ""
    ${OrIf} $INSTDIR == $0
    ${OrIf} $INSTDIR == "$WINDIR"
    ${OrIf} $INSTDIR == "$SYSDIR"
    ${OrIf} $INSTDIR == "$PROFILE"
    ${OrIf} $INSTDIR == "$PROGRAMFILES64"
    ${OrIf} $INSTDIR == "$LOCALAPPDATA"
        Abort
    ${EndIf}
FunctionEnd

!macro StopTarget
    nsExec::ExecToStack '"$PLUGINSDIR\omniproxy-installer-helper.exe" --omniproxy-installer-stop "$INSTDIR\${PRODUCT_EXECUTABLE}"'
    Pop $0
    Pop $1
    ${If} $0 != 0
        IfSilent +2
            MessageBox MB_OK|MB_ICONSTOP "$(OMNI_STOP_FAILED)"
        SetErrorLevel 66
        Abort
    ${EndIf}
!macroend

Function CreateAppShortcuts
    ; 独立图标路径避免旧版 EXE 路径的 Shell 缓存继续显示旧图标。
    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\app-icon-v2.ico" 0
    ${If} $ExistingInstall == "1"
        IfFileExists "$DESKTOP\${INFO_PRODUCTNAME}.lnk" 0 done
    ${EndIf}
    CreateShortcut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}" "" "$INSTDIR\app-icon-v2.ico" 0
done:
FunctionEnd

!macro WriteInstallLocation
    !if "${WAILS_INSTALL_SCOPE}" == "user"
        WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
        WriteRegStr HKCU "${UNINST_KEY}" "InstallScope" "user"
        WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
        WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1
    !else
        WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
        WriteRegStr HKLM "${UNINST_KEY}" "InstallScope" "machine"
        WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
        WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1
    !endif
!macroend

Function StartInstalledApp
    nsExec::ExecToStack '"$PLUGINSDIR\omniproxy-installer-helper.exe" --omniproxy-installer-launch "$INSTDIR\${PRODUCT_EXECUTABLE}"'
    Pop $0
    Pop $1
    ${If} $0 != 0
        IfSilent +2
            MessageBox MB_OK|MB_ICONEXCLAMATION "$(OMNI_RUN_FAILED)"
    ${EndIf}
FunctionEnd
