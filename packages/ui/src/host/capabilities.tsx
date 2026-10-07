import { queryOptions, skipToken, useQuery } from "@tanstack/react-query";
import { createContext, type ReactNode, useContext } from "react";
import { queryKeys } from "../query/keys.ts";
import type { Capabilities, VaultApi } from "../vault-api.ts";
import { showAppearance } from "./appearance.ts";
import { showInterfaceSize } from "./interface-size.ts";

const noCapabilities: Capabilities = {
  extensions: false,
  shortcuts: false,
  dockIcon: false,
  storageLocations: false,
  saveFiles: false,
  unlockOnShow: false,
  interfaceSize: false,
  appearance: false,
  photoPicker: false,
  identityList: false,
  copyScans: false,
  lockWhenHidden: false,
  screenshots: false,
  systemAutofill: false,
  autoBackups: false,
  print: false,
  qrCodes: false,
};

const HostCapabilities = createContext<Capabilities>(noCapabilities);

/** interfaceSizeQuery reads the recorded interface size and lays the page out at it. */
export function interfaceSizeQuery(
  api: Pick<VaultApi, "interfaceSize"> | undefined,
  enabled: boolean,
) {
  return queryOptions({
    queryKey: queryKeys.interfaceSize,
    queryFn:
      api && enabled
        ? async () => {
            const size = await api.interfaceSize();
            showInterfaceSize(size.percent);
            return size;
          }
        : skipToken,
    meta: { failure: "app.error.setting-read" },
  });
}

/** appearanceQuery reads the recorded appearance and shows the page in it. */
export function appearanceQuery(
  api: Pick<VaultApi, "appearance"> | undefined,
  enabled: boolean,
) {
  return queryOptions({
    queryKey: queryKeys.appearance,
    queryFn:
      api && enabled
        ? async () => {
            const setting = await api.appearance();
            showAppearance(setting.appearance);
            return setting;
          }
        : skipToken,
    meta: { failure: "app.error.setting-read" },
  });
}

/** CapabilitiesProvider shows its children once the host has said what it offers and the page has its interface size and appearance. */
export function CapabilitiesProvider({
  api,
  children,
}: {
  api?: VaultApi;
  children: ReactNode;
}) {
  const answer = useQuery({
    queryKey: queryKeys.capabilities,
    queryFn: api ? () => api.capabilities() : skipToken,
    meta: { failure: "app.error.capabilities" },
  });
  const offers = !api || answer.isError ? noCapabilities : answer.data;
  const size = useQuery(
    interfaceSizeQuery(api, Boolean(offers?.interfaceSize)),
  );

  const appearance = useQuery(
    appearanceQuery(api, Boolean(offers?.appearance)),
  );

  if (
    !offers ||
    (offers.interfaceSize && size.isPending) ||
    (offers.appearance && appearance.isPending)
  ) {
    return null;
  }
  return (
    <HostCapabilities.Provider value={offers}>
      {children}
    </HostCapabilities.Provider>
  );
}

export function useCapabilities(): Capabilities {
  return useContext(HostCapabilities);
}
