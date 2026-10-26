; 安装器界面文案（中英双语）。
;
; 所有自定义页面、消息框和详情输出都通过 LangString 取值，避免选了 English
; 却弹出一半中文。语言由 MUI_PAGE_LANGDIALOG 选择，静默安装回落到默认语言。

LangString OMNI_DATADIR_TITLE      ${LANG_SIMPCHINESE} "选择数据存储位置"
LangString OMNI_DATADIR_TITLE      ${LANG_ENGLISH}     "Choose the data folder"

LangString OMNI_DATADIR_INTRO      ${LANG_SIMPCHINESE} "程序的所有数据都放在这个目录：数据库、账号凭据、日志和更新缓存。C 盘只保留一个很小的配置文件（%APPDATA%\OmniProxy\app.json）用来指向这里。"
LangString OMNI_DATADIR_INTRO      ${LANG_ENGLISH}     "All program data lives in this folder: database, account credentials, logs and update cache. Only a small configuration file (%APPDATA%\OmniProxy\app.json) stays on drive C to point here."

LangString OMNI_DATADIR_HINT       ${LANG_SIMPCHINESE} "留空使用默认位置（%APPDATA%\OmniProxy）。此处只决定首次启动的位置，不会迁移已有数据；已在使用中的数据请在应用内「数据存储 → 数据主目录」搬迁。"
LangString OMNI_DATADIR_HINT       ${LANG_ENGLISH}     "Leave empty to use the default location (%APPDATA%\OmniProxy). This only sets where a fresh installation starts; existing data is never moved here. Use Settings → Data storage → Data folder to relocate data that is already in use."

LangString OMNI_DATADIR_BROWSE     ${LANG_SIMPCHINESE} "浏览..."
LangString OMNI_DATADIR_BROWSE     ${LANG_ENGLISH}     "Browse..."

LangString OMNI_DATADIR_NOT_ABS    ${LANG_SIMPCHINESE} "请选择绝对路径，例如 D:\OmniProxy。"
LangString OMNI_DATADIR_NOT_ABS    ${LANG_ENGLISH}     "Please choose an absolute path, for example D:\OmniProxy."

LangString OMNI_DATADIR_IS_ROOT    ${LANG_SIMPCHINESE} "不能使用磁盘根目录，请选择磁盘下的一个文件夹。"
LangString OMNI_DATADIR_IS_ROOT    ${LANG_ENGLISH}     "A drive root cannot be used. Please choose a folder on the drive."

LangString OMNI_DATADIR_WRITE_FAIL ${LANG_SIMPCHINESE} "无法写入数据存储位置设置，将使用默认位置。"
LangString OMNI_DATADIR_WRITE_FAIL ${LANG_ENGLISH}     "The data location could not be saved; the default location will be used."

LangString OMNI_DATADIR_CHANGE_WARN ${LANG_SIMPCHINESE} "更改数据位置不会迁移已有数据：原目录里的账号、凭据和请求记录会留在原处，程序将从新目录重新开始。要保留并搬迁现有数据，请在安装完成后使用应用内「设置 → 数据存储 → 数据主目录」。$\r$\n$\r$\n仍要更改吗？"
LangString OMNI_DATADIR_CHANGE_WARN ${LANG_ENGLISH}     "Changing the data folder does not move existing data: accounts, credentials and request history stay where they are and the application starts over in the new folder. To keep and relocate existing data, use Settings → Data storage → Data folder after installation.$\r$\n$\r$\nChange it anyway?"

LangString OMNI_DATADIR_CHANGE_FAILED ${LANG_SIMPCHINESE} "数据位置没有更改："
LangString OMNI_DATADIR_CHANGE_FAILED ${LANG_ENGLISH}     "The data folder was not changed: "

LangString OMNI_DATADIR_READ_FAILED ${LANG_SIMPCHINESE} "无法确认当前使用者的数据位置，安装已停止，原有配置与数据保持不变。请登录实际使用者的桌面后重试，并检查原数据目录是否可访问。"
LangString OMNI_DATADIR_READ_FAILED ${LANG_ENGLISH}     "The current user's data folder could not be verified. Setup has stopped without changing existing settings or data. Sign in to the user's desktop and check access to the existing data folder before retrying."

LangString OMNI_DATADIR_IN_USE     ${LANG_SIMPCHINESE} "数据位置没有更改：原位置仍包含账号与请求记录，安装器不会迁移数据。请安装完成后在应用内「设置 → 数据存储 → 数据主目录」搬迁。"
LangString OMNI_DATADIR_IN_USE     ${LANG_ENGLISH}     "The data folder was not changed: the current location still holds accounts and request history, and the installer never moves data. Use Settings → Data storage → Data folder after installation."

LangString OMNI_DATADIR_WRITTEN    ${LANG_SIMPCHINESE} "数据存储位置："
LangString OMNI_DATADIR_WRITTEN    ${LANG_ENGLISH}     "Data folder: "

LangString OMNI_RUN_TEXT           ${LANG_SIMPCHINESE} "立即启动 OmniProxy"
LangString OMNI_RUN_TEXT           ${LANG_ENGLISH}     "Launch OmniProxy now"

LangString OMNI_RUN_FAILED         ${LANG_SIMPCHINESE} "安装已完成，但无法以普通用户启动 OmniProxy。请从开始菜单打开。"
LangString OMNI_RUN_FAILED         ${LANG_ENGLISH}     "Installation finished, but OmniProxy could not be started as the current user. Open it from the Start menu."

LangString OMNI_STOP_FAILED        ${LANG_SIMPCHINESE} "无法关闭目标安装目录中的 OmniProxy，请手动退出后重试。"
LangString OMNI_STOP_FAILED        ${LANG_ENGLISH}     "OmniProxy in the installation folder could not be closed. Quit it manually and try again."

LangString OMNI_DATA_KEPT          ${LANG_SIMPCHINESE} "OmniProxy 本地数据已保留。"
LangString OMNI_DATA_KEPT          ${LANG_ENGLISH}     "Local OmniProxy data has been kept."
