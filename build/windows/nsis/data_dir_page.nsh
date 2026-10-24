; 数据存储位置页。
;
; 安装器在这里只记录「建议位置」，不直接写任何用户的 app.json：数据目录是
; 每用户概念（%APPDATA% + DPAPI 凭据），而安装器是机器级的，提权进程的
; $APPDATA 未必属于实际使用该程序的账号。
;
; 应用启动时按以下顺序解析数据目录：
;   OMNIPROXY_DATA_DIR > app.json 的 dataDir > 本页写入的 data-dir.txt > 默认值
; 因此安装时的选择只在用户还没有自己的选择时生效，静默升级也不会改动它。

!include "nsDialogs.nsh"

Var DataDirPath
Var DataDirKeepExisting
Var DataDirEdit
Var DataDirBrowse

Function DataDirPageCreate
    StrCpy $DataDirKeepExisting "0"
    StrCpy $DataDirPath ""
    ; 覆盖安装时沿用已有选择，避免升级把用户挑选的位置重置回默认。
    IfFileExists "$INSTDIR\data-dir.txt" 0 data_dir_page_no_file
        StrCpy $DataDirKeepExisting "1"
        ClearErrors
        FileOpen $0 "$INSTDIR\data-dir.txt" r
        ${IfNot} ${Errors}
            FileRead $0 $DataDirPath
            FileClose $0
            Call DataDirTrimNewLines
        ${EndIf}
    data_dir_page_no_file:

    nsDialogs::Create 1018
    Pop $0
    ${If} $0 == error
        Abort
    ${EndIf}

    ${NSD_CreateLabel} 0 0 100% 26u "默认保存到当前 Windows 用户的应用数据目录（%APPDATA%\OmniProxy）。如需放到其他磁盘，请在这里选择目录。"
    Pop $0

    ${NSD_CreateDirRequest} 0 32u -64u 13u "$DataDirPath"
    Pop $DataDirEdit
    ${NSD_CreateBrowseButton} -60u 32u 60u 13u "浏览..."
    Pop $DataDirBrowse
    ${NSD_OnClick} $DataDirBrowse DataDirBrowseClick

    ${NSD_CreateLabel} 0 52u 100% 34u "留空表示使用默认位置。目标目录由 OmniProxy 首次启动时按当前用户权限创建；已在应用内更改过数据位置的用户不受此处影响。"
    Pop $0

    nsDialogs::Show
FunctionEnd

Function DataDirTrimNewLines
    ${Do}
        StrCpy $1 $DataDirPath 1 -1
        ${If} $1 == "$\r"
        ${OrIf} $1 == "$\n"
            StrCpy $DataDirPath $DataDirPath -1
        ${Else}
            ${ExitDo}
        ${EndIf}
    ${Loop}
FunctionEnd

Function DataDirBrowseClick
    ${NSD_GetText} $DataDirEdit $0
    nsDialogs::SelectFolderDialog "选择 OmniProxy 数据存储位置" "$0"
    Pop $0
    ${If} $0 != error
        ${NSD_SetText} $DataDirEdit "$0"
    ${EndIf}
FunctionEnd

Function DataDirPageLeave
    ${NSD_GetText} $DataDirEdit $DataDirPath
    ; 手工输入的路径可能带引号，去掉首尾引号后再校验。
    StrCpy $1 $DataDirPath 1
    ${If} $1 == '$\"'
        StrCpy $DataDirPath $DataDirPath "" 1
    ${EndIf}
    StrCpy $1 $DataDirPath 1 -1
    ${If} $1 == '$\"'
        StrCpy $DataDirPath $DataDirPath -1
    ${EndIf}

    StrCmp $DataDirPath "" data_dir_leave_empty
    ${GetRoot} "$DataDirPath" $0
    ${If} $0 == ""
        MessageBox MB_OK|MB_ICONEXCLAMATION "请选择绝对路径，例如 D:\OmniProxy。"
        Abort
    ${EndIf}
    ${If} $DataDirPath == $0
        MessageBox MB_OK|MB_ICONEXCLAMATION "不能使用磁盘根目录，请选择磁盘下的一个文件夹。"
        Abort
    ${EndIf}
    StrCpy $DataDirKeepExisting "0"
    Goto data_dir_leave_done

    data_dir_leave_empty:
    ; 留空：保留覆盖安装前的设置，首次安装则什么都不写。
    data_dir_leave_done:
FunctionEnd

!macro WriteDataDirChoice
    ${IfNot} ${Silent}
        ${If} $DataDirPath == ""
            ${If} $DataDirKeepExisting == "0"
                Delete "$INSTDIR\data-dir.txt"
            ${EndIf}
        ${Else}
            ClearErrors
            FileOpen $9 "$INSTDIR\data-dir.txt" w
            ${If} ${Errors}
                DetailPrint "无法写入数据存储位置设置，将使用默认位置。"
            ${Else}
                FileWrite $9 "$DataDirPath"
                FileClose $9
                DetailPrint "数据存储位置：$DataDirPath"
            ${EndIf}
        ${EndIf}
    ${EndIf}
!macroend
