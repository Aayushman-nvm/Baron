import { useParams, Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { useProject } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { AppSidebar } from '@/components/AppSidebar'
import { ActivityRightPanel } from '@/components/ActivityRightPanel'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLayout } from '@/contexts/LayoutContext'
import { Spinner } from '@/components/ui/Spinner'
import { GitBranch } from 'lucide-react'
import { WorkItemListPage } from './WorkItemListPage'
import { WorkItemDetailPage } from './WorkItemDetailPage'
import { ProjectSettingsPage } from './ProjectSettingsPage'
import { ProjectOverviewPage } from './ProjectOverviewPage'
import { ProjectWorkflowsPage } from './ProjectWorkflowsPage'
import { MilestonesPage } from './MilestonesPage'
import { MilestoneDashboardPage } from './MilestoneDashboardPage'
import { QueuesPage } from './QueuesPage'
import { QueueSettingsPage } from './QueueSettingsPage'
import { TeamDetailPage } from './TeamDetailPage'
import { QueueWorkItemsPage } from './QueueWorkItemsPage'
import { PortalTicketListPage } from './PortalTicketListPage'
import { PortalTicketDetailPage } from './PortalTicketDetailPage'
import type { ActivityTab } from '@/components/ActivityRightPanel'

export function ProjectDetailPage() {
  const { t } = useTranslation()
  const { collapsed } = useSidebar('app')
  const { containerClass } = useLayout()
  const { namespace, projectKey } = useParams<{ namespace: string; projectKey: string }>()
  const { user } = useAuth()

  // Right panel state — mutually exclusive with left sidebar on mobile
  const [rightOpen, setRightOpen] = useState(false)
  const [activeTab, setActiveTab] = useState<ActivityTab>('logs')

  function closeRight() {
    setRightOpen(false)
  }

  // Determine if the user is a customer in this project
  const isCustomerProject = user?.global_role !== 'admin'
    && (user?.portal_projects ?? []).some((p) => p.project_key === projectKey)

  // Skip the regular project API call for customer projects (ExcludeCustomer middleware blocks it)
  const { data: project, isLoading, error } = useProject(isCustomerProject ? '' : (projectKey ?? ''))

  if (isCustomerProject) {
    // Absolute path used for the catch-all: a relative `to="support"` inside
    // a splat route gets appended to the current pathname, causing infinite
    // "/support/support/support…" loops if the user lands on a non-matching
    // URL (e.g. a stale `/items/:num` link from a search result). The
    // namespace param is read from the URL (not context) so the redirect
    // always stays in the same namespace the user is viewing.
    const supportHome = `/${namespace ?? 'd'}/projects/${projectKey}/support`
    return (
      <div className={`${containerClass(true)} py-6`}>
        <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
          <AppSidebar projectKey={projectKey} customerProject />
          <div className="flex-1 min-w-0">
            <Routes>
              <Route index element={<Navigate to={supportHome} replace />} />
              <Route path="support" element={<PortalTicketListPage />} />
              <Route path="support/:itemNumber" element={<PortalTicketDetailPage />} />
              <Route path="*" element={<Navigate to={supportHome} replace />} />
            </Routes>
          </div>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Spinner size="lg" />
      </div>
    )
  }

  if (error || !project) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <p className="text-red-600">{t('projects.notFound')}</p>
      </div>
    )
  }

  return (
    <div className={`${containerClass(true)} py-6 relative`}>
      {/* Outer flex: left sidebar + main content */}
      <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        {/* Left sidebar */}
        <AppSidebar projectKey={project.key} />

        {/* Main content — shrinks when right panel is open on desktop */}
        <div className={`flex-1 min-w-0 transition-all duration-200 ${rightOpen ? 'lg:mr-72' : ''}`}>
          <Routes>
            <Route index element={<ProjectOverviewPage />} />
            <Route path="items" element={<WorkItemListPage />} />
            <Route path="items/:itemNumber" element={<WorkItemDetailPage />} />
            <Route path="queues" element={<QueuesPage />} />
            <Route path="queues/:queueId" element={<QueueSettingsPage />} />
            <Route path="queues/:queueId/items" element={<QueueWorkItemsPage />} />
            <Route path="teams" element={<Navigate to="../settings?tab=teams" replace />} />
            <Route path="teams/:teamId" element={<TeamDetailPage />} />
            <Route path="milestones" element={<MilestonesPage />} />
            <Route path="milestones/:milestoneId" element={<MilestoneDashboardPage />} />
            <Route path="workflows" element={<ProjectWorkflowsPage />} />
            <Route path="settings" element={<ProjectSettingsPage />} />
          </Routes>
        </div>
      </div>

      {/* Toggle button — always fixed bottom-right, visible when panel is closed */}
      {!rightOpen && (
        <button
          onClick={() => setRightOpen(true)}
          title={t('activity.panel.open')}
          className="fixed bottom-6 right-4 z-40 flex items-center gap-1.5 px-3 py-2 rounded-full bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium shadow-lg transition-colors"
        >
          <GitBranch className="h-4 w-4" />
          <span className="hidden sm:inline">{t('activity.panel.logs')}</span>
        </button>
      )}

      {/* Right panel — always fixed on the right edge, overlaid */}
      {rightOpen && (
        <>
          {/* Backdrop (closes panel on click, mobile and desktop) */}
          <div
            className="fixed inset-0 z-40 bg-black/20 lg:bg-transparent lg:pointer-events-none"
            onClick={closeRight}
          />
          {/* Panel */}
          <div className="fixed top-0 right-0 bottom-0 z-50 w-80 lg:w-72 flex flex-col shadow-2xl">
            <ActivityRightPanel
              projectKey={project.key}
              namespaceSlug={namespace}
              onClose={closeRight}
              activeTab={activeTab}
              onTabChange={setActiveTab}
            />
          </div>
        </>
      )}
    </div>
  )
}
