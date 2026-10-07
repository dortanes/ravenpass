import type { workspace as english } from "../en/workspace.ts";

export const workspace: typeof english = {
  "workspace.status.sync.synced": "Синхронизировано",
  "workspace.status.sync.writing": "Синхронизация…",
  "workspace.status.sync.failed": "Не синхронизировано",
  "workspace.status.items":
    "{count, plural, one {# запись} few {# записи} many {# записей} other {# записи}}",
  "workspace.region.credentials": "Пароли",
  "workspace.region.identities": "Профили",
  "workspace.region.cards": "Карты",
  "workspace.region.notes": "Заметки",
  "workspace.region.seeds": "Сиды",
  "workspace.region.settings": "Настройки",

  "workspace.toolbar.search": "Поиск паролей",
  "workspace.toolbar.search.placeholder": "Поиск по названиям, логинам и почте",
  "workspace.toolbar.new": "Создать",
  "workspace.toolbar.lock": "Заблокировать хранилище",

  "workspace.rail.label": "Рабочая область",
  "workspace.rail.passwords": "Пароли",
  "workspace.rail.identities": "Профили",
  "workspace.rail.cards": "Карты",
  "workspace.rail.notes": "Заметки",
  "workspace.rail.seeds": "Сиды",
  "workspace.rail.settings": "Настройки",
  "workspace.back": "Назад — {place}",

  "workspace.palette.title": "Поиск по хранилищу",
  "workspace.palette.description":
    "Найдите пароль, профиль, карту, заметку, сид, группу, настройку или действие.",
  "workspace.palette.placeholder":
    "Поиск паролей, профилей, карт, заметок, сидов, групп и настроек",
  "workspace.palette.empty": "Ничего не найдено.",
  "workspace.palette.actions": "Действия",

  "workspace.sections.label": "Раздел",
  "workspace.section.all": "Все записи",
  "workspace.section.recent": "Недавние",
  "workspace.section.pinned": "Избранное",

  "workspace.groups.label": "Группы",
  "workspace.groups.all": "Все группы",

  "workspace.list.label": "Сохранённые пароли",
  "workspace.list.loading": "Загрузка…",
  "workspace.list.pinned": "В избранном",
  "workspace.list.resize": "Изменить ширину списка",
  "workspace.list.empty.all": "Добавленные пароли появятся здесь.",
  "workspace.list.empty.recent": "Открытые пароли появятся здесь.",
  "workspace.list.empty.pinned":
    "Пароли, добавленные в избранное, появятся здесь.",
  "workspace.list.empty.search": "Подходящих паролей нет.",

  "workspace.loading": "Загрузка паролей…",
  "workspace.detail.loading": "Открываем пароль…",
  "workspace.detail.edit": "Изменить",
  "workspace.detail.pin": "В избранное",
  "workspace.detail.unpin": "Убрать из избранного",
  "workspace.detail.duplicate": "Дублировать",
  "workspace.detail.delete": "Удалить",
  "workspace.duplicate.name": "{name} (копия)",
  "workspace.delete.detail": "Это действие нельзя отменить.",
  "workspace.delete.cancel": "Отмена",
  "workspace.delete.confirm": "Удалить",
  "workspace.delete.busy": "Удаляем…",
  "workspace.field.notes": "Заметки",
  "workspace.field.hidden": "Скрыто",
  "workspace.notes.copy": "Скопировать заметки",

  "workspace.editor.groups": "Группы",
  "workspace.editor.groups.none": "Без группы",
  "workspace.editor.groups.search": "Найти группу",
  "workspace.editor.groups.empty": "Подходящей группы нет.",
  "workspace.editor.hint.dismiss": "Понятно",
  "workspace.editor.tags": "Теги",
  "workspace.editor.tags.placeholder": "Личный, рабочий, общий",
  "workspace.editor.tags.remove": "Убрать тег {tag}",
  "workspace.editor.tags.add": "Добавить тег {tag}",
  "workspace.tags.show": "Показать всё с тегом {tag}",
  "workspace.reveal.pin": "Подтвердите, что это вы, чтобы показать «{item}».",
  "workspace.reveal.error":
    "Не удалось подтвердить, что это вы. Повторите попытку.",
  "workspace.editor.remaining":
    "{used} из {limit, plural, one {# символа} few {# символов} many {# символов} other {# символа}}",
  "workspace.editor.cancel": "Отмена",
  "workspace.editor.busy": "Сохраняем…",
  "workspace.editor.update": "Сохранить изменения",
  "workspace.date.placeholder": "ДД.ММ.ГГГГ",
  "workspace.date.choose": "Выбрать дату",

  "workspace.empty.select.title": "Выберите пароль",
  "workspace.empty.select.detail":
    "Выберите пароль в списке, чтобы увидеть подробности.",
  "workspace.empty.first.title": "Добавьте первый пароль",
  "workspace.empty.first.detail": "Добавленные пароли появятся в списке.",
  "workspace.empty.first.action": "Добавить пароль",
  "workspace.empty.first.import": "Импортировать",

  "workspace.backup.saved":
    "Зашифрованная копия сохранена. Храните её в месте, которое сможете найти позже.",

  "workspace.copy.login": "Логин скопирован.",
  "workspace.copy.email": "Адрес почты скопирован.",
  "workspace.copy.password": "Пароль скопирован.",
  "workspace.copy.totp": "Одноразовый код скопирован.",
  "workspace.copy.website": "Адрес сайта скопирован.",
  "workspace.copy.notes": "Заметки скопированы.",
  "workspace.copy.clears": "Буфер обмена очистится через {delay}.",

  "workspace.storage.moved": "Хранилище перенесено в новое место.",
  "workspace.storage.moved.copy-left":
    "Хранилище перенесено. Старая копия осталась в {location}.",
  "workspace.vault.forgotten":
    "Хранилище «{name}» убрано из списка. Его файл остался на месте.",
  "workspace.vault.deleted": "Хранилище «{name}» удалено.",
  "workspace.vault.deleted.keys-kept":
    "Хранилище «{name}» удалено, но часть его ключей осталась на этом устройстве.",

  "workspace.error.list":
    "Не удалось загрузить ваши пароли. Перезапустите приложение.",
  "workspace.error.read":
    "Не удалось загрузить часть хранилища. Заблокируйте и снова откройте хранилище, чтобы повторить попытку.",
  "workspace.error.outside-change":
    "Не удалось показать последние изменения. Заблокируйте и снова откройте хранилище, чтобы увидеть их.",
  "workspace.error.close":
    "Не удалось закрыть выбранный пароль. Заблокируйте хранилище.",
  "workspace.error.open":
    "Не удалось открыть этот пароль. Выберите его снова или перезапустите Ravenpass.",
  "workspace.error.copy":
    "Не удалось скопировать это значение. Откройте пароль и повторите.",
  "workspace.error.website":
    "Не удалось открыть сайт. Скопируйте адрес и откройте его вручную.",
  "workspace.error.clipboard":
    "Не удалось сохранить настройку буфера обмена. Повторите попытку.",
  "workspace.error.interface-size":
    "Не удалось сохранить размер интерфейса. Повторите попытку.",
  "workspace.error.appearance":
    "Не удалось сохранить оформление. Повторите попытку.",
  "workspace.error.dock-icon":
    "Не удалось сохранить настройку значка в Dock. Повторите попытку.",
  "workspace.error.auto-lock":
    "Не удалось сохранить настройку автоблокировки. Повторите попытку.",
  "workspace.error.site-icons":
    "Не удалось сохранить настройку значков сайтов. Повторите попытку.",
  "workspace.error.bank-details":
    "Не удалось сохранить настройку данных банка. Повторите попытку.",
  "workspace.error.shortcut":
    "Не удалось сохранить сочетание клавиш. Повторите попытку.",
  "workspace.error.pin":
    "Не удалось добавить этот пароль в избранное. Повторите попытку.",
  "workspace.error.unpin":
    "Не удалось убрать этот пароль из избранного. Повторите попытку.",
  "workspace.error.save":
    "Не удалось сохранить этот пароль. Проверьте хранилище и повторите.",
  "workspace.error.saved-partly": "Пароль сохранён, но {detail}",
  "workspace.error.delete":
    "Не удалось удалить этот пароль. Повторите попытку.",
  "workspace.error.duplicate":
    "Не удалось создать копию этого пароля. Повторите попытку.",
  "workspace.error.deleted-partly": "Пароль удалён, но {detail}",
  "workspace.error.refresh":
    "список не удалось обновить. Перезапустите Ravenpass и проверьте его.",
  "workspace.error.group":
    "Не удалось изменить группы. Повторите попытку или откройте хранилище заново.",
  "workspace.error.export":
    "Не удалось сохранить зашифрованную копию. Выберите другое место и повторите.",
  "workspace.error.storage-move":
    "Не удалось перенести хранилище. Оно осталось на прежнем месте.",
  "workspace.error.vault-switch":
    "Не удалось открыть это хранилище. Повторите попытку или выберите другое.",
  "workspace.error.vault-create":
    "Не удалось начать создание хранилища. Повторите попытку.",
  "workspace.error.vault-open":
    "Этот файл не удалось открыть как хранилище. Выберите другой файл.",
  "workspace.error.vault-forget":
    "Не удалось убрать это хранилище из списка. Повторите попытку.",
  "workspace.error.vault-delete":
    "Не удалось удалить это хранилище. Его файл остался на месте.",
  "workspace.error.lock":
    "Не удалось заблокировать хранилище. Перезапустите Ravenpass, чтобы заблокировать его.",

  "vault-menu.current": "Хранилище",
  "vault-menu.heading": "Хранилища",
  "vault-menu.new": "Новое хранилище…",
  "vault-menu.open": "Открыть файл хранилища…",

  "credential.untitled": "Пароль без названия",
  "credential.login.empty": "Логин не сохранён",
  "credential.website.empty": "Сайт не сохранён",
  "credential.website.open": "Открыть сайт",

  "credential.field.label": "Название",
  "credential.field.website": "Сайт",
  "credential.field.login": "Логин",
  "credential.field.email": "Почта",
  "credential.field.password": "Пароль",
  "credential.field.totp": "Одноразовый код",

  "credential.totp.placeholder":
    "Ключ настройки или ссылка для одноразового кода",
  "credential.totp.unavailable": "Сейчас кода нет",

  "credential.copy.login": "Скопировать логин",
  "credential.copy.email": "Скопировать почту",
  "credential.copy.password": "Скопировать пароль",
  "credential.copy.totp": "Скопировать одноразовый код",
  "credential.copy.website": "Скопировать {address}",
  "credential.password.reveal": "Показать пароль",
  "credential.password.conceal": "Скрыть пароль",
  "credential.totp.reveal": "Показать настройку одноразового кода",
  "credential.totp.conceal": "Скрыть настройку одноразового кода",
  "credential.totp.qr": "Прочитать QR-код",
  "credential.totp.qr.file": "Из файла с изображением",
  "credential.totp.qr.clipboard": "Из буфера обмена",

  "credential.delete.title": "Удалить этот пароль?",

  "credential.passkeys": "Ключи доступа",
  "credential.mark.totp": "Есть одноразовый код",
  "credential.mark.passkey": "Есть ключ доступа",
  "credential.passkey.created": "Создан {date}",
  "credential.apps": "Приложения",

  "credential.editor.new.title": "Новый пароль",
  "credential.editor.edit.title": "Изменить пароль",
  "credential.editor.requirement": "Укажите название и пароль",
  "credential.editor.requirement.passkeys": "Укажите название",
  "credential.editor.create": "Добавить пароль",
  "credential.editor.add": "Добавить в пароль",
  "credential.site.looking-up": "Ищем сайт…",
  "credential.hint.site":
    "Введите или вставьте адрес сайта, чтобы автоматически заполнить название.",
  "credential.error.lookup":
    "Не удалось получить название сайта. Введите название вручную.",
  "credential.error.qr": "Не удалось прочитать QR-код. Повторите попытку.",
  "credential.remove.website": "Удалить сайт",
  "credential.remove.passkey": "Удалить ключ доступа",
  "credential.remove.app": "Удалить приложение",
};
