/** 全局文案：页面与组件只引用这里，避免同一语义出现多种说法。 */
export const messages = {
  loadFailed: "加载失败，请重试",
  saveFailed: "保存失败，请稍后重试",
  networkError: "网络异常，请检查连接",
  retry: "重试",
  backHome: "返回首页",
  notFoundTitle: "页面不存在",
  notFoundDescription: "该地址可能已被移动或从未存在。",
  noWorkTitle: "还没有作品",
  noWorkDescription:
    "作品用于隔离每本小说的世界观、体系与榜单。创建作品后，人物、势力、地点与榜单都会归入当前作品。",
  noWorkBadge: "未选择作品",
} as const;
