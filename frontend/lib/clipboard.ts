/** 复制文本到剪贴板；浏览器不支持时返回 false，由调用方提示手动选择。 */
export async function copyText(value: string): Promise<boolean> {
  if (!navigator.clipboard?.writeText) return false;
  try {
    await navigator.clipboard.writeText(value);
    return true;
  } catch {
    return false;
  }
}
