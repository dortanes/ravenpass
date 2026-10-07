import { access } from "./access.ts";
import { autofill } from "./autofill.ts";
import { breaches } from "./breaches.ts";
import { card } from "./card.ts";
import { codes } from "./codes.ts";
import { confirmation } from "./confirmation.ts";
import { extension } from "./extension.ts";
import { failures } from "./failures.ts";
import { general } from "./general.ts";
import { generator } from "./generator.ts";
import { identity } from "./identity.ts";
import { imports } from "./imports.ts";
import { intro } from "./intro.ts";
import { merge } from "./merge.ts";
import { note } from "./note.ts";
import { saving } from "./saving.ts";
import { seed } from "./seed.ts";
import { settings } from "./settings.ts";
import { sharing } from "./sharing.ts";
import { system } from "./system.ts";
import { trash } from "./trash.ts";
import { workspace } from "./workspace.ts";

export const en = {
  ...failures,
  ...general,
  ...intro,
  ...access,
  ...workspace,
  ...codes,
  ...identity,
  ...card,
  ...note,
  ...seed,
  ...settings,
  ...imports,
  ...extension,
  ...saving,
  ...autofill,
  ...sharing,
  ...confirmation,
  ...system,
  ...trash,
  ...merge,
  ...generator,
  ...breaches,
};
