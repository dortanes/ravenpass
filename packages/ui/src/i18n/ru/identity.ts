import type { identity as english } from "../en/identity.ts";

export const identity: typeof english = {
  "identity.untitled": "Профиль без названия",
  "identity.email.empty": "Почта не сохранена",
  "identity.search.label": "Поиск профилей",
  "identity.search.placeholder": "Поиск по названиям и почте",

  "identity.list.label": "Сохранённые профили",
  "identity.list.expiring": "Срок документа истекает {relative}",
  "identity.list.expired": "Срок документа истёк",
  "identity.list.empty.all": "Добавленные профили появятся здесь.",
  "identity.list.empty.recent": "Открытые профили появятся здесь.",
  "identity.list.empty.pinned":
    "Профили, добавленные в избранное, появятся здесь.",
  "identity.list.empty.search": "Подходящих профилей нет.",

  "identity.loading": "Загрузка профилей…",
  "identity.detail.loading": "Открываем профиль…",
  "identity.empty.select.title": "Выберите профиль",
  "identity.empty.select.detail":
    "Выберите профиль в списке, чтобы увидеть подробности.",
  "identity.empty.first.title": "Добавьте первый профиль",
  "identity.empty.first.detail": "Добавленные профили появятся в списке.",
  "identity.empty.first.action": "Добавить профиль",

  "identity.block.personal": "Личное",
  "identity.block.contact": "Контакты",
  "identity.block.addresses": "Адреса",
  "identity.block.documents": "Документы",

  "identity.field.label": "Название",
  "identity.field.label.placeholder": "Личный",
  "identity.field.full-name": "Полное имя",
  "identity.field.birthday": "Дата рождения",
  "identity.field.email": "Почта",
  "identity.field.phone": "Телефон",
  "identity.field.address": "Адрес",
  "identity.field.address-name": "Название",
  "identity.field.address-name.placeholder": "Дом",
  "identity.field.street": "Улица, дом",
  "identity.field.city": "Город",
  "identity.field.region": "Регион",
  "identity.field.postal-code": "Индекс",
  "identity.field.country": "Страна",
  "identity.field.document-name": "Название",
  "identity.field.document-name.placeholder": "Читательский билет",
  "identity.field.number": "Номер",
  "identity.field.issuer": "Кем выдан",
  "identity.field.issued-on": "Выдан",
  "identity.field.expires-on": "Действует до",

  "identity.document.passport": "Паспорт",
  "identity.document.drivers-license": "Водительское удостоверение",
  "identity.document.id-card": "Удостоверение личности",
  "identity.document.tax-number": "Налоговый номер",
  "identity.document.other": "Другой документ",
  "identity.document.expires": "Действует до {date} · {left}",
  "identity.document.expired": "Истёк {date}",
  "identity.address.untitled": "Адрес {number}",
  "identity.document.issuer": "Выдан: {issuer}",
  "identity.document.no-expiry": "Без срока действия",

  "identity.copy.full-name": "Скопировать полное имя",
  "identity.copy.birthday": "Скопировать дату рождения",
  "identity.copy.email": "Скопировать почту",
  "identity.copy.phone": "Скопировать телефон",
  "identity.copy.address": "Скопировать адрес",
  "identity.copy.street": "Скопировать улицу",
  "identity.copy.city": "Скопировать город",
  "identity.copy.region": "Скопировать регион",
  "identity.copy.postal-code": "Скопировать индекс",
  "identity.copy.country": "Скопировать страну",
  "identity.copy.number": "Скопировать номер",
  "identity.number.reveal": "Показать номер",
  "identity.number.conceal": "Скрыть номер",
  "identity.copied.full-name": "Полное имя скопировано.",
  "identity.copied.birthday": "Дата рождения скопирована.",
  "identity.copied.phone": "Телефон скопирован.",
  "identity.copied.address": "Адрес скопирован.",
  "identity.copied.street": "Улица скопирована.",
  "identity.copied.city": "Город скопирован.",
  "identity.copied.region": "Регион скопирован.",
  "identity.copied.postal-code": "Индекс скопирован.",
  "identity.copied.country": "Страна скопирована.",
  "identity.copied.number": "Номер скопирован.",

  "identity.scan.label": "Сканы",
  "identity.scan.attach": "Прикрепить скан",
  "identity.scan.from-photos": "Из фотографий",
  "identity.scan.from-files": "Из файлов",
  "identity.scan.actions": "Действия для «{name}»",
  "identity.scan.copy": "Скопировать",
  "identity.scan.save": "Сохранить копию",
  "identity.scan.remove": "Убрать «{name}»",
  "identity.scan.saved": "Копия сохранена.",
  "identity.scan.save.title": "Сохранить незашифрованную копию?",
  "identity.scan.save.detail":
    "Копия сохранится вне хранилища без шифрования. Любой, кто откроет файл, сможет увидеть скан.",
  "identity.scan.save.confirm": "Сохранить копию",
  "identity.copied.scan": "Скан скопирован.",

  "identity.editor.drop-scans.title": "Удалить сканы убранных документов?",
  "identity.editor.drop-scans.detail":
    "При сохранении {count, plural, one {будет удалён # скан} few {будут удалены # скана} many {будут удалены # сканов} other {будут удалены # скана}} этих документов. Это действие нельзя отменить.",
  "identity.editor.drop-scans.confirm": "Сохранить и удалить",

  "identity.photo.choose": "Выбрать фото",
  "identity.photo.remove": "Убрать фото",
  "identity.photo.crop.title": "Кадрирование фото",
  "identity.photo.crop.detail":
    "Перетащите фото и измените масштаб, чтобы лицо оказалось в круге.",
  "identity.photo.zoom": "Масштаб",
  "identity.photo.cancel": "Отмена",
  "identity.photo.use": "Использовать фото",
  "identity.photo.using": "Кадрируем…",

  "identity.editor.new.title": "Новый профиль",
  "identity.editor.edit.title": "Изменить профиль",
  "identity.editor.create": "Добавить профиль",
  "identity.editor.requirement":
    "Укажите название профиля и каждого документа типа «Другой документ»",
  "identity.editor.unfinished-date":
    "Допишите или очистите выделенную дату, чтобы сохранить.",
  "identity.editor.add": "Добавить в профиль",
  "identity.remove.email": "Удалить почту",
  "identity.remove.phone": "Удалить телефон",
  "identity.remove.address": "Удалить адрес",
  "identity.remove.document": "Удалить документ",

  "identity.error.list":
    "Не удалось загрузить ваши профили. Перезапустите приложение.",
  "identity.error.close":
    "Не удалось закрыть выбранный профиль. Заблокируйте хранилище.",
  "identity.error.open":
    "Не удалось открыть этот профиль. Выберите его снова или перезапустите Ravenpass.",
  "identity.error.copy":
    "Не удалось скопировать это значение. Откройте профиль и повторите.",
  "identity.error.pin":
    "Не удалось добавить этот профиль в избранное. Повторите попытку.",
  "identity.error.unpin":
    "Не удалось убрать этот профиль из избранного. Повторите попытку.",
  "identity.error.save":
    "Не удалось сохранить этот профиль. Проверьте введённые данные и повторите.",
  "identity.error.saved-partly": "Профиль сохранён, но {detail}",
  "identity.error.delete":
    "Не удалось удалить этот профиль. Повторите попытку.",
  "identity.error.duplicate":
    "Не удалось создать копию этого профиля. Повторите попытку.",
  "identity.error.deleted-partly": "Профиль удалён, но {detail}",
  "identity.error.scan-attach":
    "Не удалось прикрепить этот файл. Выберите изображение JPEG, PNG или WebP либо файл PDF.",
  "identity.error.scan-save":
    "Не удалось сохранить копию. Выберите другое место и повторите.",
  "identity.error.scan-discard":
    "Не удалось выгрузить выбранные сканы из памяти. Заблокируйте хранилище, чтобы очистить их.",
  "identity.error.photo":
    "Это изображение не подходит. Выберите фото в формате JPEG, PNG или WebP.",
  "identity.error.photo-crop":
    "Не удалось кадрировать фото. Выберите его снова.",
  "identity.error.photo-discard":
    "Не удалось выгрузить выбранное изображение из памяти. Заблокируйте хранилище, чтобы очистить его.",
};
