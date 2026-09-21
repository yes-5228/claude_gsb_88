// 应用外壳：左侧模块导航 + 顶部状态条 + 业务内容区。
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { useMeta } from '../providers/MetaProvider';

interface NavItem {
  to: string;
  label: string;
  end?: boolean;
}

interface NavGroup {
  title: string;
  items: NavItem[];
}

const NAV_GROUPS: NavGroup[] = [
  { title: '总览', items: [{ to: '/', label: '运行看板', end: true }] },
  { title: '管网台账', items: [{ to: '/segments', label: '管段台账' }] },
  {
    title: '清淤作业',
    items: [
      { to: '/tasks', label: '清淤任务' },
      { to: '/records', label: '清淤记录' }
    ]
  },
  { title: '质量管理', items: [{ to: '/acceptances', label: '验收记录' }] }
];

/** 根据当前路径推断所属模块，显示在顶部状态条上。 */
function currentModule(pathname: string): string {
  const matched = NAV_GROUPS.flatMap((group) => group.items)
    .filter((item) => item.to !== '/' && pathname.startsWith(item.to))
    .sort((left, right) => right.to.length - left.to.length)[0];
  return matched ? matched.label : '运行看板';
}

function MetaStatus() {
  const { loading, error, reload, enums } = useMeta();
  if (loading) {
    return <span className="topbar-status">字典加载中…</span>;
  }
  if (error) {
    return (
      <span className="topbar-status topbar-status-error">
        {error}
        <button type="button" className="btn-link" onClick={reload}>
          重试
        </button>
      </span>
    );
  }
  const groups = enums ? Object.keys(enums).length : 0;
  return <span className="topbar-status">字典分组 {groups} 项</span>;
}

export function AppLayout() {
  const location = useLocation();

  return (
    <div className="app-shell">
      <aside className="app-sidebar">
        <div className="brand">
          <span className="brand-mark">排</span>
          <div className="brand-text">
            <strong>排水管网清淤记录系统</strong>
            <small>Drainage Desilting Records</small>
          </div>
        </div>
        <nav className="app-nav">
          {NAV_GROUPS.map((group) => (
            <div key={group.title} className="nav-group">
              <p className="nav-group-title">{group.title}</p>
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) => `nav-link${isActive ? ' nav-link-active' : ''}`}
                >
                  {item.label}
                </NavLink>
              ))}
            </div>
          ))}
        </nav>
        <footer className="app-sidebar-footer">
          <span>Fiber + React</span>
          <span>v1.0.0</span>
        </footer>
      </aside>
      <div className="app-main">
        <div className="app-topbar">
          <span className="topbar-module">{currentModule(location.pathname)}</span>
          <MetaStatus />
        </div>
        <main className="app-content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
