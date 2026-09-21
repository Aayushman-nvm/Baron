import { useMemo, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { X, GitBranch, AlignLeft } from 'lucide-react'
import { useProjectEvents } from '@/hooks/useWorkItems'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { Spinner } from '@/components/ui/Spinner'
import type { ProjectEvent } from '@/api/workitems'

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

function eventLabel(e: ProjectEvent, t: ReturnType<typeof useTranslation>['t']): string {
  if (e.event_type === 'created') return t('activity.createdItem')
  if (e.event_type === 'comment_added') return t('activity.addedComment')
  if (e.event_type === 'comment_updated') return t('activity.editedComment')
  if (e.event_type === 'comment_deleted') return t('activity.deletedComment')
  if (e.field_name) {
    const field = t(`activity.fields.${e.field_name}`, { defaultValue: e.field_name.replace(/_/g, ' ') })
    if (e.old_value && e.new_value) return t('activity.changed', { field })
    if (e.new_value) return t('activity.set', { field })
    if (e.old_value) return t('activity.cleared', { field })
  }
  return e.event_type.replace(/_/g, ' ')
}

function relTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return new Date(iso).toLocaleDateString()
}

// ─────────────────────────────────────────────────────────────────────────────
// Log view — chronological flat list
// ─────────────────────────────────────────────────────────────────────────────

function LogView({ events, projectKey, onItemClick }: {
  events: ProjectEvent[]
  projectKey: string
  onItemClick: (itemNumber: number) => void
}) {
  const { t } = useTranslation()
  const bottomRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom on new events
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [events.length])

  if (!events.length) {
    return (
      <p className="text-sm text-gray-400 dark:text-gray-500 text-center py-8">
        {t('activity.noActivity')}
      </p>
    )
  }

  return (
    <div className="space-y-0 overflow-y-auto flex-1">
      {events.map((e) => (
        <div
          key={e.id}
          className="group px-3 py-2.5 hover:bg-gray-50 dark:hover:bg-gray-800/60 cursor-pointer border-b border-gray-100 dark:border-gray-800"
          onClick={() => onItemClick(e.item_number)}
        >
          {/* Item badge + actor */}
          <div className="flex items-center gap-1.5 text-xs mb-0.5">
            <span className="font-mono font-semibold text-indigo-600 dark:text-indigo-400 shrink-0">
              {e.display_id}
            </span>
            <span className="text-gray-400 dark:text-gray-500 truncate flex-1 min-w-0">
              {e.item_title}
            </span>
          </div>
          {/* Event description */}
          <div className="text-xs text-gray-600 dark:text-gray-300">
            <span className="font-medium">{e.actor?.display_name ?? t('common.system')}</span>
            {' '}
            <span className="text-gray-500 dark:text-gray-400">{eventLabel(e, t)}</span>
          </div>
          {/* Timestamp */}
          <div className="text-[11px] text-gray-400 dark:text-gray-500 mt-0.5">
            {new Date(e.created_at).toLocaleString()}
          </div>
        </div>
      ))}
      <div ref={bottomRef} />
    </div>
  )
}

// ─────────────────────────────────────────────────────────────────────────────
// Commit graph — GitHub-style branching visualization
// ─────────────────────────────────────────────────────────────────────────────

interface GraphNode {
  event: ProjectEvent
  lane: number       // horizontal column (0 = trunk)
  isBranchStart: boolean
  isBranchEnd: boolean
  parentLane: number // lane of the trunk row this branches from/to
}

/**
 * Assign lanes: each unique work item that deviates from the main timeline
 * gets its own lane. The trunk (lane 0) carries one event per unique timestamp
 * tick. Items with more than 1 event get a side branch.
 *
 * Strategy:
 * - Group consecutive events by work_item_id.
 * - If an item has a run of > 1 consecutive events, put that run in a side lane.
 * - Otherwise keep on trunk.
 */
function buildGraph(events: ProjectEvent[]): GraphNode[] {
  if (!events.length) return []

  const nodes: GraphNode[] = []
  // lane assignment: item_id → lane number
  const laneMap = new Map<string, number>()
  let nextLane = 1
  let i = 0

  while (i < events.length) {
    const e = events[i]
    // Count consecutive events for this item
    let runEnd = i + 1
    while (runEnd < events.length && events[runEnd].work_item_id === e.work_item_id) runEnd++
    const runLength = runEnd - i

    if (runLength === 1) {
      // Single event → trunk
      nodes.push({ event: e, lane: 0, isBranchStart: false, isBranchEnd: false, parentLane: 0 })
      i++
    } else {
      // Multi-event run → side branch
      let lane = laneMap.get(e.work_item_id)
      if (!lane) {
        lane = nextLane++
        laneMap.set(e.work_item_id, lane)
      }
      for (let j = i; j < runEnd; j++) {
        nodes.push({
          event: events[j],
          lane,
          isBranchStart: j === i,
          isBranchEnd: j === runEnd - 1,
          parentLane: 0,
        })
      }
      i = runEnd
    }
  }

  return nodes
}

const LANE_WIDTH = 16  // px per lane
const ROW_HEIGHT = 44  // px per row
const DOT_R = 4        // dot radius
const TRUNK_COLOR = '#6366f1'  // indigo-500
const BRANCH_COLORS = ['#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6']

function CommitGraph({ events, projectKey, onItemClick }: {
  events: ProjectEvent[]
  projectKey: string
  onItemClick: (itemNumber: number) => void
}) {
  const { t } = useTranslation()
  const nodes = useMemo(() => buildGraph(events), [events])

  if (!nodes.length) {
    return (
      <p className="text-sm text-gray-400 dark:text-gray-500 text-center py-8">
        {t('activity.noActivity')}
      </p>
    )
  }

  const maxLane = Math.max(...nodes.map(n => n.lane))
  const svgWidth = (maxLane + 1) * LANE_WIDTH + 8
  const svgHeight = nodes.length * ROW_HEIGHT

  function laneColor(lane: number): string {
    if (lane === 0) return TRUNK_COLOR
    return BRANCH_COLORS[(lane - 1) % BRANCH_COLORS.length]
  }

  function cx(lane: number) { return lane * LANE_WIDTH + DOT_R + 4 }
  function cy(idx: number) { return idx * ROW_HEIGHT + ROW_HEIGHT / 2 }

  // Build SVG lines between nodes
  const lines: React.ReactNode[] = []
  for (let idx = 0; idx < nodes.length - 1; idx++) {
    const cur = nodes[idx]
    const next = nodes[idx + 1]

    if (cur.lane === next.lane) {
      // Straight vertical line
      lines.push(
        <line key={`l-${idx}`}
          x1={cx(cur.lane)} y1={cy(idx)} x2={cx(next.lane)} y2={cy(idx + 1)}
          stroke={laneColor(cur.lane)} strokeWidth={2}
        />
      )
    } else if (cur.isBranchEnd && next.lane === cur.parentLane) {
      // Branch merges back to trunk
      lines.push(
        <path key={`l-${idx}`}
          d={`M ${cx(cur.lane)} ${cy(idx)} C ${cx(cur.lane)} ${cy(idx) + ROW_HEIGHT * 0.5} ${cx(next.lane)} ${cy(idx + 1) - ROW_HEIGHT * 0.3} ${cx(next.lane)} ${cy(idx + 1)}`}
          stroke={laneColor(cur.lane)} strokeWidth={2} fill="none"
        />
      )
    } else if (next.isBranchStart && cur.lane === next.parentLane) {
      // Branch starts from trunk
      lines.push(
        <path key={`l-${idx}`}
          d={`M ${cx(cur.lane)} ${cy(idx)} C ${cx(cur.lane)} ${cy(idx) + ROW_HEIGHT * 0.3} ${cx(next.lane)} ${cy(idx + 1) - ROW_HEIGHT * 0.5} ${cx(next.lane)} ${cy(idx + 1)}`}
          stroke={laneColor(next.lane)} strokeWidth={2} fill="none"
        />
      )
    } else {
      // Same lane continuation
      lines.push(
        <line key={`l-${idx}`}
          x1={cx(cur.lane)} y1={cy(idx)} x2={cx(next.lane)} y2={cy(idx + 1)}
          stroke={laneColor(cur.lane)} strokeWidth={2}
        />
      )
    }

    // Also keep trunk running through branch rows
    if (cur.lane !== 0 && !cur.isBranchStart) {
      // trunk already drawn by previous iteration
    }
    if (cur.lane !== 0) {
      // Draw trunk line alongside branch
      lines.push(
        <line key={`t-${idx}`}
          x1={cx(0)} y1={cy(idx)} x2={cx(0)} y2={cy(idx + 1)}
          stroke={laneColor(0)} strokeWidth={2} opacity={0.3}
        />
      )
    }
  }

  return (
    <div className="overflow-y-auto flex-1">
      <div className="relative flex" style={{ minHeight: svgHeight }}>
        {/* SVG graph on the left */}
        <svg
          width={svgWidth}
          height={svgHeight}
          className="shrink-0"
          style={{ minWidth: svgWidth }}
        >
          {lines}
          {nodes.map((node, idx) => (
            <circle
              key={node.event.id}
              cx={cx(node.lane)}
              cy={cy(idx)}
              r={DOT_R}
              fill={laneColor(node.lane)}
              stroke="white"
              strokeWidth={1.5}
              className="cursor-pointer"
              onClick={() => onItemClick(node.event.item_number)}
            />
          ))}
        </svg>

        {/* Labels on the right */}
        <div className="flex-1 min-w-0">
          {nodes.map((node, idx) => {
            const e = node.event
            return (
              <div
                key={e.id}
                className="flex flex-col justify-center px-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/60 rounded"
                style={{ height: ROW_HEIGHT }}
                onClick={() => onItemClick(e.item_number)}
              >
                <div className="flex items-center gap-1 min-w-0">
                  <span className="font-mono text-[11px] font-semibold text-indigo-600 dark:text-indigo-400 shrink-0">
                    {e.display_id}
                  </span>
                  <span className="text-[11px] text-gray-500 dark:text-gray-400 truncate">
                    {eventLabel(e, t)}
                  </span>
                </div>
                <div className="flex items-center gap-1 min-w-0">
                  <span className="text-[10px] text-gray-400 dark:text-gray-500 truncate">
                    {e.actor?.display_name ?? t('common.system')} · {relTime(e.created_at)}
                  </span>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

// ─────────────────────────────────────────────────────────────────────────────
// Main panel component
// ─────────────────────────────────────────────────────────────────────────────

export type ActivityTab = 'logs' | 'graph'

interface ActivityRightPanelProps {
  projectKey: string
  namespaceSlug?: string
  onClose: () => void
  activeTab: ActivityTab
  onTabChange: (tab: ActivityTab) => void
}

export function ActivityRightPanel({
  projectKey,
  namespaceSlug,
  onClose,
  activeTab,
  onTabChange,
}: ActivityRightPanelProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { p } = useNamespacePath()

  // Debug: confirm component mounts and what projectKey is received
  console.log('[ActivityRightPanel] mounted, projectKey=', projectKey, 'ns=', namespaceSlug)

  const { data: events = [], isLoading, error } = useProjectEvents(projectKey, namespaceSlug)

  // Debug: log query state
  console.log('[ActivityRightPanel] events=', events.length, 'loading=', isLoading, 'error=', error)

  function handleItemClick(itemNumber: number) {
    navigate(p(`/projects/${projectKey}/items/${itemNumber}`))
  }

  return (
    <div className="flex flex-col h-full bg-white dark:bg-gray-900 border-l border-gray-200 dark:border-gray-700 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-gray-700 shrink-0">
        <div className="flex items-center gap-1">
          <button
            onClick={() => onTabChange('logs')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium transition-colors ${
              activeTab === 'logs'
                ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                : 'text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
            }`}
          >
            <AlignLeft className="h-3.5 w-3.5" />
            {t('activity.panel.logs')}
          </button>
          <button
            onClick={() => onTabChange('graph')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium transition-colors ${
              activeTab === 'graph'
                ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                : 'text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800'
            }`}
          >
            <GitBranch className="h-3.5 w-3.5" />
            {t('activity.panel.graph')}
          </button>
        </div>
        <button
          onClick={onClose}
          className="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-400 dark:text-gray-500"
          aria-label={t('common.close')}
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Body */}
      {isLoading ? (
        <div className="flex items-center justify-center flex-1">
          <Spinner size="sm" />
        </div>
      ) : error ? (
        <div className="p-4 text-sm text-red-500">
          Failed to load: {(error as Error).message}
        </div>
      ) : activeTab === 'logs' ? (
        <LogView events={events} projectKey={projectKey} onItemClick={handleItemClick} />
      ) : (
        <CommitGraph events={events} projectKey={projectKey} onItemClick={handleItemClick} />
      )}
    </div>
  )
}
