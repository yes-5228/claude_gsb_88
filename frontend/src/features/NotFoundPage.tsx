// 未匹配路由的兜底页面。
import { Link } from 'react-router-dom';
import { PageHeader } from '../components/PageHeader';

export function NotFoundPage() {
  return (
    <div className="page">
      <PageHeader title="页面不存在" description="请确认访问地址是否正确，或通过左侧导航进入对应业务模块。" />
      <div className="card">
        <div className="card-body">
          <Link className="btn btn-ghost" to="/">
            返回运行看板
          </Link>
        </div>
      </div>
    </div>
  );
}
