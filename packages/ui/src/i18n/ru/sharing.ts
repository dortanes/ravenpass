import type { sharing as english } from "../en/sharing.ts";

export const sharing: typeof english = {
  "sharing.prompt.title": "Передать «{file}»?",
  "sharing.prompt.description":
    "Введите пин-код, чтобы передать файл из профиля «{identity}» на сайт {site}.",
  "sharing.prompt.share": "Передать",
  "sharing.prompt.error": "Не удалось передать файл. Повторите попытку.",
  "sharing.prompt.save-passkey.title": "Сохранить ключ доступа для {site}?",
  "sharing.prompt.save-passkey.description":
    "Введите пин-код, чтобы сохранить ключ доступа для {site} в Ravenpass.",
  "sharing.prompt.save-passkey.confirm": "Сохранить",
  "sharing.prompt.sign-in.title": "Войти на {site}?",
  "sharing.prompt.sign-in.description":
    "Введите пин-код, чтобы войти на {site} как {account} с ключом доступа.",
  "sharing.prompt.sign-in.description.no-account":
    "Введите пин-код, чтобы войти на {site} с ключом доступа.",
  "sharing.prompt.sign-in.confirm": "Войти",
  "sharing.prompt.fill.title": "Вставить данные входа на {site}?",
  "sharing.prompt.fill.description":
    "Введите пин-код, чтобы расширение браузера вставило данные входа «{account}» на {site}.",
  "sharing.prompt.fill.confirm": "Вставить",
  "sharing.prompt.fill-card.title": "Вставить «{file}» на {site}?",
  "sharing.prompt.fill-card.description":
    "Введите пин-код, чтобы расширение браузера вставило эту карту на {site}.",
  "sharing.prompt.passkey.error":
    "Не удалось подтвердить запрос. Повторите попытку.",
};
