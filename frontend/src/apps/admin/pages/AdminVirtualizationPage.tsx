import { HostPanel } from "./virtualization/HostPanel";
import { HostDetailPanel } from "./virtualization/HostDetailPanel";
import { ContainerGroupPanel, ContainerPanel } from "./virtualization/ContainerPanels";
import { TemplatePanel } from "./virtualization/TemplatePanel";
import { sectionMeta } from "./virtualization/constants";
import type { VirtualizationSection } from "./virtualization/types";

type AdminVirtualizationPageProps = {
  section?: VirtualizationSection;
};

export function AdminVirtualizationPage({
  section = "hosts",
}: AdminVirtualizationPageProps) {
  const activeSection = normalizeSection(section);
  const meta = sectionMeta[activeSection];

  return (
    <div className="admin-page virtualization-page">
      {activeSection === "hosts" ? <HostPanel /> : null}
      {activeSection === "host-detail" ? (
        <HostDetailPanel mode="overview" />
      ) : null}
      {activeSection === "host-images" ? (
        <HostDetailPanel mode="images" />
      ) : null}
      {activeSection === "host-containers" ? (
        <HostDetailPanel mode="containers" />
      ) : null}
      {activeSection === "containers" ? <ContainerPanel /> : null}
      {activeSection === "groups" ? <ContainerGroupPanel /> : null}
      {activeSection === "templates" ? <TemplatePanel /> : null}
    </div>
  );
}

function normalizeSection(section: string): VirtualizationSection {
  if (
    section === "containers" ||
    section === "groups" ||
    section === "templates" ||
    section === "host-detail" ||
    section === "host-images" ||
    section === "host-containers"
  ) {
    return section;
  }
  return "hosts";
}
