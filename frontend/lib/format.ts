/** 时间与字节的统一展示格式，避免各页面自行拼装。 */
function pad(value: number): string {
  return String(value).padStart(2, "0");
}

/** `YYYY-MM-DD HH:mm`；无效或缺失显示「—」。 */
export function formatDateTime(value: string | null | undefined): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** 长路径中间省略，保留头尾便于识别文件；短路径原样返回。 */
export function truncateMiddle(value: string, maxLength = 48): string {
  if (value.length <= maxLength) return value;
  const keep = maxLength - 1;
  const head = Math.ceil(keep / 2);
  const tail = Math.floor(keep / 2);
  return `${value.slice(0, head)}…${value.slice(value.length - tail)}`;
}

/** 人类可读的字节数；无有效数据时显示「—」。 */
export function formatBytes(bytes: number | null | undefined): string {
  if (bytes === null || bytes === undefined || !Number.isFinite(bytes) || bytes <= 0) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  const rounded = value >= 100 || index === 0 ? Math.round(value) : Math.round(value * 10) / 10;
  return `${rounded} ${units[index]}`;
}
