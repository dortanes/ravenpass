import type { settings as english } from "../en/settings.ts";

export const settings: typeof english = {
  "settings.nav.label": "Разделы настроек",
  "settings.nav.heading": "Настройки",
  "settings.general.heading": "Основные",
  "settings.general.summary": "Язык и внешний вид.",
  "settings.general.language": "Язык",
  "settings.general.interface": "Интерфейс",
  "settings.general.macos": "macOS",
  "settings.interface-size": "Размер интерфейса",
  "settings.interface-size.natural": "{size} (по умолчанию)",
  "settings.appearance": "Оформление",
  "settings.appearance.system": "Как в системе",
  "settings.appearance.light": "Светлое",
  "settings.appearance.dark": "Тёмное",
  "settings.launch-at-login": "Открывать при входе в систему",
  "settings.launch-at-login.blocked":
    "Заблокировано в Системных настройках. Разрешите Ravenpass в разделе «Объекты входа».",
  "settings.launch-at-login.open": "Открыть «Объекты входа»",
  "settings.launch-at-login.hidden": "Запускать только в строке меню",
  "settings.dock-icon": "Скрывать значок в Dock при закрытии окна",
  "settings.dock-icon.detail": "Ravenpass остаётся в строке меню.",
  "settings.unlock.heading": "Разблокировка хранилища",
  "settings.locking.heading": "Автоблокировка",
  "settings.auto-lock": "Блокировать при бездействии",
  "settings.auto-lock.hidden": "Блокировать при переходе в другое приложение",
  "settings.auto-lock.delay": "Через",
  "settings.delay.immediately": "Сразу",
  "settings.auto-lock.note":
    "Ravenpass также блокируется, когда устройство переходит в режим сна.",
  "settings.auto-lock.note.hidden":
    "Ravenpass также блокируется, когда устройство засыпает. После разблокировки для автозаполнения хранилище остаётся открытым не менее 3 минут после последнего заполнения, а при более долгой задержке блокировки — дольше.",
  "settings.auto-lock.off.title": "Выключить автоблокировку?",
  "settings.auto-lock.off.detail":
    "Хранилище останется открытым, пока вы не заблокируете его, не закроете Ravenpass или устройство не перейдёт в режим сна.",
  "settings.auto-lock.off.confirm": "Выключить",
  "settings.auto-lock.off.cancel": "Отмена",
  "settings.recovery-key.heading": "Ключ восстановления",
  "settings.recovery-key.title": "Сменить ключ восстановления",
  "settings.recovery-key.detail":
    "Другие устройства один раз запросят новый ключ. Прежние резервные копии открываются только текущим ключом.",
  "settings.recovery-key.change": "Сменить…",
  "recovery-key-change.steps.verify": "Подтверждение",
  "recovery-key-change.steps.phrase": "Новый ключ",
  "recovery-key-change.steps.confirm": "Проверка ключа",
  "recovery-key-change.verify.title": "Смена ключа восстановления",
  "recovery-key-change.verify.description":
    "Ravenpass создаст новый ключ восстановления и зашифрует им хранилище.",
  "recovery-key-change.verify.devices":
    "Если это хранилище открыто на других устройствах, то там нужно будет ввести новый ключ.",
  "recovery-key-change.verify.backups":
    "Резервные копии, сделанные до смены ключа, открываются только старым ключом. Сохраните его.",
  "recovery-key-change.verify.this-device.both":
    "На этом устройстве вводить новый ключ не нужно: пин-код и биометрия продолжат работать.",
  "recovery-key-change.verify.this-device.pin":
    "На этом устройстве вводить новый ключ не нужно: пин-код продолжит работать.",
  "recovery-key-change.verify.this-device.biometry":
    "На этом устройстве вводить новый ключ не нужно: биометрия продолжит работать.",
  "recovery-key-change.verify.this-device.none":
    "На этом устройстве хранилище будет открываться новым ключом.",
  "recovery-key-change.verify.pin": "Введите пин-код, чтобы продолжить.",
  "recovery-key-change.verify.current":
    "Подтвердить вашу личность пин-кодом или биометрией на этом устройстве нельзя, поэтому введите текущий ключ восстановления.",
  "recovery-key-change.verify.pin-then-biometry":
    "Введите пин-код, чтобы он продолжил работать с новым ключом. Затем подтвердите личность биометрией.",
  "recovery-key-change.verify.busy": "Проверяем…",
  "recovery-key-change.phrase.title": "Сохраните новый ключ восстановления",
  "recovery-key-change.phrase.description":
    "Хранилище перейдёт на этот ключ, когда вы его проверите.",
  "recovery-key-change.confirm.title": "Проверьте новый ключ восстановления",
  "recovery-key-change.confirm.description":
    "Введите три слова из только что сохранённого ключа.",
  "recovery-key-change.saving": "Переводим хранилище на новый ключ",
  "recovery-key-change.done": "Ключ восстановления изменён.",
  "recovery-key-change.errors.begin-failed":
    "Не удалось создать новый ключ восстановления. Повторите попытку.",
  "recovery-key-change.errors.finish-failed":
    "Не удалось сменить ключ восстановления. Повторите попытку.",
  "recovery-key-change.errors.cancel-failed":
    "Не удалось отменить новый ключ восстановления. Текущий ключ остаётся в силе. Перезапустите приложение.",
  "settings.clipboard.clear": "Удалять скопированные данные",
  "settings.clipboard.clear.detail":
    "Буфер очищается, только если в нём всё ещё находятся данные, скопированные из Ravenpass.",
  "settings.clipboard.delay": "Удалять через",
  "settings.clipboard.note":
    "При блокировке хранилища буфер обмена тоже очищается.",
  "settings.clipboard.note.hidden":
    "При блокировке хранилища буфер обмена очищается, кроме блокировки при переходе в другое приложение.",
  "settings.screenshots": "Разрешить снимки экрана",
  "settings.screenshots.detail":
    "Хранилище также будет видно в недавних приложениях. Экраны автозаполнения останутся скрытыми.",
  "settings.screenshots.error":
    "Не удалось изменить настройку снимков экрана. Повторите попытку.",
  "settings.site-icons": "Значок сайта",
  "settings.site-icons.detail":
    "Загружать значки с сайтов, сохранённых в хранилище.",
  "settings.site-icons.note":
    "При выключении загруженные значки удаляются с этого устройства.",
  "settings.bank-details": "Название и цвет банка",
  "settings.bank-details.detail":
    "Загружать с сайта банка, указанного в карте.",
  "settings.groups.heading": "Группы",
  "settings.groups.summary": "Распределяйте записи по вкладкам.",
  "settings.groups.count":
    "{count, plural, one {# запись} few {# записи} many {# записей} other {# записи}}",
  "settings.groups.empty": "Добавленные группы появятся здесь.",
  "settings.groups.new": "Новая группа…",
  "settings.groups.rename": "Переименовать…",
  "settings.groups.delete": "Удалить группу…",
  "settings.groups.name": "Название",
  "settings.groups.name.placeholder": "Работа",
  "settings.groups.create.title": "Новая группа",
  "settings.groups.rename.title": "Переименование группы",
  "settings.groups.save": "Сохранить",
  "settings.groups.cancel": "Отмена",
  "settings.groups.delete.title": "Удалить «{name}»?",
  "settings.groups.delete.detail":
    "Записи из этой группы останутся в хранилище.",
  "settings.groups.delete.confirm": "Удалить группу",
  "settings.groups.default.label": "Группа для новых записей",
  "settings.groups.default.none": "Без группы",
  "settings.shortcuts.heading": "Сочетания клавиш",
  "settings.shortcuts.summary": "Доступны, пока хранилище открыто.",
  "settings.shortcuts.palette": "Поиск по всему хранилищу",
  "settings.shortcuts.search": "Поиск в текущем списке",
  "settings.shortcuts.change": "Изменить сочетание для «{action}»",
  "settings.shortcuts.reset": "Вернуть сочетание по умолчанию для «{action}»",
  "settings.shortcuts.recording": "Нажмите клавиши…",
  "settings.shortcuts.note":
    "Выберите сочетание и нажмите новые клавиши. Esc — отмена, Delete — сброс.",
  "settings.shortcuts.bare":
    "Добавьте клавишу-модификатор, чтобы сочетание не срабатывало при наборе текста.",
  "settings.shortcuts.reserved":
    "{keys} используется системой или редактором текста. Выберите другое сочетание.",
  "settings.shortcuts.taken":
    "{keys} уже назначено для действия «{action}». Выберите другое сочетание.",
  "settings.security.heading": "Разблокировка",
  "settings.security.summary": "Способы разблокировки и автоблокировка.",
  "settings.autofill.heading": "Автозаполнение",
  "settings.autofill.summary":
    "Подставляйте пароли, ключи доступа и коды в приложениях и браузерах.",
  "settings.autofill.system.autofill": "Служба автозаполнения",
  "settings.autofill.system.autofill.detail":
    "Подставляет пароли и коды в приложениях и браузерах.",
  "settings.autofill.system.passkeys": "Поставщик ключей доступа",
  "settings.autofill.system.passkeys.detail":
    "Позволяет входить по ключам доступа в приложениях и браузерах.",
  "settings.autofill.system.on": "Включено",
  "settings.autofill.system.off": "Выключено",
  "settings.autofill.system.open": "Открыть настройки Android",
  "settings.autofill.system.chrome":
    "В Chrome также включите «Автозаполнение с помощью другого сервиса».",
  "settings.autofill.system.error":
    "Не удалось открыть настройки Android. Повторите попытку.",
  "settings.autofill.suggestions.title": "Системное автозаполнение",
  "settings.autofill.suggestions": "Предлагать сохранённые аккаунты",
  "settings.autofill.suggestions.detail":
    "Показывать аккаунты под полями входа, не открывая Ravenpass.",
  "settings.autofill.suggestions.note":
    "macOS сохраняет адреса сайтов и имена пользователей вне зашифрованного хранилища. Пароли, коды и закрытые ключи для входа остаются в Ravenpass.",
  "settings.autofill.error.suggestions":
    "Не удалось изменить подсказки аккаунтов. Повторите попытку.",
  "settings.extensions.title": "Привязанные расширения браузера",
  "settings.extensions.linked": "Привязано {date}",
  "settings.extensions.unlink": "Отвязать…",
  "settings.extensions.unlink.action": "Отвязать {name}, привязано {date}",
  "settings.extensions.unlink.title": "Отвязать «{name}»?",
  "settings.extensions.unlink.detail":
    "Это расширение, привязанное {date}, перестанет подставлять пароли из Ravenpass. Остальные привязанные расширения продолжат работать.",
  "settings.extensions.unlink.confirm": "Отвязать",
  "settings.extensions.unlink.cancel": "Отмена",
  "settings.extensions.rename": "Переименовать…",
  "settings.extensions.rename.action": "Переименовать {name}",
  "settings.extensions.rename.title": "Переименование расширения",
  "settings.extensions.rename.save": "Сохранить",
  "settings.extensions.rename.cancel": "Отмена",
  "settings.extensions.name": "Название",
  "settings.extensions.link": "Привязать расширение…",
  "settings.extensions.empty":
    "Здесь появятся расширения браузера, которые вы привяжете.",
  "settings.extensions.sign-in": "Подсказка для входа",
  "settings.extensions.sign-in.card": "Карточка «Войти как…»",
  "settings.extensions.sign-in.field": "Меню под полем",
  "settings.extensions.confirm-fills":
    "Подтверждать вставку пароля из расширения",
  "settings.extensions.confirm-fills.detail":
    "Спрашивать, прежде чем расширение вставит пароль или одноразовый код.",
  "settings.extensions.confirm-fills.unavailable":
    "Чтобы включить, задайте пин-код или включите биометрию.",
  "settings.extensions.unreachable":
    "Привязанные расширения сейчас не могут связаться с Ravenpass. Возможно, его подключение занято другим приложением.",
  "settings.extensions.dialog.title": "Привязка расширения браузера",
  "settings.extensions.dialog.step.open":
    "Откройте расширение Ravenpass в браузере.",
  "settings.extensions.dialog.step.paste": "Вставьте этот ключ.",
  "settings.extensions.dialog.step.finish": "Привязка завершится здесь сама.",
  "settings.extensions.dialog.key": "Одноразовый ключ",
  "settings.extensions.dialog.copy": "Копировать",
  "settings.extensions.dialog.copy.label": "Копировать ключ",
  "settings.extensions.dialog.copied": "Скопировано",
  "settings.extensions.dialog.expires": "Истекает через {time}",
  "settings.extensions.dialog.expired": "Срок действия ключа истёк.",
  "settings.extensions.dialog.renew": "Получить новый ключ",
  "settings.extensions.dialog.cancel": "Отмена",
  "settings.extensions.linked-notice": "Расширение «{name}» привязано.",
  "settings.extensions.error.link":
    "Не удалось привязать расширение. Повторите попытку.",
  "settings.extensions.error.unlink":
    "Не удалось отвязать расширение. Повторите попытку.",
  "settings.extensions.error.copy":
    "Не удалось скопировать ключ. Выделите его и скопируйте вручную.",
  "settings.extensions.error.sign-in":
    "Не удалось изменить способ входа на сайтах. Повторите попытку.",
  "settings.extensions.error.confirm-fills":
    "Не удалось изменить подтверждение вставки. Повторите попытку.",
  "settings.extensions.error.rename":
    "Не удалось переименовать расширение. Повторите попытку.",
  "settings.extensions.error.load":
    "Не удалось загрузить привязанные расширения. Откройте этот раздел снова.",
  "settings.storage.type": "Тип",
  "settings.storage.file": "Файл хранилища",
  "settings.storage.checking": "Проверяем…",
  "settings.storage.move": "Перенести…",
  "settings.storage.moving": "Переносим…",
  "settings.storage.unrestricted":
    "В этом расположении нельзя ограничить доступ к файлу.",
  "settings.storage.shared.title": "Перенести хранилище?",
  "settings.storage.shared.detail":
    "После переноса файл {location} будет удалён. Другие устройства, которые открывают хранилище оттуда, потеряют доступ, пока вы не откроете на каждом из них перенесённый файл.",
  "settings.storage.shared.confirm": "Перенести",
  "settings.storage.shared.cancel": "Отмена",
  "settings.vaults.heading": "Хранилище",
  "settings.vaults.summary": "Хранение, резервные копии, группы и импорт.",
  "settings.vaults.list": "Хранилища на этом устройстве",
  "settings.vaults.current": "Открыто",
  "settings.vaults.actions": "Действия для «{name}»",
  "settings.vaults.forget": "Убрать из списка",
  "settings.vaults.delete": "Удалить навсегда…",
  "settings.vaults.note":
    "Если убрать хранилище из списка, его файл останется.",
  "settings.delete.title": "Удалить «{name}»?",
  "settings.delete.description":
    "Будут стёрты {location} и ключи к нему на этом устройстве. После этого записи можно будет вернуть только из зашифрованной копии с ключом восстановления.",
  "settings.delete.cancel": "Отмена",
  "settings.delete.confirm": "Удалить хранилище",
  "settings.backup.state.current": "Содержит все последние изменения.",
  "settings.backup.state.stale":
    "Сделана до последнего изменения. Сохраните новую копию.",
  "settings.backup.state.unknown": "Нет сведений о последней копии.",
  "settings.backup.export": "Сохранить копию…",
  "settings.backup.exporting": "Сохраняем…",
  "storage.type.local-file": "Файл на этом устройстве",
  "storage.type.local-file.detail":
    "Зашифрованный файл в папке, которую вы выберете.",
  "storage.type.local-file.private":
    "Зашифрованный файл, который может прочитать только Ravenpass.",
  "storage.type.document": "Файл на ваш выбор",
  "storage.type.document.detail":
    "Google Диск, Dropbox или папка на этом устройстве.",
  "storage.type.document.concurrent":
    "Изменения, внесённые на двух устройствах одновременно, нельзя объединить.",
  "settings.privacy.heading": "Конфиденциальность",
  "settings.privacy.summary": "Данные с сайтов и буфер обмена.",
  "settings.privacy.clipboard": "Буфер обмена",
  "settings.privacy.websites": "Данные сайтов",
  "settings.storage.heading": "Хранение",
  "settings.backup.heading": "Резервные копии",
  "settings.backup.summary": "Сохранение зашифрованной копии хранилища.",
  "settings.backup.copy": "Зашифрованная копия",
  "settings.backup.auto": "Автоматические копии",
  "settings.backup.frequency": "Частота",
  "settings.backup.frequency.daily": "Каждый день",
  "settings.backup.frequency.weekly": "Каждую неделю",
  "settings.backup.frequency.monthly": "Каждый месяц",
  "settings.backup.keep": "Хранить",
  "settings.backup.keep.latest": "Только последнюю",
  "settings.backup.keep.count": "Последние {count}",
  "settings.backup.folder": "Папка для копий",
  "settings.backup.folder.empty": "Папка не выбрана",
  "settings.backup.folder.choose": "Выбрать…",
  "settings.backup.folder.change": "Изменить…",
  "settings.backup.note.failed":
    "Не удалось сохранить последнюю копию. Проверьте, доступна ли папка для копий, или выберите другую.",
  "settings.backup.note.last": "Последняя копия: {date}",
  "settings.backup.note.open":
    "Автоматические копии сохраняются, пока хранилище открыто.",
  "settings.backup.error":
    "Не удалось изменить настройки автоматических копий. Повторите попытку.",
  "settings.about.heading": "О программе",
  "settings.about.summary": "Версия, лицензия и поддержка.",
  "settings.about.version": "Версия",
  "settings.about.build": "Сборка",
  "settings.about.source": "Исходный код",
  "settings.about.release-notes": "Что нового в этой версии",
  "settings.about.report-problem": "Сообщить о проблеме",
  "settings.about.report-problem.detail":
    "Откроется GitHub, версии приложения и системы уже будут указаны.",
  "settings.about.report-vulnerability": "Сообщить об уязвимости",
  "settings.about.report-vulnerability.detail":
    "Отчёт увидят только разработчики.",
  "settings.about.license": "Лицензия и стороннее ПО",
  "settings.about.license.detail":
    "Ravenpass — свободное ПО под лицензией GPL-3.0-or-later.",
  "settings.about.license.missing": "Текст лицензии недоступен в этой сборке.",
  "settings.about.license.online": "Открыть лицензию на GitHub",
  "settings.about.license.close": "Закрыть",
  "settings.about.donate": "Поддержать Ravenpass",
};
