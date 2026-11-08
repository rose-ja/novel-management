import type { ReactNode } from "react";

export type TableColumn = {
  key: string;
  label: string;
  align?: "left" | "center" | "right";
  width?: string;
};

export type TableProps = {
  columns: TableColumn[];
  rows: ReactNode[][];
  loading?: boolean;
  empty?: ReactNode;
  caption?: string;
  rowKey?: (index: number) => string;
};

/** 数据表：统一 loading 骨架与空状态槽位，表格本身保持语义化 table 结构。 */
export function Table({ columns, rows, loading = false, empty, caption, rowKey }: TableProps) {
  const showEmpty = !loading && rows.length === 0 && empty;

  return (
    <div className="ui-table-wrap">
      <table className="ui-table">
        {caption ? <caption className="ui-table__caption">{caption}</caption> : null}
        <thead>
          <tr>
            {columns.map((column) => (
              <th key={column.key} scope="col" style={{ width: column.width, textAlign: column.align ?? "left" }}>
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {loading
            ? Array.from({ length: 3 }, (_, index) => (
                <tr key={`skeleton-${index}`} className="ui-table__skeleton-row">
                  {columns.map((column) => (
                    <td key={column.key}>
                      <span className="ui-skeleton ui-skeleton--text" />
                    </td>
                  ))}
                </tr>
              ))
            : rows.map((row, index) => (
                <tr key={rowKey ? rowKey(index) : index}>
                  {row.map((cell, cellIndex) => (
                    <td key={columns[cellIndex]?.key ?? cellIndex} style={{ textAlign: columns[cellIndex]?.align ?? "left" }}>
                      {cell}
                    </td>
                  ))}
                </tr>
              ))}
        </tbody>
      </table>
      {showEmpty ? <div className="ui-table__empty">{empty}</div> : null}
    </div>
  );
}
