/** 回收站根类型展示名；未知类型回退为「未知类型」，避免列表崩溃。 */
function trashTypeLabel(rootType: string): string {
  switch (rootType) {
    case "work":
      return "作品";
    case "system_category":
      return "体系类别";
    case "system_node":
      return "体系节点";
    case "character":
      return "人物";
    default:
      return "未知类型";
  }
}

/** 作品名称沿用书名号；体系类别/节点与未知类型直接显示名称。 */
function trashDisplayName(rootType: string, name: string): string {
  return rootType === "work" ? `《${name}》` : name;
}

/** 操作按钮与标题中的对象描述：作品用书名号，其余用「」。 */
function trashEntityLabel(rootType: string, name: string): string {
  return rootType === "work" ? `《${name}》` : `「${name}」`;
}

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

  cancel: "取消",
  close: "关闭",
  confirm: "确认",

  /** 稳定错误码到用户文案的映射；未覆盖时回退到后端文案。 */
  errorByCode: {
    NAME_CONFLICT: "名称已存在，请换一个名称",
    VERSION_CONFLICT: "内容已被修改，请刷新后重试",
    VALIDATION_ERROR: "输入不合法，请检查后重试",
    SNAPSHOT_FAILED: "快照创建失败，操作已取消",
    UNDO_EXPIRED: "撤销窗口已关闭",
    ALREADY_DELETED: "记录已不在回收站",
    RESTORE_CONFLICT: "恢复目标与现有记录重名",
    CATEGORY_LIMIT: "最多 10 类（含内置境界）",
    CATEGORY_BUILTIN_IMMUTABLE: "内置类别不可修改",
    CATEGORY_DISABLED: "该类别已停用",
    NODE_LEVEL_LIMIT: "体系节点最多两级",
    NODE_PARENT_MISMATCH: "上级节点与类别不匹配",
    CROSS_WORK_REFERENCE: "只能引用同一作品下的体系节点",
    PRECONDITION_REQUIRED: "请求缺少版本信息，请刷新后重试",
    HAS_CHILDREN: "仍有下级节点，无法删除",
    PARENT_DELETED: "上级或类别仍在回收站",
    REFERENCE_CONFLICT: "仍被其他记录引用，无法删除",
    NOT_FOUND: "资源不存在",
    FORBIDDEN_ORIGIN: "请求来源校验失败，请从本机页面操作",
    INTERNAL_ERROR: "服务内部错误，请稍后重试",
  } as Record<string, string>,

  workSwitcherAria: "切换当前作品",
  workLoading: "正在读取作品…",
  workNone: "未选择作品",
  workMenuTitle: "作品",
  workListFailed: "作品列表加载失败",
  workListRetry: "重新加载",
  workEmptyMenu: "还没有作品，先创建一本开始。",
  workCreate: "新建作品",
  workEdit: "编辑作品",
  workDelete: "删除作品",
  workCreateTitle: "新建作品",
  workCreateDescription: "作品用于隔离每本小说的世界观、体系与榜单。",
  workNameLabel: "作品名称",
  workNameHint: "必填，1–200 字符；名称全局唯一，英文不区分大小写。",
  workDescriptionLabel: "简介",
  workDescriptionHint: "可选，最多 20,000 字符。",
  workEditTitle: (name: string) => `编辑作品《${name}》`,
  workDeleteTitle: (name: string) => `删除作品《${name}》？`,
  workDeleteDescription: "将整本作品及其全部内容移入回收站；删除前会自动创建快照。",
  workDeleteConfirmLabel: "输入作品名称以确认",
  workDeleteConfirmHint: "需与作品名称完全一致。",
  workCreateSubmit: "创建作品",
  workSaveSubmit: "保存修改",
  workDeleteSubmit: "删除作品",
  workCreated: (name: string) => `已创建《${name}》`,
  workUpdated: (name: string) => `已保存《${name}》`,
  workSwitched: (name: string) => `已切换到《${name}》`,
  workMovedToTrash: (name: string) => `《${name}》已移入回收站`,
  workUndone: (name: string) => `已撤销删除《${name}》`,
  undo: "撤销",
  unsavedTitle: "放弃未保存的修改？",
  unsavedDescription: "离开后本次修改不会保留。",
  unsavedKeep: "继续编辑",
  unsavedDiscard: "放弃并离开",

  trashNav: "回收站",
  trashTitle: "回收站",
  trashDescription: (total: number) => `跨作品 · 共 ${total} 项`,
  trashSnapshotNotice: "永久删除与清空回收站前会自动创建快照，保留最近 3 份。",
  trashClear: "清空回收站",
  trashClearTitle: "清空回收站？",
  trashClearDescription: (total: number) =>
    `将永久删除全部 ${total} 项，包括整本作品与整组类别。此操作不可撤销；删除前会自动创建快照。`,
  trashCleared: "回收站已清空",
  trashTypeAll: "全部类型",
  trashTypeWork: "作品",
  trashTypeSystemCategory: "体系类别",
  trashTypeSystemNode: "体系节点",
  trashTypeCharacter: "人物",
  trashTypeUnknown: "未知类型",
  trashTypeLabel,
  trashDisplayName,
  trashEntityLabel,
  trashSearchPlaceholder: "搜索名称…",
  trashColumnType: "类型",
  trashColumnName: "名称",
  trashColumnParent: "父级 / 所属",
  trashColumnDeletedAt: "删除时间",
  trashColumnAffected: "影响条数",
  trashColumnActions: "操作",
  trashAffected: (rootType: string, count: number) => {
    switch (rootType) {
      case "work":
        return `共 ${count} 项（类别与节点）`;
      case "system_category":
        return `共 ${count} 个节点`;
      case "system_node":
        return `共 ${count > 0 ? count : 1} 项`;
      case "character":
        return `共 ${count > 0 ? count : 1} 项`;
      default:
        return `共 ${count} 项`;
    }
  },
  trashRestore: "恢复",
  trashRestoreTitle: (rootType: string, name: string) => {
    switch (rootType) {
      case "work":
        return `恢复作品《${name}》？`;
      case "system_category":
        return `整组恢复类别「${name}」？`;
      case "system_node":
        return `恢复节点「${name}」？`;
      case "character":
        return `恢复人物「${name}」？`;
      default:
        return `恢复「${name}」？`;
    }
  },
  trashRestoreDescription: (rootType: string) => {
    switch (rootType) {
      case "work":
        return "恢复作品及其类别与节点；如与现有记录重名，可改名后恢复。";
      case "system_category":
        return "整组恢复类别及其节点；如与现有记录重名，可改名后恢复。";
      case "system_node":
        return "恢复节点；如与现有记录重名，可改名后恢复。";
      case "character":
        return "恢复人物及其别名与体系引用；如与现有记录重名，可改名后恢复。";
      default:
        return "将整组恢复到原位置；如与现有记录重名，可改名后恢复。";
    }
  },
  trashRestoreAria: (rootType: string, name: string) =>
    `恢复${trashTypeLabel(rootType)}${trashEntityLabel(rootType, name)}`,
  trashRestoreSubmit: "恢复",
  trashRestored: (rootType: string, name: string) =>
    rootType === "work"
      ? `已恢复《${name}》`
      : `已恢复${trashTypeLabel(rootType)}${trashEntityLabel(rootType, name)}`,
  trashRestoreConflictTitle: "恢复目标与现有记录重名",
  trashRestoreConflictDescription: "同一位置已存在同名记录。请输入恢复后的名称，原记录不会被覆盖。",
  trashRestoreNewNameLabel: "恢复后名称",
  trashPurge: "永久删除",
  trashPurgeTitle: (rootType: string, name: string) => {
    switch (rootType) {
      case "work":
        return `永久删除作品《${name}》？`;
      case "system_category":
        return `永久删除体系类别「${name}」？`;
      case "system_node":
        return `永久删除体系节点「${name}」？`;
      case "character":
        return `永久删除人物「${name}」？`;
      default:
        return `永久删除「${name}」？`;
    }
  },
  trashPurgeDescription: (rootType: string) => {
    switch (rootType) {
      case "system_category":
        return "该类别及其下节点将永久删除，不可恢复。删除前会自动创建快照。";
      case "system_node":
        return "节点将永久删除，不可恢复。删除前会自动创建快照。";
      case "character":
        return "人物及别名、体系引用将永久删除，不可恢复。删除前会自动创建快照。";
      default:
        return "此操作不可撤销。删除前会自动创建快照。";
    }
  },
  trashPurgeAria: (rootType: string, name: string) =>
    `永久删除${trashTypeLabel(rootType)}${trashEntityLabel(rootType, name)}`,
  trashPurgeSubmit: "永久删除",
  trashPurged: (rootType: string, name: string) =>
    rootType === "work"
      ? `已永久删除《${name}》`
      : `已永久删除${trashTypeLabel(rootType)}${trashEntityLabel(rootType, name)}`,
  trashEmptyTitle: "回收站是空的",
  trashEmptyDescription: "删除的内容会在这里保留，直到你恢复或永久删除。",
  trashEmptyAction: "返回概览",
  trashFilterEmptyTitle: "没有符合筛选的回收站项",
  trashFilterEmptyDescription: "调整或清除筛选条件后再试。",
  trashClearFilter: "清除筛选",
  trashSnapshotFailed: "快照创建失败，未执行删除",
  trashTotal: (total: number) => `共 ${total} 项`,
  paginationPrev: "上一页",
  paginationNext: "下一页",
  paginationPage: (page: number, totalPages: number) => `第 ${page} / ${totalPages} 页`,

  systemCategory: "类别",
  systemNode: "节点",
  systemBuiltin: "内置",
  systemRequired: "必开",
  systemDisabled: "停用",
  systemEra: "纪元",
  systemReviewPending: "待校对",
  systemReviewConfirmed: "已校对",

  systemTitle: "体系",
  systemWorkFallback: "当前作品",
  systemDescription: (workName: string) => `《${workName}》 · 类别与层级`,
  systemAddCategory: "新增类别",
  systemCategoryListAria: "类别列表",
  systemCategoryEmptyTitle: "还没有类别",
  systemCategoryEmptyDescription: "内置「境界」应已自动创建；如未显示，请重试加载。",
  systemNormal: "普通",
  systemStateEnabled: "启用",
  systemStateDisabled: "已停用",
  systemNodeCount: (count: number) => `${count} 节点`,
  systemReferenceCount: (count: number) => `被引用 ${count} 人`,
  systemReferenceLabel: "被引用",
  systemBuiltinDescription: "人物主境界与子层级；名称固定，不可停用或删除。",
  systemNormalDescription: "人物属性分层；节点最多两级，同级可手动排序。",
  systemEraDescription: "纪元类别只按手动顺序排列，不做日期计算。",
  systemCategoryDisabledNotice: "该类别已停用，人物表单不再显示该字段；已有数据保留。",
  systemPendingOnly: "仅看待校对",
  systemNodeSectionTitle: "节点树",
  systemAddPrimaryNode: "新增主层级",
  systemAddPrimaryNodeDisabled: "该类别已停用，无法新增节点",
  systemNodeEmptyTitle: "还没有节点",
  systemNodeEmptyDescription: "「新增主层级」后，这里会显示类别的两级节点。",
  systemNodeEmptyReadOnlyDescription: "该类别已停用；启用后可新增主层级。",
  systemNodePendingEmptyTitle: "没有待校对节点",
  systemNodePendingEmptyDescription: "当前类别没有待校对节点；取消勾选可查看全部节点。",
  systemNodeTreeAria: "节点树",
  systemNodeLoading: "正在加载节点",
  systemNodeReadOnlyNotice: "该类别已停用，节点仅供查看；启用后可新增、改名、删除或调整顺序。",
  systemNodeReordered: "已调整顺序",
  systemNodeReorderFailed: "顺序保存失败，已恢复原顺序",
  systemExpandNode: (name: string) => `展开节点：${name}`,
  systemCollapseNode: (name: string) => `收起节点：${name}`,
  systemMoveNodeUp: (name: string) => `上移节点：${name}`,
  systemMoveNodeDown: (name: string) => `下移节点：${name}`,
  systemMoveNodeUpAtStart: (name: string) => `上移节点：${name}（已在首位）`,
  systemMoveNodeDownAtEnd: (name: string) => `下移节点：${name}（已在末位）`,
  systemMoveNodeUpFiltered: (name: string) => `上移节点：${name}（待校对筛选中不可调整顺序）`,
  systemMoveNodeDownFiltered: (name: string) => `下移节点：${name}（待校对筛选中不可调整顺序）`,
  systemAddChildNodeAria: (name: string) => `新增子层级：${name}`,
  systemRenameNodeAria: (name: string) => `改名节点：${name}`,
  systemDeleteNodeAria: (name: string) => `删除节点：${name}`,
  systemLoadingCategories: "正在加载类别",
  systemLoadingDetail: "正在加载类别详情",

  systemAddCategoryTitle: "新增类别",
  systemAddCategoryDescription: "名称在同一作品内不可重复，英文不区分大小写；类型创建后不可更改。",
  systemCategoryNameLabel: "类别名称",
  systemCategoryNameHint: "必填，最多 200 字符；同作品内不可重名，英文不区分大小写。",
  systemCategoryNameRequired: "请输入类别名称",
  systemCategoryTypeLabel: "类型",
  systemCategoryTypeHint: "创建后不可更改。",
  systemCategoryTypeNormalOption: "普通",
  systemCategoryTypeEraOption: "纪元",
  systemCategoryCreateSubmit: "创建类别",
  systemCategoryCreated: (name: string) => `已创建类别「${name}」`,
  systemRenameCategory: "改名",
  systemRenameCategoryTitle: (name: string) => `改名类别「${name}」`,
  systemRenameCategorySubmit: "保存",
  systemCategoryRenamed: (name: string) => `已保存类别「${name}」`,
  systemDisableCategory: "停用",
  systemEnableCategory: "启用",
  systemDisableCategoryTitle: (name: string) => `停用类别「${name}」？`,
  systemDisableCategoryDescription: "停用后人物表单不再显示该字段，已有数据保留，可重新启用。",
  systemDisableCategorySubmit: "停用",
  systemCategoryDisabledToast: (name: string) => `已停用类别「${name}」`,
  systemCategoryEnabled: (name: string) => `已启用类别「${name}」`,
  systemDeleteCategory: "删除",
  systemDeleteCategoryTitle: (name: string) => `删除类别「${name}」？`,
  systemDeleteCategoryDescription: (name: string, nodeCount: number) =>
    `类别「${name}」及其下全部 ${nodeCount} 个节点会一并进入回收站，可在回收站整组恢复。`,
  systemDeleteCategorySubmit: "删除",
  systemCategoryDeleted: (name: string) => `类别「${name}」已移入回收站`,
  systemCategoryUndone: (name: string) => `已撤销删除类别「${name}」`,

  systemAddNodeTitle: "新增主层级",
  systemAddChildNodeTitle: (parentName: string) => `新增子层级 · ${parentName}`,
  systemAddNodeDescription: "节点名称在同一父级下不可重复；每个类别最多两级。",
  systemAddChildNodeDescription: (parentName: string) =>
    `父级：${parentName}；子层级不能再新增下级。`,
  systemNodeNameLabel: "节点名称",
  systemNodeNameHint: "必填，最多 200 字符；同类别同一父级下不可重名，英文不区分大小写。",
  systemNodeNameRequired: "请输入节点名称",
  systemEraStartLabel: "纪元起",
  systemEraEndLabel: "纪元止",
  systemEraTextHint: "可选，自由文本；仅用于展示，不校验格式、不参与日期计算。",
  systemAddNodeSubmit: "创建节点",
  systemNodeCreated: (name: string) => `已新增节点「${name}」`,
  systemRenameNodeTitle: (name: string) => `改名节点「${name}」`,
  systemRenameNodeSubmit: "保存",
  systemNodeRenamed: (name: string) => `已保存节点「${name}」`,
  systemDeleteNodeTitle: (name: string) => `删除节点「${name}」？`,
  systemDeleteNodeDescription: "节点将进入回收站，可在 8 秒内撤销。",
  systemDeleteNodeSubmit: "删除",
  systemNodeDeleted: (name: string) => `节点「${name}」已移入回收站`,
  systemNodeUndone: (name: string) => `已撤销删除节点「${name}」`,
  systemNodeHasChildrenTitle: "仍有下级节点，无法删除",
  systemNodeHasChildrenDescription: "请先删除以下子层级后再试：",
  systemNodeHasChildrenFallback: "该节点仍有下级节点，请先删除子层级。",
  systemVersionRefreshed: "数据已变化，已为你刷新",
  systemReferenceNodeDeleteTitle: (name: string, count: number) =>
    `无法删除「${name}」：仍有 ${count} 人引用该节点`,
  systemReferenceDisableTitle: (name: string, count: number) =>
    `无法停用「${name}」：仍有 ${count} 人引用该类别`,
  systemReferenceDeleteTitle: (name: string, count: number) =>
    `无法删除「${name}」：仍有 ${count} 人引用该类别`,
  systemReferenceDescription: "请先在人物中解除引用，再重试。",
  systemReferencePeople: (count: number) => `${count} 人`,
  systemReferenceUnnamed: "未命名人物",
  systemReferenceUnnamedNode: "未命名节点",
  systemReferenceInTrash: "回收站中",
  systemReferenceClose: "知道了",

  snapshotCardTitle: "自动快照",
  snapshotCardDescription:
    "删除作品、永久删除与清空回收站前会自动创建快照，保留最近 3 份；恢复请使用命令行。",
  snapshotColumnTime: "时间",
  snapshotColumnReason: "原因",
  snapshotColumnSize: "数据库大小",
  snapshotColumnStatus: "状态",
  snapshotEmpty: "暂无快照",
  snapshotEmptyDescription: "执行删除作品、永久删除或清空回收站后，这里会出现自动快照。",
  snapshotStatusOK: "成功",
  snapshotStatusFailed: "失败",
  snapshotReasons: {
    work_delete: "删除作品",
    trash_purge: "永久删除",
    purge_all: "清空回收站",
    import: "导入",
    cli_restore: "命令行恢复",
  } as Record<string, string>,

  characterTitle: "人物",
  characterWorkFallback: "当前作品",
  characterDescription: (workName: string) => `《${workName}》 · 人物档案`,
  characterAdd: "新建人物",
  characterSearchPlaceholder: "搜索姓名、别名或称号…",
  characterSearchLabel: "搜索",
  characterFilterRealm: "境界",
  characterFilterRealmAll: "全部境界",
  characterFilterPending: "仅看待校对",
  characterSortLabel: "排序",
  characterSortUpdatedDesc: "最近更新",
  characterSortUpdatedAsc: "最早更新",
  characterSortNameAsc: "姓名 A→Z",
  characterSortNameDesc: "姓名 Z→A",
  characterPageSizeLabel: "每页",
  characterColumnName: "姓名",
  characterColumnNames: "别名 / 称号",
  characterColumnRealm: "境界",
  characterColumnFaction: "势力",
  characterColumnRanking: "榜单",
  characterColumnReview: "校对",
  characterColumnUpdatedAt: "更新时间",
  characterColumnActions: "操作",
  characterNoValue: "—",
  characterNameOverflow: (count: number) => `+${count}`,
  characterReviewPending: "待校对",
  characterReviewConfirmed: "已校对",
  characterEdit: "编辑",
  characterLoading: "正在加载人物",
  characterTableCaption: "人物列表",
  characterEmptyTitle: "还没有人物",
  characterEmptyDescription: "新建第一个人物后，可在这里搜索、筛选并维护档案。",
  characterFilterEmptyTitle: "没有符合筛选的人物",
  characterFilterEmptyDescription: "调整或清除筛选条件后再试。",
  characterClearFilter: "清除筛选",

  characterCreateTitle: "新建人物",
  characterCreateDescription: "人物归属于当前作品；同名（英文不区分大小写）会被拒绝。",
  characterNameLabel: "姓名",
  characterNameHint: "必填，1–200 字符；同一作品内不可重名，英文不区分大小写。",
  characterNameRequired: "请输入人物姓名",
  characterNamesLabel: "别名 / 称号",
  characterNamesHint: (count: number) => `可选，最多 100 条；当前 ${count} 条。`,
  characterNamesMany: "超过 20 条后请确认是否都必要",
  characterNameValueRequired: "别名或称号不能为空",
  characterNameAdd: "添加别名 / 称号",
  characterNameKindAlias: "别名",
  characterNameKindTitle: "称号",
  characterNameValuePlaceholder: "输入内容",
  characterNameKindAria: (index: number) => `第 ${index} 条类型`,
  characterNameValueAria: (index: number) => `第 ${index} 条内容`,
  characterNameRemove: (index: number) => `移除第 ${index} 条别名或称号`,
  characterDescriptionLabel: "简介",
  characterDescriptionHint: "可选，最多 20,000 字符。",
  characterDescriptionCount: (count: number) => `${count} / 20000`,
  characterSystemSection: "体系引用",
  characterSystemLoading: "正在加载体系字段",
  characterSystemLoadFailed: "体系字段加载失败；可先保存其他信息。",
  characterSystemEmpty: "当前作品没有启用的体系类别。",
  characterSystemPlaceholder: "未选择",
  characterCreateSubmit: "创建人物",
  characterCreated: (name: string) => `已创建人物「${name}」`,

  characterDetailDescription: "查看并维护人物基本信息与体系引用。",
  characterNotFoundTitle: "人物不存在或已删除",
  characterNotFoundDescription: "该人物可能已被删除，或不属于当前作品。",
  characterBackToList: "返回人物列表",
  characterEditTitle: "编辑人物",
  characterSaveSubmit: "保存修改",
  characterUpdated: (name: string) => `已保存人物「${name}」`,
  characterVersionConflict: (version: number | null) =>
    version === null
      ? "内容已被修改，请刷新最新内容后重试"
      : `内容已被修改（最新版本 ${version}），请刷新最新内容后重试`,
  characterRefresh: "刷新最新内容",
  characterDelete: "删除人物",
  characterDeleteTitle: (name: string) => `删除人物「${name}」？`,
  characterDeleteDescription:
    "人物将移入回收站，8 秒内可撤销；永久删除前仍可恢复。",
  characterDeleteSubmit: "删除人物",
  characterDeleteReferenceTitle: "人物仍被引用，无法删除",
  characterMovedToTrash: (name: string) => `人物「${name}」已移入回收站，8 秒内可撤销`,
  characterUndone: (name: string) => `已撤销删除人物「${name}」`,
  characterDeletedTitle: "人物已移入回收站",
  characterDeletedDescription: "可在 8 秒内撤销，或稍后前往回收站恢复。",
  characterGoTrash: "前往回收站",
  characterRelationsTitle: "关系",
  characterSystemRefsTitle: "体系引用",
  characterSystemRefsEmpty: "暂无体系引用",
  characterFactionTitle: "势力",
  characterFactionEmpty: "暂无势力关联",
  characterLocationTitle: "地点",
  characterLocationEmpty: "暂无地点关联",
  characterRankingTitle: "榜单",
  characterRankingEmpty: "暂无榜单记录",
  characterArchiveTitle: "档案",
  characterDescriptionEmpty: "暂无简介",
  characterCreatedAt: "创建时间",
  characterUpdatedAt: "更新时间",
  characterImagePlaceholder: "暂无主图",
} as const;

export type MessageCode = keyof typeof messages;
