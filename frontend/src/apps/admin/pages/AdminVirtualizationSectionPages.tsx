import { AdminVirtualizationPage } from "./AdminVirtualizationPage";

export function AdminVirtualizationHostsPage() {
  return <AdminVirtualizationPage section="hosts" />;
}

export function AdminVirtualizationContainersPage() {
  return <AdminVirtualizationPage section="containers" />;
}

export function AdminVirtualizationGroupsPage() {
  return <AdminVirtualizationPage section="groups" />;
}

export function AdminVirtualizationTemplatesPage() {
  return <AdminVirtualizationPage section="templates" />;
}

export function AdminVirtualizationHostDetailPage() {
  return <AdminVirtualizationPage section="host-detail" />;
}

export function AdminVirtualizationHostImagesPage() {
  return <AdminVirtualizationPage section="host-images" />;
}

export function AdminVirtualizationHostContainersPage() {
  return <AdminVirtualizationPage section="host-containers" />;
}
