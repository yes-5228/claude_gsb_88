// 路由装配：按业务模块拆分页面，每个模块各自拥有列表 / 详情 / 表单路由。
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { AppLayout } from './components/AppLayout';
import { ToastProvider } from './components/Toast';
import { AcceptanceDetailPage } from './features/acceptances/AcceptanceDetailPage';
import { AcceptanceFormPage } from './features/acceptances/AcceptanceFormPage';
import { AcceptanceListPage } from './features/acceptances/AcceptanceListPage';
import { DashboardPage } from './features/dashboard/DashboardPage';
import { NotFoundPage } from './features/NotFoundPage';
import { RecordDetailPage } from './features/records/RecordDetailPage';
import { RecordFormPage } from './features/records/RecordFormPage';
import { RecordListPage } from './features/records/RecordListPage';
import { SegmentDetailPage } from './features/segments/SegmentDetailPage';
import { SegmentFormPage } from './features/segments/SegmentFormPage';
import { SegmentListPage } from './features/segments/SegmentListPage';
import { TaskDetailPage } from './features/tasks/TaskDetailPage';
import { TaskFormPage } from './features/tasks/TaskFormPage';
import { TaskListPage } from './features/tasks/TaskListPage';
import { MetaProvider } from './providers/MetaProvider';

export function App() {
  return (
    <BrowserRouter>
      <ToastProvider>
        <MetaProvider>
          <Routes>
            <Route element={<AppLayout />}>
              {/* 总览 */}
              <Route index element={<DashboardPage />} />

              {/* 管段台账 */}
              <Route path="segments" element={<SegmentListPage />} />
              <Route path="segments/new" element={<SegmentFormPage />} />
              <Route path="segments/:id" element={<SegmentDetailPage />} />
              <Route path="segments/:id/edit" element={<SegmentFormPage />} />

              {/* 清淤任务 */}
              <Route path="tasks" element={<TaskListPage />} />
              <Route path="tasks/new" element={<TaskFormPage />} />
              <Route path="tasks/:id" element={<TaskDetailPage />} />
              <Route path="tasks/:id/edit" element={<TaskFormPage />} />

              {/* 清淤记录 */}
              <Route path="records" element={<RecordListPage />} />
              <Route path="records/new" element={<RecordFormPage />} />
              <Route path="records/:id" element={<RecordDetailPage />} />
              <Route path="records/:id/edit" element={<RecordFormPage />} />

              {/* 验收记录 */}
              <Route path="acceptances" element={<AcceptanceListPage />} />
              <Route path="acceptances/new" element={<AcceptanceFormPage />} />
              <Route path="acceptances/:id" element={<AcceptanceDetailPage />} />

              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Routes>
        </MetaProvider>
      </ToastProvider>
    </BrowserRouter>
  );
}
