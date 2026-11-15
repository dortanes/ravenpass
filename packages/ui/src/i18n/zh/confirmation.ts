import type { confirmation as english } from "../en/confirmation.ts";

export const confirmation: typeof english = {
  "confirmation.unlock.title": "解锁 Ravenpass",
  "confirmation.unlock.description": "一个浏览器扩展想要打开你的密码库。",
  "confirmation.unlock.description.autofill":
    "自动填充想要打开你的密码库以便登录。",
  "confirmation.change-unlock.title": "更改密码库的解锁方式？",
  "confirmation.change-unlock.description": "输入当前的 PIN 码以确认此更改。",
  "confirmation.change-unlock.confirm": "确认",
  "confirmation.change-unlock.error": "无法确认此更改。请重试。",
  "confirmation.decline": "拒绝",
  "confirmation.decline-error": "Ravenpass 无法拒绝此请求。请重试。",
};
