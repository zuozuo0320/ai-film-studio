import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ConfigProvider, App as AntdApp } from "antd";
import zhCN from "antd/locale/zh_CN";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { ProjectListPage } from "../features/project/ProjectListPage";
import { WorkbenchPage } from "../features/project/WorkbenchPage";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000, refetchOnWindowFocus: false },
  },
});

const router = createBrowserRouter([
  { path: "/", element: <ProjectListPage /> },
  { path: "/projects/:id", element: <WorkbenchPage /> },
]);

export function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <AntdApp>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </AntdApp>
    </ConfigProvider>
  );
}
