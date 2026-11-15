import type { confirmation as english } from "../en/confirmation.ts";

export const confirmation: typeof english = {
  "confirmation.unlock.title": "解锁 Ravenpass",
  "confirmation.unlock.description": "浏览器扩展正在请求打开您的保险库。",
  "confirmation.unlock.description.autofill":
    "自动填充正在请求打开您的保险库以协助登录。",
  "confirmation.change-unlock.title": "更改保险库解锁方式？",
  "confirmation.change-unlock.description": "输入当前 PIN 码以确认此更改。",
  "confirmation.change-unlock.confirm": "确认",
  "confirmation.change-unlock.error": "无法确认此更改。请重试。",
  "confirmation.decline": "拒绝",
  "confirmation.decline-error": "Ravenpass 无法拒绝此请求。请重试。",
};
