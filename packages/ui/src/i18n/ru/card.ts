import type { card as english } from "../en/card.ts";

export const card: typeof english = {
  "card.untitled": "Карта без названия",
  "card.search.label": "Поиск карт",
  "card.search.placeholder": "Поиск по названиям, банкам и последним цифрам",

  "card.list.label": "Сохранённые карты",
  "card.list.empty.all": "Добавленные карты появятся здесь.",
  "card.list.empty.recent": "Открытые карты появятся здесь.",
  "card.list.empty.pinned": "Карты, добавленные в избранное, появятся здесь.",
  "card.list.empty.search": "Подходящих карт нет.",

  "card.loading": "Загрузка карт…",
  "card.detail.loading": "Открываем карту…",
  "card.empty.select.title": "Выберите карту",
  "card.empty.select.detail":
    "Выберите карту в списке, чтобы увидеть подробности.",
  "card.empty.first.title": "Добавьте первую карту",
  "card.empty.first.detail": "Добавленные карты появятся в списке.",
  "card.empty.first.action": "Добавить карту",

  "card.block.card": "Карта",
  "card.block.bank": "Банк",
  "card.block.billing": "Платёжный адрес",

  "card.field.label": "Название",
  "card.field.label.placeholder": "Основная",
  "card.field.number": "Номер",
  "card.field.holder": "Владелец карты",
  "card.field.expiry": "Срок действия",
  "card.field.expiry.placeholder": "ММ/ГГ",
  "card.field.security-code": "CVV/CVC",
  "card.field.pin": "Пин-код",
  "card.field.bank-site": "Сайт банка",
  "card.field.bank-site.placeholder": "mybank.ru",
  "card.field.bank-name": "Банк",
  "card.field.color": "Цвет",

  "card.color.automatic": "Автоматически",
  "card.color.swatch": "Цвет {number}",
  "card.billing.none": "Нет",
  "card.billing.own": "Мой адрес",

  "card.face.back": "Перевернуть и показать код безопасности",
  "card.face.front": "Вернуть лицевую сторону",
  "card.face.expiry": "Срок",
  "card.validity.title": "Срок действия · {expiry}",
  "card.validity.expires": "До {date} · {left}",
  "card.validity.expired": "Истекла {date}",
  "card.validity.none": "Без срока действия",
  "card.bank.open": "Открыть сайт банка",
  "card.bank.looking-up": "Ищем банк…",
  "card.hint.bank-site":
    "Укажите сайт банка, чтобы автоматически заполнить его название и цвет.",

  "card.number.reveal": "Показать номер",
  "card.number.conceal": "Скрыть номер",
  "card.number.mistyped": "Похоже, в номере опечатка",
  "card.security-code.reveal": "Показать код безопасности",
  "card.security-code.conceal": "Скрыть код безопасности",
  "card.pin.reveal": "Показать пин-код",
  "card.pin.conceal": "Скрыть пин-код",

  "card.copy.number": "Скопировать номер",
  "card.copy.holder": "Скопировать имя владельца",
  "card.copy.expiry": "Скопировать срок действия",
  "card.copy.security-code": "Скопировать код безопасности",
  "card.copy.pin": "Скопировать пин-код",
  "card.copy.bank-name": "Скопировать название банка",
  "card.copy.bank-site": "Скопировать сайт банка",
  "card.copied.number": "Номер карты скопирован.",
  "card.copied.holder": "Имя владельца скопировано.",
  "card.copied.expiry": "Срок действия скопирован.",
  "card.copied.security-code": "Код безопасности скопирован.",
  "card.copied.pin": "Пин-код скопирован.",
  "card.copied.bank-name": "Название банка скопировано.",
  "card.copied.bank-site": "Сайт банка скопирован.",

  "card.delete.title": "Удалить эту карту?",

  "card.editor.new.title": "Новая карта",
  "card.editor.edit.title": "Изменить карту",
  "card.editor.create": "Добавить карту",
  "card.editor.requirement": "Укажите название и номер карты",
  "card.editor.unfinished":
    "Исправьте или очистите выделенные поля, чтобы сохранить.",

  "card.error.list":
    "Не удалось загрузить ваши карты. Перезапустите приложение.",
  "card.error.close":
    "Не удалось закрыть выбранную карту. Заблокируйте хранилище.",
  "card.error.open":
    "Не удалось открыть эту карту. Выберите её снова или перезапустите Ravenpass.",
  "card.error.copy":
    "Не удалось скопировать это значение. Откройте карту и повторите.",
  "card.error.pin":
    "Не удалось добавить эту карту в избранное. Повторите попытку.",
  "card.error.unpin":
    "Не удалось убрать эту карту из избранного. Повторите попытку.",
  "card.error.save":
    "Не удалось сохранить эту карту. Проверьте введённые данные и повторите.",
  "card.error.saved-partly": "Карта сохранена, но {detail}",
  "card.error.delete": "Не удалось удалить эту карту. Повторите попытку.",
  "card.error.duplicate":
    "Не удалось создать копию этой карты. Повторите попытку.",
  "card.error.deleted-partly": "Карта удалена, но {detail}",
  "card.error.lookup":
    "Не удалось получить данные с сайта банка. Введите название банка вручную.",
  "card.error.addresses":
    "Не удалось получить адреса из ваших профилей. Введите платёжный адрес вручную.",
};
