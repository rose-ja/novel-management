"use client";

import { EmptyState } from "@/components/ui/empty-state";
import { Table, type TableColumn } from "@/components/ui/table";
import type { SeatView } from "@/lib/api/rankings";
import { messages } from "@/lib/messages";

import { SeatActionsCell, SeatCharacterCell, SeatNoteCell, SeatNumberCell } from "./seat-row";

const columns: TableColumn[] = [
  { key: "seat_no", label: messages.rankingSeatNo, width: "72px" },
  { key: "character", label: messages.rankingSeatOccupied },
  { key: "note", label: messages.rankingSeatNote },
  { key: "actions", label: messages.rankingSeatActions, align: "right", width: "180px" },
];

export type SeatTableProps = {
  rankingID: string;
  seats: SeatView[];
  loading?: boolean;
  highlightSeat?: number | null;
  onAssign: (seat: SeatView) => void;
  onAdjust: (seat: SeatView) => void;
  onRemove: (seat: SeatView) => void;
  onNoteSaved: (updated: SeatView) => void;
};

/** 固定席位表格；空席保留可见并可直接指派。 */
export function SeatTable({
  rankingID,
  seats,
  loading = false,
  highlightSeat,
  onAssign,
  onAdjust,
  onRemove,
  onNoteSaved,
}: SeatTableProps) {
  const rows = seats.map((seat) => [
    <SeatNumberCell key="seat" seatNo={seat.seat_no} highlighted={highlightSeat === seat.seat_no} />,
    <SeatCharacterCell key="character" seat={seat} />,
    <SeatNoteCell key="note" rankingID={rankingID} seat={seat} onSaved={onNoteSaved} />,
    <SeatActionsCell key="actions" seat={seat} onAssign={onAssign} onAdjust={onAdjust} onRemove={onRemove} />,
  ]);

  return (
    <Table
      columns={columns}
      rows={rows}
      loading={loading}
      skeletonRows={8}
      empty={<EmptyState variant="inline" title={messages.rankingSeatsTitle} description={messages.rankingNoValue} />}
      rowKey={(index) => String(seats[index]?.seat_no ?? index)}
      caption={messages.rankingSeatsTitle}
    />
  );
}
