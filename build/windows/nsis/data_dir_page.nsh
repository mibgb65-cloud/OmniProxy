; 数据存储位置页。
;
; 升级安装时本页显示当前用户**正在使用**的数据目录（由应用自身的解析逻辑
; 通过安装辅助程序读出，来源是 C 盘上的指针文件 %APPDATA%\OmniProxy\app.json），
; 而不是安装目录里的猜测值。用户改动时先给出「不会迁移已有数据」的警告。
;
; 写入也走辅助程序：真正的迁移只能由应用在退出时完成（可回滚），安装器只在
; 目标目录还没有数据时允许改指针，已有数据时直接拒绝。

!include "nsDialogs.nsh"

Var DataDirPath
Var DataDirCurrent
Var DataDirChanged
Var DataDirEdit
Var DataDirBrowse

Function DataDirPageCreate
    StrCpy $DataDirChanged "0"
    StrCpy $DataDirCurrent ""
    StrCpy $DataDirPath ""

    nsExec::ExecToStack '"$PLUGINSDIR\omniproxy-installer-helper.exe" --omniproxy-installer-data-dir-read'
    Pop $0
    Pop $1
    ${If} $0 == "0"
    ${AndIf} $1 != ""
        StrCpy $DataDirPath $1
        Call DataDirTrimNewLines
        StrCpy $DataDirCurrent $DataDirPath
    ${Else}
        StrCpy $DataDirPath ""
    ${EndIf}

    nsDialogs::Create 1018
    Pop $0
    ${If} $0 == error
        Abort
    ${EndIf}

    ${NSD_CreateLabel} 0 0 100% 34u "$(OMNI_DATADIR_INTRO)"
    Pop $0

    ${NSD_CreateDirRequest} 0 40u -64u 13u "$DataDirPath"
    Pop $DataDirEdit
    ${NSD_CreateBrowseButton} -60u 40u 60u 13u "$(OMNI_DATADIR_BROWSE)"
    Pop $DataDirBrowse
    ${NSD_OnClick} $DataDirBrowse DataDirBrowseClick

    ${NSD_CreateLabel} 0 60u 100% 34u "$(OMNI_DATADIR_HINT)"
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
    ; 去掉结尾的反斜杠，避免 "D:\Data" 与 "D:\Data\" 被当成两个位置。
    StrLen $1 $DataDirPath
    ${If} $1 > 3
        StrCpy $1 $DataDirPath 1 -1
        ${If} $1 == "\"
            StrCpy $DataDirPath $DataDirPath -1
        ${EndIf}
    ${EndIf}
FunctionEnd

Function DataDirBrowseClick
    ${NSD_GetText} $DataDirEdit $0
    nsDialogs::SelectFolderDialog "$(OMNI_DATADIR_TITLE)" "$0"
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
    Call DataDirTrimNewLines

    StrCmp $DataDirPath "" data_dir_leave_empty
    ${GetRoot} "$DataDirPath" $0
    ${If} $0 == ""
        MessageBox MB_OK|MB_ICONEXCLAMATION "$(OMNI_DATADIR_NOT_ABS)"
        Abort
    ${EndIf}
    ${If} $DataDirPath == $0
        MessageBox MB_OK|MB_ICONEXCLAMATION "$(OMNI_DATADIR_IS_ROOT)"
        Abort
    ${EndIf}

    ${If} $DataDirPath == $DataDirCurrent
        StrCpy $DataDirChanged "0"
        Goto data_dir_leave_done
    ${EndIf}
    ; 与原位置不同：先警告不会迁移已有数据，用户取消就留在本页。
    MessageBox MB_YESNO|MB_ICONEXCLAMATION "$(OMNI_DATADIR_CHANGE_WARN)" IDNO data_dir_leave_abort
    StrCpy $DataDirChanged "1"
    Goto data_dir_leave_done

    data_dir_leave_abort:
    Abort

    data_dir_leave_empty:
    ; 留空表示保持当前位置不变。
    StrCpy $DataDirChanged "0"

    data_dir_leave_done:
FunctionEnd

!macro WriteDataDirChoice
    ${IfNot} ${Silent}
        ${If} $DataDirChanged == "1"
        ${AndIf} $DataDirPath != ""
            nsExec::ExecToStack '"$PLUGINSDIR\omniproxy-installer-helper.exe" --omniproxy-installer-data-dir-write "$DataDirPath"'
            Pop $0
            Pop $1
            ${If} $0 == "0"
                DetailPrint "$(OMNI_DATADIR_WRITTEN)$DataDirPath"
            ${ElseIf} $0 == "3"
                MessageBox MB_OK|MB_ICONEXCLAMATION "$(OMNI_DATADIR_IN_USE)"
            ${Else}
                MessageBox MB_OK|MB_ICONEXCLAMATION "$(OMNI_DATADIR_CHANGE_FAILED)$1"
            ${EndIf}
        ${EndIf}
    ${EndIf}
!macroend
