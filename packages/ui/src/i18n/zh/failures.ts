import type { failures as english } from "../en/failures.ts";

/** One message per failure code the desktop service reports. */
export const failures: typeof english = {
  "failure.general":
    "Ravenpass 无法完成此操作。请重试，如果问题仍然存在，请重新打开保险库。",
  "failure.window-unavailable":
    "Ravenpass 无法连接到窗口。请重新打开应用并重试。",
  "failure.authentication-canceled": "身份验证已取消。准备好后请重试。",
  "failure.authentication-failed": "无法验证您的身份。请重试。",
  "failure.unlock-unavailable":
    "安全设备解锁不可用。请检查设备的安全设置后重试。",
  "failure.unlock-key-missing":
    "生物识别已无法在此设备上解锁此保险库。请使用 PIN 码或恢复密钥，然后在设置中重新开启生物识别。",
  "failure.way-in-not-set":
    "Ravenpass 无法在此设备上设置此解锁方式。请重试或选择其他方式。",
  "failure.vault-locked": "保险库已锁定。请解锁后重试。",
  "failure.vault-exists": "该位置已存在保险库。请为新保险库选择其他文件。",
  "failure.setup-in-progress": "设置已在进行中。请先完成或取消当前设置。",
  "failure.setup-not-active":
    "保险库设置已不再处于活动状态。请重新开始创建保险库。",
  "failure.no-local-vault": "该位置未找到保险库。请选择加密副本以恢复。",
  "failure.recovery-replaced-key":
    "此副本使用了已被替换的恢复密钥。打开前请确认是由您本人恢复的。",
  "failure.key-change-unfinished":
    "恢复密钥已更改，但 Ravenpass 未能完成在此设备上的保存。今后请使用新的恢复密钥，并重新解锁保险库。",
  "failure.key-change-uncertain":
    "Ravenpass 无法确认恢复密钥是否已更改。请同时保留新旧恢复密钥，直到保险库再次成功打开。",
  "failure.recovery-needs-confirmation":
    "此副本可能比您的最新保险库更旧。在恢复前请确认可能丢失新近更改。",
  "failure.recovery-changed": "恢复期间保险库发生了变动。请重新开始恢复流程。",
  "failure.recovery-phrase-invalid": "恢复密钥不匹配。请核对单词后重试。",
  "failure.recovery-key-mismatch":
    "恢复密钥与此保险库不匹配。请核对单词后重试。",
  "failure.recovery-key-not-saved":
    "未保存文件。请选择一个位置以保存恢复密钥。",
  "failure.recovery-key-save-failed": "无法保存恢复密钥。请选择其他位置重试。",
  "failure.recovery-key-copy-failed":
    "无法复制恢复密钥。请将其保存到私密文件或手动抄写下来。",
  "failure.recovery-key-print-failed":
    "无法打印恢复密钥。请检查打印机后重试，或手动抄写下来。",
  "failure.vault-changed":
    "保险库自打开后发生了变动。编辑前请锁定 Ravenpass 并重新打开。",
  "failure.vault-missing":
    "保险库文件已不在打开时的位置。它可能已在其他设备上被移动。请从新位置打开保险库文件。",
  "failure.save-unconfirmed":
    "保存未能确认。进行更多更改前请重新打开 Ravenpass。",
  "failure.authentication-invalid":
    "无法打开保险库。请检查恢复密钥或使用已确认有效的副本。",
  "failure.unsupported-format":
    "此保险库文件的格式当前版本的 Ravenpass 无法打开。请使用恢复密钥从加密副本恢复，或创建新保险库。",
  "failure.malformed-vault":
    "此保险库文件无法读取。请使用恢复密钥从加密副本恢复。",
  "failure.invalid-item": "部分字段无效。请检查后重试。",
  "failure.invalid-code":
    "该一次性代码设置无法生成代码。请粘贴网站提供的设置密钥或一次性代码链接。",
  "failure.no-code": "此密码未设置一次性代码。请先在项目中添加代码后重试。",
  "failure.code-setup-expired":
    "此一次性代码设置已过期。请重新打开其链接或扫描二维码。",
  "failure.qr-code-missing": "未找到二维码。请选择或复制显示完整二维码的图片。",
  "failure.qr-code-not-setup":
    "该二维码非一次性代码设置码。请使用网站为验证器应用显示的二维码。",
  "failure.qr-code-ambiguous":
    "图片中包含多个设置二维码。请使用仅包含一个二维码的图片。",
  "failure.resource-limit":
    "此项目或保险库超出大小限制。请缩减项目内容后重试。",
  "failure.passkeys-full":
    "这些密码包含的通行密钥总数超过了单个项目能容纳的上限。请先从其中一个密码中移除通行密钥后再合并。",
  "failure.breach-checks-off": "密码泄露检查已关闭。请在隐私设置中开启。",
  "failure.breach-check-unreachable":
    "Ravenpass 无法连接到泄露检查服务。请检查网络连接后重试。",
  "failure.permission-denied":
    "Ravenpass 无法访问该文件或文件夹。请选择其他位置重试。",
  "failure.file-size-invalid":
    "该文件为空或过大。请选择有效的 Ravenpass 文件。",
  "failure.file-unverified": "文件无法验证。请选择其他位置重试。",
  "failure.file-not-selected": "未选择文件。请选择一个文件以继续。",
  "failure.location-not-selected": "未能选择位置。请重试。",
  "failure.item-unreadable": "无法打开此项目。请在列表中重新选择。",
  "failure.photo-unsupported":
    "无法读取该图片。请选择 JPEG、PNG 或 WebP 照片。",
  "failure.photo-too-large":
    "该图片过大。请选择小于 32 MB 且低于 5000 万像素的图片。",
  "failure.scan-unsupported":
    "无法将该文件作为扫描件读取。请选择 JPEG、PNG、WebP 图片或 PDF。",
  "failure.scan-too-large":
    "该扫描件过大无法保存。请选择较小的图片，或小于 960 KB 的 PDF。",
  "failure.scan-limit": "此文档已有 4 份扫描件。附加新扫描件前请先移除一份。",
  "failure.field-not-copyable": "该字段无法复制。请选择其他字段。",
  "failure.field-empty": "该字段为空。复制前请先添加内容。",
  "failure.copy-failed": "无法复制内容。请打开项目手动复制。",
  "failure.website-missing": "此密码未保存网址。请先在项目中添加网址后重试。",
  "failure.website-unsupported":
    "该网址不是 Ravenpass 能打开的网络地址。请将其修改为以 https:// 开头的地址，或复制后自行打开。",
  "failure.website-open-failed": "无法打开该网站。请复制地址后自行打开。",
  "failure.export-canceled": "导出已取消。请选择保存副本的文件位置。",
  "failure.export-stale":
    "导出前保险库发生了变动。请重新打开保险库并导出新副本。",
  "failure.export-unconfirmed":
    "已保存加密副本，但 Ravenpass 无法确认其是否为最新版本。请重新打开保险库再次导出。",
  "failure.export-unrecorded":
    "已保存加密副本，但 Ravenpass 无法记录。请重新导出以便在设置中显示最新副本信息。",
  "failure.storage-unavailable":
    "Ravenpass 无法访问存放保险库的文件夹。请重新连接，或选择您的保险库文件。",
  "failure.storage-selection-invalid":
    "Ravenpass 无法读取保险库保存位置。请选择保险库文件以继续。",
  "failure.storage-occupied": "该位置已存在同名文件。请选择其他名称或文件夹。",
  "failure.storage-same-location":
    "您的保险库已存放在该处。如需移动，请选择其他文件夹。",
  "failure.storage-kind-unsupported":
    "当前版本将保险库保存在文件中。请选择文件位置。",
  "failure.storage-path-invalid":
    "该位置无法存放保险库文件。请选择其他名称或文件夹。",
  "failure.move-unverified": "新位置的保险库无法完成验证。文件保留在原处。",
  "failure.second-copy":
    "该文件是您已有保险库的另一份副本。请使用一个文件作为主保险库，并将另一个保留为加密备份。",
  "failure.older-copy":
    "此文件比该设备上使用的保险库更旧。请等待最新版本同步，或切回此前使用的保险库文件。",
  "failure.vault-diverged":
    "此保险库在其他设备上同时发生了更改。请在另一台设备上打开并解锁 Ravenpass 以继续使用该版本。",
  "failure.vault-key-replaced":
    "此保险库的恢复密钥已在其他设备上更改。请使用新恢复密钥解锁一次。",
  "failure.witness-missing":
    "此设备没有该保险库的记录。请使用恢复密钥进行恢复。",
  "failure.vault-unknown": "Ravenpass 无法识别该保险库。请重启应用并重试。",
  "failure.no-other-vault":
    "没有其他可返回的保险库。请完成此保险库的创建，或打开一个保险库文件。",
  "failure.group-unknown": "Ravenpass 无法识别该分组。请重新打开保险库后重试。",
  "failure.group-name-invalid":
    "分组名称为空、过长或已被使用。请选择其他名称。",
  "failure.language-unsupported": "Ravenpass 不支持该语言。请从列表中选择。",
  "failure.clear-delay-unsupported":
    "Ravenpass 不支持该清空延迟。请从列表中选择。",
  "failure.appearance-unsupported":
    "Ravenpass 不支持该外观模式。请从列表中选择。",
  "failure.sign-in-style-unsupported":
    "Ravenpass 不支持该登录展示方式。请从列表中选择。",
  "failure.auto-lock-delay-unsupported":
    "Ravenpass 不支持该锁定延迟。请从列表中选择。",
  "failure.backup-choice-unsupported":
    "Ravenpass 不支持该备份频率或保留数量。请从列表中选择。",
  "failure.backup-folder-missing":
    "未选择备份文件夹。请先选择文件夹，然后开启自动备份。",
  "failure.shortcut-invalid":
    "Ravenpass 无法将这些按键用作快捷键。请选择其他按键。",
  "failure.pin-invalid": "PIN 码须为 6 至 12 位数字。请输入后重试。",
  "failure.pin-wrong": "PIN 码错误。请稍候重试。",
  "failure.pin-too-soon": "连续输入错误次数过多。请等待几秒后重试。",
  "failure.pin-removed":
    "PIN 码错误次数过多已被移除。请通过其他方式解锁，或使用恢复密钥恢复访问权限。",
  "failure.pin-missing":
    "此设备上的该保险库未设置 PIN 码。请通过其他方式解锁，或在设置中添加 PIN 码。",
  "failure.biometry-unavailable":
    "此设备无法验证您的身份。请在系统设置中开启屏幕锁屏，或使用 PIN 码。",
  "failure.biometry-disabled":
    "此保险库已关闭生物识别。请使用 PIN 码，或在设置中重新开启。",
  "failure.unlock-method-required":
    "保险库须保留至少一种解锁方式。请先设置另一种方式。",
  "failure.unlock-record-unreadable":
    "Ravenpass 无法读取此保险库在此设备上的解锁方式。请使用恢复密钥恢复。",
  "failure.owner-unverified": "未确认您的身份，未做任何更改。请重试。",
  "failure.import-source-unknown":
    "Ravenpass 无法从该应用导入。请从列表中选择受支持的应用。",
  "failure.import-unrecognized":
    "Ravenpass 无法将此文件识别为 {source} 导出文件。请选择由 {source} 保存的文件。",
  "failure.import-too-large":
    "此文件过大无法导入。请导出较少项目生成较小的文件后重试。",
  "failure.import-account-bound":
    "此文件只能在 {source} 中使用您的账户打开。请设置文件密码重新导出，或以未加密方式导出。",
  "failure.import-password-wrong":
    "密码无法打开此文件。请输入导出时设置的密码。",
  "failure.import-encryption-unsupported":
    "此文件的加密格式 Ravenpass 无法打开。请从 {source} 重新导出。",
  "failure.import-not-active": "请重新选择文件进行导入。",
  "failure.import-empty":
    "没有可导入的内容：所有项目已在 Ravenpass 中，或无法被导入。",
  "failure.import-limit":
    "您的保险库无法容纳此文件中的所有内容。请关闭“将文件夹转为分组”，或从导出中移除部分项目后重试。",
  "failure.import-trash-failed": "Ravenpass 无法将文件移至废纸篓。请手动删除。",
  "failure.import-reveal-failed":
    "Ravenpass 无法打开包含该文件的文件夹。请找到该文件并手动删除。",
  "failure.link-expired": "此连接密钥已过期。请生成新密钥以关联扩展。",
  "failure.link-canceled": "关联已取消。请生成新密钥以关联扩展。",
  "failure.link-unavailable":
    "Ravenpass 无法打开供扩展连接的通道。可能其他应用正在使用该通道，请稍后重试。",
  "failure.extension-not-found":
    "该扩展已被解除关联。请重新打开设置查看仍保持关联的扩展。",
  "failure.extension-name-invalid": "名称为空或过长。最多支持 64 个字符。",
  "failure.confirmation-ended":
    "此请求已结束。请从浏览器或发起请求的应用重新发起。",
};
