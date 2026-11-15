import type { sharing as english } from "../en/sharing.ts";

export const sharing: typeof english = {
  "sharing.prompt.title": "共享“{file}”？",
  "sharing.prompt.description":
    "输入 PIN 码，将此文件从 {identity} 发送到 {site}。",
  "sharing.prompt.share": "共享",
  "sharing.prompt.error": "Ravenpass 无法共享此文件。请重试。",
  "sharing.prompt.save-passkey.title": "为 {site} 保存通行密钥？",
  "sharing.prompt.save-passkey.description":
    "输入 PIN 码，在 Ravenpass 中为 {site} 保存通行密钥。",
  "sharing.prompt.save-passkey.confirm": "保存",
  "sharing.prompt.sign-in.title": "登录 {site}？",
  "sharing.prompt.sign-in.description":
    "输入 PIN 码，用通行密钥以 {account} 的身份登录 {site}。",
  "sharing.prompt.sign-in.description.no-account":
    "输入 PIN 码，用通行密钥登录 {site}。",
  "sharing.prompt.sign-in.confirm": "登录",
  "sharing.prompt.fill.title": "在 {site} 上填充登录信息？",
  "sharing.prompt.fill.description":
    "输入 PIN 码，允许浏览器扩展在 {site} 上填充 {account}。",
  "sharing.prompt.fill.confirm": "填充",
  "sharing.prompt.fill-card.title": "在 {site} 上填充“{file}”？",
  "sharing.prompt.fill-card.description":
    "输入 PIN 码，允许浏览器扩展在 {site} 上填充这张卡片。",
  "sharing.prompt.passkey.error": "Ravenpass 无法确认此请求。请重试。",
};
