import { ApiError, NetworkError, requestVoid, type ApiErrorBody } from "./client";
import type { CharacterImage } from "./characters";

export const maxImageBytes = 10 * 1024 * 1024;

export const imageAccept = "image/jpeg,image/png,image/webp";

/** 客户端预校验，错误文案与后端保持一致。 */
export function validateImageFile(file: File): string | null {
  if (!["image/jpeg", "image/png", "image/webp"].includes(file.type)) {
    return "仅支持 JPG、PNG 或 WebP 图片";
  }
  if (file.size <= 0 || file.size > maxImageBytes) {
    return "图片不能超过 10MB";
  }
  return null;
}

/** 上传人物主图；XHR 用于展示上传进度。 */
export function uploadCharacterImage(
  characterID: string,
  file: File,
  onProgress?: (percent: number) => void,
): Promise<CharacterImage> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", `/api/v1/characters/${characterID}/image`);
    xhr.setRequestHeader("Accept", "application/json");
    xhr.setRequestHeader("X-Requested-With", "XMLHttpRequest");
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable && onProgress) {
        onProgress(Math.round((event.loaded / event.total) * 100));
      }
    };
    xhr.onerror = () => reject(new NetworkError());
    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          resolve(JSON.parse(xhr.responseText) as CharacterImage);
        } catch {
          reject(new ApiError({ code: "INVALID_RESPONSE", message: "图片上传响应无法解析", status: xhr.status }));
        }
        return;
      }
      let body: ApiErrorBody | null = null;
      try {
        body = JSON.parse(xhr.responseText) as ApiErrorBody;
      } catch {
        body = null;
      }
      reject(
        new ApiError({
          code: body?.error?.code ?? `HTTP_${xhr.status}`,
          message: body?.error?.message ?? "图片上传失败，请重试",
          status: xhr.status,
          requestId: body?.error?.request_id,
          details: body?.error?.details,
        }),
      );
    };
    const form = new FormData();
    form.append("file", file);
    xhr.send(form);
  });
}

export function deleteCharacterImage(characterID: string): Promise<void> {
  return requestVoid(`/api/v1/characters/${characterID}/image`, { method: "DELETE" });
}
