// 应用入口：挂载 React 根节点。
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './App';
import './styles/app.css';

const container = document.getElementById('root');
if (!container) {
  throw new Error('缺少 #root 挂载节点');
}

createRoot(container).render(
  <StrictMode>
    <App />
  </StrictMode>
);
