import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ConfigProvider, App as AntdApp, theme } from "antd";
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
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: "#1e6f64",
          colorSuccess: "#2f855a",
          colorWarning: "#b7791f",
          colorError: "#c2413d",
          colorInfo: "#1e6f64",
          colorText: "#171717",
          colorTextSecondary: "#5f6360",
          colorBgLayout: "#f4f6f5",
          colorBgContainer: "#ffffff",
          colorBorder: "#dde3df",
          borderRadius: 8,
          controlHeight: 38,
          fontFamily:
            "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif",
        },
        components: {
          Button: {
            borderRadius: 7,
            controlHeight: 38,
            primaryShadow: "none",
          },
          Card: {
            borderRadiusLG: 8,
          },
          Modal: {
            borderRadiusLG: 8,
          },
        },
      }}
    >
      <AntdApp>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </AntdApp>
    </ConfigProvider>
  );
}
