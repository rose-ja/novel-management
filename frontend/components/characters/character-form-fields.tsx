"use client";

import { Plus, X } from "lucide-react";
import { useEffect, useState } from "react";

import { FormField } from "@/components/ui/form-field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import type { CharacterNameInput, CharacterNameKind, CharacterSystemRefInput } from "@/lib/api/characters";
import {
  listCategories,
  listNodes,
  type CategoryType,
  type SystemNode,
} from "@/lib/api/system";
import { messages } from "@/lib/messages";

export type CharacterFormValues = {
  name: string;
  names: CharacterNameInput[];
  description: string;
  /** categoryID → nodeID；空字符串表示未选择。 */
  systemRefs: Record<string, string>;
};

export type CharacterCategoryField = {
  id: string;
  name: string;
  categoryType: CategoryType;
  isBuiltin: boolean;
  options: { value: string; label: string }[];
};

export type SystemFieldsStatus = "idle" | "loading" | "ready" | "error";

export function createCharacterFormValues(): CharacterFormValues {
  return { name: "", names: [], description: "", systemRefs: {} };
}

/** 把节点树压平为下拉选项；子节点显示「父 · 子」。 */
export function flattenNodeOptions(
  nodes: SystemNode[],
  parentName?: string,
): { value: string; label: string }[] {
  return nodes.flatMap((node) => {
    const label = parentName ? `${parentName} · ${node.name}` : node.name;
    const children = node.children ? flattenNodeOptions(node.children, node.name) : [];
    return [{ value: node.id, label }, ...children];
  });
}

/** 读取启用的体系类别及其节点，供人物表单与境界筛选复用。 */
export function useCharacterSystemFields(workID: string | null, enabled: boolean) {
  const [categories, setCategories] = useState<CharacterCategoryField[]>([]);
  const [status, setStatus] = useState<SystemFieldsStatus>("idle");

  useEffect(() => {
    if (!enabled || !workID) {
      setCategories([]);
      setStatus("idle");
      return;
    }
    let active = true;
    setStatus("loading");
    (async () => {
      try {
        const items = await listCategories(workID, false);
        const fields = await Promise.all(
          items.map(async (category) => {
            const nodes = await listNodes(category.id);
            return {
              id: category.id,
              name: category.name,
              categoryType: category.category_type,
              isBuiltin: category.is_builtin,
              options: flattenNodeOptions(nodes),
            };
          }),
        );
        if (!active) return;
        setCategories(fields);
        setStatus("ready");
      } catch {
        if (!active) return;
        setCategories([]);
        setStatus("error");
      }
    })();
    return () => {
      active = false;
    };
  }, [workID, enabled]);

  return { categories, status };
}

/** 提交时丢弃空白别名/称号；调用方需先做必填校验。 */
export function toCharacterNameInputs(values: CharacterFormValues): CharacterNameInput[] {
  return values.names
    .map((name) => ({ kind: name.kind, value: name.value.trim() }))
    .filter((name) => name.value !== "");
}

/** 提交时只保留已选择的体系引用。 */
export function toCharacterSystemRefs(values: CharacterFormValues): CharacterSystemRefInput[] {
  return Object.entries(values.systemRefs)
    .filter(([, nodeID]) => nodeID !== "")
    .map(([categoryID, nodeID]) => ({ category_id: categoryID, node_id: nodeID }));
}

export type CharacterFormFieldsProps = {
  values: CharacterFormValues;
  onChange: (next: CharacterFormValues) => void;
  disabled?: boolean;
  categories: CharacterCategoryField[];
  systemStatus: SystemFieldsStatus;
};

/** 创建与编辑共用的人物字段；体系字段按当前作品的启用类别动态渲染。 */
export function CharacterFormFields({
  values,
  onChange,
  disabled = false,
  categories,
  systemStatus,
}: CharacterFormFieldsProps) {
  function updateName(index: number, patch: Partial<CharacterNameInput>) {
    onChange({
      ...values,
      names: values.names.map((name, current) => (current === index ? { ...name, ...patch } : name)),
    });
  }

  function removeName(index: number) {
    onChange({ ...values, names: values.names.filter((_, current) => current !== index) });
  }

  function addName() {
    onChange({ ...values, names: [...values.names, { kind: "alias", value: "" }] });
  }

  return (
    <>
      <FormField id="character-name" label={messages.characterNameLabel} required hint={messages.characterNameHint}>
        <Input
          id="character-name"
          value={values.name}
          maxLength={200}
          disabled={disabled}
          onChange={(event) => onChange({ ...values, name: event.target.value })}
        />
      </FormField>

      <div className="ui-field">
        <span className="ui-field__label">{messages.characterNamesLabel}</span>
        {values.names.length > 0 ? (
          <div className="character-names-editor">
            {values.names.map((name, index) => (
              <div className="character-names-row" key={index}>
                <Select
                  aria-label={`${messages.characterNamesLabel} ${index + 1}`}
                  value={name.kind}
                  disabled={disabled}
                  options={[
                    { value: "alias", label: messages.characterNameKindAlias },
                    { value: "title", label: messages.characterNameKindTitle },
                  ]}
                  onChange={(event) =>
                    updateName(index, { kind: event.target.value as CharacterNameKind })
                  }
                />
                <Input
                  aria-label={`${messages.characterNamesLabel} ${index + 1}`}
                  value={name.value}
                  maxLength={100}
                  placeholder={messages.characterNameValuePlaceholder}
                  disabled={disabled}
                  onChange={(event) => updateName(index, { value: event.target.value })}
                />
                <button
                  type="button"
                  className="ui-icon-button ui-icon-button--sm"
                  aria-label={messages.characterNameRemove(index + 1)}
                  disabled={disabled}
                  onClick={() => removeName(index)}
                >
                  <X aria-hidden="true" size={16} />
                </button>
              </div>
            ))}
          </div>
        ) : null}
        <div className="character-names-actions">
          <button
            type="button"
            className="ui-button ui-button--secondary ui-button--sm"
            disabled={disabled || values.names.length >= 100}
            onClick={addName}
          >
            <Plus aria-hidden="true" size={14} />
            <span>{messages.characterNameAdd}</span>
          </button>
          <span className="ui-field__hint">{messages.characterNamesHint(values.names.length)}</span>
        </div>
        {values.names.length > 20 ? (
          <p className="character-form-note">{messages.characterNamesMany}</p>
        ) : null}
      </div>

      <FormField
        id="character-description"
        label={messages.characterDescriptionLabel}
        hint={messages.characterDescriptionHint}
      >
        <textarea
          id="character-description"
          className="ui-textarea"
          value={values.description}
          maxLength={20000}
          rows={5}
          disabled={disabled}
          onChange={(event) => onChange({ ...values, description: event.target.value })}
        />
      </FormField>
      <p className="character-description-count">
        {messages.characterDescriptionCount(values.description.length)}
      </p>

      <section className="character-form-section" aria-labelledby="character-system-title">
        <h3 id="character-system-title" className="character-form-section__title">
          {messages.characterSystemSection}
        </h3>
        {systemStatus === "loading" ? (
          <Skeleton variant="lines" lines={2} label={messages.characterSystemLoading} />
        ) : null}
        {systemStatus === "error" ? (
          <p className="character-form-note">{messages.characterSystemLoadFailed}</p>
        ) : null}
        {systemStatus === "ready" && categories.length === 0 ? (
          <p className="character-form-note">{messages.characterSystemEmpty}</p>
        ) : null}
        {categories.length > 0 ? (
          <div className="character-system-fields">
            {categories.map((category) => (
              <FormField
                key={category.id}
                id={`character-ref-${category.id}`}
                label={category.name}
              >
                <Select
                  id={`character-ref-${category.id}`}
                  value={values.systemRefs[category.id] ?? ""}
                  placeholder={messages.characterSystemPlaceholder}
                  disabled={disabled}
                  options={category.options}
                  onChange={(event) =>
                    onChange({
                      ...values,
                      systemRefs: { ...values.systemRefs, [category.id]: event.target.value },
                    })
                  }
                />
              </FormField>
            ))}
          </div>
        ) : null}
      </section>
    </>
  );
}
