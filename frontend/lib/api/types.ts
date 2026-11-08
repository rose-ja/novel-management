export type AppSettings = {
  current_work_id: string | null;
  last_export_at: string | null;
  last_restore_at: string | null;
  data_dir: string;
  snapshot_keep: number;
  app_version: string;
};

export const EMPTY_SETTINGS: AppSettings = {
  current_work_id: null,
  last_export_at: null,
  last_restore_at: null,
  data_dir: "",
  snapshot_keep: 3,
  app_version: "",
};
