"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { useCallback, useEffect, useState } from "react";

import { AdjustSeatDialog } from "@/components/rankings/adjust-seat-dialog";
import { AssignSeatDialog } from "@/components/rankings/assign-seat-dialog";
import { CapacityDialog } from "@/components/rankings/capacity-dialog";
import { DeleteRankingDialog } from "@/components/rankings/delete-ranking-dialog";
import { JumpToSeat } from "@/components/rankings/jump-to-seat";
import { RemoveSeatDialog } from "@/components/rankings/remove-seat-dialog";
import { SeatTable } from "@/components/rankings/seat-table";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { Pagination } from "@/components/ui/pagination";
import { Select } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { ApiError } from "@/lib/api/client";
import { toUserMessage } from "@/lib/api/errors";
import {
  getRanking,
  listSeats,
  putSeat,
  updateRanking,
  type DeleteRankingResult,
  type RankingDetail,
  type SeatPage,
  type SeatView,
} from "@/lib/api/rankings";
import { undoTrash } from "@/lib/api/trash";
import { messages } from "@/lib/messages";

const SEAT_PAGE_SIZE = 50;

type SeatFilter = "all" | "occupied" | "empty";

const seatFilterOptions: { value: SeatFilter; label: string }[] = [
  { value: "all", label: messages.rankingAllSeats },
  { value: "occupied", label: messages.rankingOnlyOccupied },
  { value: "empty", label: messages.rankingOnlyEmpty },
];

type LoadStatus = "loading" | "ready" | "error" | "not_found";

export function RankingDetailPage({ rankingID }: { rankingID: string }) {
  const router = useRouter();
  const { showToast } = useToast();
  const [detail, setDetail] = useState<RankingDetail | null>(null);
  const [seats, setSeats] = useState<SeatPage | null>(null);
  const [status, setStatus] = useState<LoadStatus>("loading");
  const [error, setError] = useState<Error | null>(null);
  const [reloadTick, setReloadTick] = useState(0);
  const [page, setPage] = useState(1);
  const [seatFilter, setSeatFilter] = useState<SeatFilter>("all");
  const [highlight, setHighlight] = useState<number | null>(null);
  const [nameValue, setNameValue] = useState("");
  const [nameSaving, setNameSaving] = useState(false);
  const [conflict, setConflict] = useState(false);
  const [assignSeat, setAssignSeat] = useState<SeatView | null>(null);
  const [adjustSeat, setAdjustSeat] = useState<SeatView | null>(null);
  const [removeSeat, setRemoveSeat] = useState<SeatView | null>(null);
  const [capacityOpen, setCapacityOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  const reload = useCallback(() => setReloadTick((value) => value + 1), []);

  useEffect(() => {
    let active = true;
    setStatus("loading");
    setError(null);
    setConflict(false);
    getRanking(rankingID, page, SEAT_PAGE_SIZE)
      .then(async (loaded) => {
        if (!active) return;
        let seatPage = loaded.seats;
        if (seatFilter !== "all") {
          seatPage = await listSeats(rankingID, {
            page,
            pageSize: SEAT_PAGE_SIZE,
            occupied: seatFilter === "occupied",
          });
        }
        if (!active) return;
        setDetail(loaded);
        setSeats(seatPage);
        setNameValue((current) => (current === "" ? loaded.name : current));
        setStatus("ready");
      })
      .catch((cause: unknown) => {
        if (!active) return;
        if (cause instanceof ApiError && cause.code === "NOT_FOUND") {
          setStatus("not_found");
          return;
        }
        setError(cause instanceof Error ? cause : new Error(messages.loadFailed));
        setStatus("error");
      });
    return () => {
      active = false;
    };
  }, [rankingID, page, seatFilter, reloadTick]);

  useEffect(() => {
    if (detail) setNameValue(detail.name);
  }, [detail]);

  useEffect(() => {
    if (!highlight || !seats) return;
    const node = document.querySelector(`[data-seat-no="${highlight}"]`);
    node?.scrollIntoView({ block: "center" });
    const timer = window.setTimeout(() => setHighlight(null), 2000);
    return () => window.clearTimeout(timer);
  }, [highlight, seats]);

  function jumpToSeat(seatNo: number, targetPage: number) {
    setSeatFilter("all");
    setPage(targetPage);
    setHighlight(seatNo);
  }

  async function saveName() {
    if (!detail || nameSaving) return;
    const trimmed = nameValue.trim();
    if (!trimmed || trimmed === detail.name) {
      setNameValue(detail.name);
      return;
    }
    setNameSaving(true);
    setConflict(false);
    try {
      const updated = await updateRanking(rankingID, { name: trimmed, version: detail.version });
      setDetail(updated);
      showToast({ tone: "success", message: messages.rankingUpdated(updated.name) });
      reload();
    } catch (cause) {
      if (cause instanceof ApiError && cause.code === "VERSION_CONFLICT") {
        setConflict(true);
        return;
      }
      showToast({ tone: "error", message: toUserMessage(cause, messages.saveFailed) });
    } finally {
      setNameSaving(false);
    }
  }

  function handleRemoved(removed: SeatView, _updated: SeatView) {
    reload();
    const removedName = removed.character?.name ?? "";
    showToast({
      tone: "undo",
      message: messages.rankingRemoved(removedName),
      actionLabel: messages.undo,
      onAction: () => {
        getRanking(rankingID, 1, SEAT_PAGE_SIZE)
          .then((loaded) => {
            const current = loaded.seats.items.find((item) => item.seat_no === removed.seat_no);
            if (!current) throw new Error(messages.rankingRestoreSeatFailed);
            return putSeat(rankingID, removed.seat_no, {
              characterID: removed.character?.id ?? null,
              note: removed.note,
              version: current.version,
            });
          })
          .then(() => {
            showToast({ tone: "success", message: messages.rankingAdjusted(removed.seat_no) });
            reload();
          })
          .catch((cause: unknown) => {
            showToast({ tone: "error", message: toUserMessage(cause, messages.rankingRestoreSeatFailed) });
          });
      },
    });
  }

  function handleDeleted(result: DeleteRankingResult, name: string) {
    showToast({
      tone: "undo",
      message: messages.rankingMovedToTrash(name),
      actionLabel: messages.undo,
      onAction: () => {
        undoTrash(result.undo_token)
          .then(() => showToast({ tone: "success", message: messages.rankingUndone(name) }))
          .catch((cause: unknown) => showToast({ tone: "error", message: toUserMessage(cause, messages.saveFailed) }));
      },
    });
    router.push("/rankings");
  }

  if (status === "not_found") {
    return (
      <div className="page">
        <EmptyState
          variant="page"
          title={messages.rankingNotFoundTitle}
          description={messages.rankingNotFoundDescription}
          action={
            <Link className="ui-button ui-button--secondary ui-button--md" href="/rankings">
              <span className="ui-button__label">{messages.rankingBackToList}</span>
            </Link>
          }
        />
      </div>
    );
  }

  return (
    <div className="page">
      <div className="page-breadcrumb">
        <Link className="ui-button ui-button--ghost ui-button--sm" href="/rankings">
          <ArrowLeft aria-hidden="true" size={16} />
          {messages.rankingBackToList}
        </Link>
      </div>

      <PageHeader
        title={detail?.name ?? messages.rankingTitle}
        description={messages.rankingDetailDescription}
        actions={
          detail ? (
            <div className="page-actions">
              <Button variant="secondary" onClick={() => setCapacityOpen(true)}>
                {messages.rankingCapacity}
              </Button>
              <Button variant="danger" onClick={() => setDeleting(true)}>
                {messages.rankingDelete}
              </Button>
            </div>
          ) : undefined
        }
      />

      {status === "loading" && !detail ? (
        <div aria-busy="true">
          <Skeleton variant="lines" lines={8} label={messages.rankingDetailDescription} />
        </div>
      ) : null}

      {status === "error" ? (
        <Alert
          tone="danger"
          title={toUserMessage(error, messages.loadFailed)}
          action={
            <Button variant="secondary" size="sm" onClick={reload}>
              {messages.retry}
            </Button>
          }
        />
      ) : null}

      {conflict ? (
        <Alert
          tone="warning"
          title={messages.rankingVersionConflict}
          action={
            <Button variant="secondary" size="sm" onClick={reload}>
              {messages.rankingRefresh}
            </Button>
          }
        />
      ) : null}

      {detail ? (
        <>
          <section className="ranking-summary" aria-label={messages.rankingTitle}>
            <div className="ranking-summary__item">
              <span className="ranking-summary__label">{messages.rankingSummarySeats}</span>
              <span className="ranking-summary__value">{detail.seat_count}</span>
            </div>
            <div className="ranking-summary__item">
              <span className="ranking-summary__label">{messages.rankingSummaryOccupied}</span>
              <span className="ranking-summary__value">{detail.occupied_count}</span>
            </div>
            <div className="ranking-summary__item">
              <span className="ranking-summary__label">{messages.rankingSummaryEmpty}</span>
              <span className="ranking-summary__value">{detail.empty_count}</span>
            </div>
            {detail.review_status === "pending" ? (
              <Badge tone="warning" outlined>
                {messages.characterReviewPending}
              </Badge>
            ) : null}
            <div className="ranking-rename">
              <Input
                value={nameValue}
                maxLength={200}
                aria-label={messages.rankingNameLabel}
                onChange={(event) => setNameValue(event.target.value)}
              />
              <Button variant="secondary" size="sm" onClick={saveName} disabled={nameSaving}>
                {nameSaving ? messages.saving : messages.rankingRenameSubmit}
              </Button>
            </div>
          </section>

          <div className="ranking-seat-toolbar">
            <Select
              aria-label={messages.rankingSeatFilterLabel}
              value={seatFilter}
              options={seatFilterOptions}
              onChange={(event) => {
                setSeatFilter(event.target.value as SeatFilter);
                setPage(1);
              }}
            />
            <JumpToSeat seatCount={detail.seat_count} pageSize={SEAT_PAGE_SIZE} onJump={jumpToSeat} />
            {seats ? (
              <span className="character-muted">
                {messages.rankingSeatPage(seats.page, seats.page_size)}
              </span>
            ) : null}
          </div>

          <SeatTable
            rankingID={rankingID}
            seats={seats?.items ?? []}
            loading={status === "loading"}
            highlightSeat={highlight}
            onAssign={setAssignSeat}
            onAdjust={setAdjustSeat}
            onRemove={setRemoveSeat}
            onNoteSaved={(updated) => {
              showToast({
                tone: "success",
                message: updated.note
                  ? messages.rankingSeatNoteSaved(updated.seat_no)
                  : messages.rankingSeatNoteCleared(updated.seat_no),
              });
              reload();
            }}
          />

          {seats && seats.total > SEAT_PAGE_SIZE ? (
            <Pagination page={seats.page} pageSize={SEAT_PAGE_SIZE} total={seats.total} onChange={setPage} />
          ) : null}

          <AssignSeatDialog
            open={Boolean(assignSeat)}
            rankingID={rankingID}
            workID={detail.work_id}
            seat={assignSeat}
            seats={seats?.items ?? []}
            onOpenChange={(open) => {
              if (!open) setAssignSeat(null);
            }}
            onAssigned={(updated) => {
              showToast({
                tone: "success",
                message: messages.rankingAssigned(updated.seat_no, updated.character?.name ?? ""),
              });
              setAssignSeat(null);
              reload();
            }}
            onJumpSeat={(seatNo) => jumpToSeat(seatNo, Math.floor((seatNo - 1) / SEAT_PAGE_SIZE) + 1)}
          />

          <AdjustSeatDialog
            open={Boolean(adjustSeat)}
            rankingID={rankingID}
            seat={adjustSeat}
            seats={seats?.items ?? []}
            seatCount={detail.seat_count}
            onOpenChange={(open) => {
              if (!open) setAdjustSeat(null);
            }}
            onAdjusted={(targetSeatNo) => {
              showToast({ tone: "success", message: messages.rankingAdjusted(targetSeatNo) });
              setAdjustSeat(null);
              jumpToSeat(targetSeatNo, Math.floor((targetSeatNo - 1) / SEAT_PAGE_SIZE) + 1);
              reload();
            }}
          />

          <RemoveSeatDialog
            open={Boolean(removeSeat)}
            rankingID={rankingID}
            seat={removeSeat}
            onOpenChange={(open) => {
              if (!open) setRemoveSeat(null);
            }}
            onRemoved={handleRemoved}
          />

          <CapacityDialog
            open={capacityOpen}
            rankingID={rankingID}
            seatCount={detail.seat_count}
            version={detail.version}
            onOpenChange={setCapacityOpen}
            onUpdated={(updated) => {
              setDetail(updated);
              setCapacityOpen(false);
              setPage(1);
              showToast({ tone: "success", message: messages.rankingCapacityUpdated(updated.seat_count) });
              reload();
            }}
            onJumpSeat={(seatNo) => jumpToSeat(seatNo, Math.floor((seatNo - 1) / SEAT_PAGE_SIZE) + 1)}
          />

          <DeleteRankingDialog
            ranking={deleting ? { id: detail.id, name: detail.name, version: detail.version } : null}
            onOpenChange={(open) => {
              if (!open) setDeleting(false);
            }}
            onDeleted={handleDeleted}
          />
        </>
      ) : null}
    </div>
  );
}
