import { maskitoDate } from "@maskito/kit";
import { useMaskito } from "@maskito/react";
import { useQuery } from "@tanstack/react-query";
import { cn } from "cn";
import { LoaderCircle } from "lucide-react";
import { useReducedMotion } from "motion/react";
import { useMemo, useRef, useState } from "react";
import {
  type BillingChoice,
  billingChoiceOf,
  billingOf,
  cardColor,
  cardSwatches,
  digitsMask,
  digitsWithin,
  expiryDisplay,
  expiryStored,
  failsCheck,
  numberGroups,
  numberMask,
  readableOn,
} from "../cards/card.ts";
import { detectNetwork, digitsOf, networkName } from "../cards/networks.ts";
import type { MessageKey } from "../i18n/messages.ts";
import { useTranslator } from "../i18n/translator.tsx";
import { addressName, emptyAddress } from "../identities/identity.ts";
import { typeIn } from "../motion/type-in.ts";
import { queryKeys } from "../query/keys.ts";
import type {
  Address,
  AddressLink,
  Card,
  CardInput,
  CardLimits,
  Group,
  IdentityAddresses,
  VaultApi,
} from "../vault-api.ts";
import { hintSeen, markHintSeen } from "../workspace/hints.ts";
import { AddressRows } from "./editor/AddressRows.tsx";
import {
  bareField,
  EditorHeader,
  EditorHint,
  EditorRow,
  editorRow,
  GroupField,
  labelColumn,
  NotesField,
  TagField,
  type Tagging,
  toggled,
  useRemaining,
} from "./editor/EditorFields.tsx";
import { Input } from "./ui/input.tsx";
import { ScrollArea } from "./ui/scroll-area.tsx";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "./ui/select.tsx";
import { RevealButton } from "./workspace/Fields.tsx";

const formID = "card-editor";

const block = "min-w-0 shrink-0 overflow-hidden rounded-row bg-field";

const expiryMask = maskitoDate({ mode: "mm/yy", separator: "/" });

/** What the editor asks of the host besides saving. */
export type CardEditorHost = Pick<
  VaultApi,
  "lookupBank" | "listIdentityAddresses"
>;

type BillingKind = BillingChoice["kind"];

const noBilling = "none";
const ownBilling = "own";

const noIdentities: readonly IdentityAddresses[] = [];

function linkValue(link: AddressLink): string {
  return `link:${link.identityId}:${link.addressId}`;
}

export function CardEditor({
  initial,
  initialGroups,
  tagging,
  groups,
  limits,
  busy,
  bankLookup,
  host,
  onSave,
  onCancel,
  onFailure,
}: {
  initial?: Card;
  /** The groups the card starts in, which for a new one is the chosen default. */
  initialGroups: string[];
  tagging: Tagging;
  groups: Group[];
  limits: CardLimits | null;
  busy: boolean;
  /** Whether leaving the bank site asks the bank's site for its name and colour. */
  bankLookup: boolean;
  host: CardEditorHost;
  onSave: (input: CardInput, groups: string[]) => void;
  onCancel: () => void;
  onFailure: (cause: unknown, message: MessageKey) => void;
}) {
  const { t } = useTranslator();
  const counter = useRemaining();
  const [label, setLabel] = useState(initial?.label ?? "");
  const [digits, setDigits] = useState(initial?.number ?? "");
  const [holder, setHolder] = useState(initial?.holder ?? "");
  const [expiry, setExpiry] = useState(() =>
    expiryDisplay(initial?.expiry ?? ""),
  );
  const [securityCode, setSecurityCode] = useState(initial?.securityCode ?? "");
  const [pin, setPin] = useState(initial?.pin ?? "");
  const [codeShown, setCodeShown] = useState(false);
  const [pinShown, setPinShown] = useState(false);
  const [bankSite, setBankSite] = useState(initial?.bankSite ?? "");
  const [bankName, setBankName] = useState(initial?.bankName ?? "");
  // Null picks the automatic colour: the bank site's, else the network's.
  const initialColor = initial?.color ?? "";
  const [swatch, setSwatch] = useState<string | null>(
    cardSwatches.includes(initialColor) ? initialColor : null,
  );
  const [found, setFound] = useState(
    cardSwatches.includes(initialColor) ? "" : initialColor,
  );
  const picked = useRef(false);
  // A new site replaces the bank name only while it is empty or still the last lookup's.
  const lookedUp = useRef(initial?.bankSite.trim() ?? "");
  const filledName = useRef(initial?.bankName ?? "");
  const [lookingUp, setLookingUp] = useState(false);
  const reduceMotion = useReducedMotion() ?? false;
  // Shown once, on the first new card, while bank lookup is on.
  const [hintOpen, setHintOpen] = useState(
    () => !initial && !hintSeen("card-bank-site"),
  );
  const hintShown = hintOpen && bankLookup;
  const initialBilling = billingChoiceOf(initial);
  const [billingKind, setBillingKind] = useState<BillingKind>(
    initialBilling.kind,
  );
  const [link, setLink] = useState<AddressLink | null>(
    initialBilling.kind === "linked" ? initialBilling.link : null,
  );
  const [own, setOwn] = useState<Address>(
    initialBilling.kind === "own" ? initialBilling.address : emptyAddress,
  );
  const [notes, setNotes] = useState(initial?.notes ?? "");
  const [membership, setMembership] = useState<string[]>(initialGroups);
  const [tags, setTags] = useState<string[]>(initial?.tags ?? []);

  const network = detectNetwork(digits);
  const numberRef = useMaskito({
    options: useMemo(
      () => numberMask(network, limits?.numberMax),
      [network, limits?.numberMax],
    ),
  });
  const expiryRef = useMaskito({ options: expiryMask });
  const securityCodeRef = useMaskito({
    options: useMemo(
      () => digitsMask(limits?.securityCodeMax ?? 4),
      [limits?.securityCodeMax],
    ),
  });
  const pinRef = useMaskito({
    options: useMemo(() => digitsMask(limits?.pinMax ?? 12), [limits?.pinMax]),
  });

  const identities =
    useQuery({
      queryKey: queryKeys.identityAddresses,
      queryFn: () => host.listIdentityAddresses(),
      meta: { failure: "card.error.addresses" },
    }).data ?? noIdentities;

  function dismissHint() {
    if (!hintShown) return;
    setHintOpen(false);
    markHintSeen("card-bank-site");
  }

  // A name the user typed and a colour picked in this editor survive the lookup.
  async function lookUpBank() {
    const site = bankSite.trim();
    if (!bankLookup || !site || site === lookedUp.current) return;
    lookedUp.current = site;
    dismissHint();
    setLookingUp(true);
    try {
      const brand = await host.lookupBank(site);
      const previous = filledName.current;
      const replaceable = (current: string) =>
        !current.trim() || (previous !== "" && current === previous);
      filledName.current = brand.name;
      setFound(brand.color);
      if (!picked.current) setSwatch(null);
      await Promise.all([
        replaceable(bankName)
          ? typeIn(brand.name, setBankName, reduceMotion)
          : null,
        replaceable(label) ? typeIn(brand.name, setLabel, reduceMotion) : null,
      ]);
    } catch (cause) {
      onFailure(cause, "card.error.lookup");
    } finally {
      setLookingUp(false);
    }
  }

  function pickColor(color: string | null) {
    picked.current = true;
    setSwatch(color);
  }

  function chooseBilling(value: string) {
    if (value === noBilling || value === ownBilling) {
      setBillingKind(value);
      return;
    }
    const [, identityId, addressId] = value.split(":");
    if (identityId === undefined || addressId === undefined) return;
    setLink({ identityId, addressId });
    setBillingKind("linked");
  }

  const billing: BillingChoice =
    billingKind === "linked" && link
      ? { kind: "linked", link }
      : billingKind === "own"
        ? { kind: "own", address: own }
        : { kind: "none" };
  const storedExpiry = expiryStored(expiry);
  const numberComplete = limits
    ? digitsWithin(digits, limits.numberMin, limits.numberMax, false)
    : Boolean(digits);
  const codeComplete =
    !limits ||
    digitsWithin(
      securityCode,
      limits.securityCodeMin,
      limits.securityCodeMax,
      true,
    );
  const pinComplete =
    !limits || digitsWithin(pin, limits.pinMin, limits.pinMax, true);
  const unfinished = storedExpiry === null || !codeComplete || !pinComplete;
  const name = label.trim();
  // A linked address the list does not hold yet is still offered, under the name it was read with.
  const linkedFallback =
    initial?.linked &&
    initial.billingLink &&
    !identities.some(
      (identity) => identity.identityId === initial.billingLink?.identityId,
    )
      ? {
          identityId: initial.billingLink.identityId,
          label: initial.linked.identityLabel,
          addresses: [initial.linked.address],
        }
      : null;
  const choices = linkedFallback ? [linkedFallback, ...identities] : identities;

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (storedExpiry === null) return;
    onSave(
      {
        label: name,
        holder,
        number: digits,
        expiry: storedExpiry,
        securityCode,
        pin,
        network: detectNetwork(digits),
        bankName,
        bankSite,
        color: swatch ?? found,
        ...billingOf(billing),
        notes,
        tags,
      },
      membership,
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-[11px]">
      <EditorHeader
        title={t(initial ? "card.editor.edit.title" : "card.editor.new.title")}
        name={name}
        form={formID}
        submit={t(initial ? "workspace.editor.update" : "card.editor.create")}
        canSubmit={Boolean(name) && numberComplete && !unfinished && !lookingUp}
        busy={busy}
        onCancel={onCancel}
      />

      <ScrollArea className="min-h-0 flex-1">
        <form
          id={formID}
          onSubmit={submit}
          className="flex flex-1 flex-col gap-2"
        >
          <div className={block}>
            <EditorRow
              label={t("card.field.label")}
              htmlFor="card-name"
              counter={counter(label, limits?.label, 20)}
            >
              <Input
                id="card-name"
                className={bareField}
                value={label}
                onChange={(event) => setLabel(event.target.value)}
                placeholder={t("card.field.label.placeholder")}
                maxLength={limits?.label}
                disabled={busy || lookingUp}
                required
              />
            </EditorRow>
            <EditorRow label={t("card.field.number")} htmlFor="card-number">
              <Input
                id="card-number"
                ref={numberRef}
                className={cn(
                  bareField,
                  "font-mono tabular-nums aria-invalid:text-destructive",
                )}
                value={numberGroups(digits, network).join(" ")}
                onInput={(event) =>
                  setDigits(digitsOf(event.currentTarget.value))
                }
                inputMode="numeric"
                autoComplete="off"
                spellCheck={false}
                aria-invalid={
                  digits !== "" && !numberComplete ? true : undefined
                }
                disabled={busy}
                required
              />
              {failsCheck(digits, network) && (
                <span className="shrink-0 text-[11px] text-warning">
                  {t("card.number.mistyped")}
                </span>
              )}
              {network && (
                <span className="shrink-0 text-[11px] text-muted-foreground">
                  {networkName(network)}
                </span>
              )}
            </EditorRow>
            <EditorRow
              label={t("card.field.holder")}
              htmlFor="card-holder"
              counter={counter(holder, limits?.holder, 20)}
            >
              <Input
                id="card-holder"
                className={bareField}
                value={holder}
                onChange={(event) => setHolder(event.target.value)}
                maxLength={limits?.holder}
                autoComplete="off"
                disabled={busy}
              />
            </EditorRow>
            <EditorRow label={t("card.field.expiry")} htmlFor="card-expiry">
              <Input
                id="card-expiry"
                ref={expiryRef}
                className={cn(
                  bareField,
                  "font-mono tabular-nums aria-invalid:text-destructive",
                )}
                value={expiry}
                onInput={(event) => setExpiry(event.currentTarget.value)}
                placeholder={t("card.field.expiry.placeholder")}
                inputMode="numeric"
                autoComplete="off"
                aria-invalid={storedExpiry === null ? true : undefined}
                disabled={busy}
              />
            </EditorRow>
            <EditorRow
              label={t("card.field.security-code")}
              htmlFor="card-security-code"
            >
              <Input
                id="card-security-code"
                ref={securityCodeRef}
                type={codeShown ? "text" : "password"}
                className={cn(
                  bareField,
                  "font-mono aria-invalid:text-destructive",
                )}
                value={securityCode}
                onInput={(event) => setSecurityCode(event.currentTarget.value)}
                inputMode="numeric"
                autoComplete="off"
                aria-invalid={codeComplete ? undefined : true}
                disabled={busy}
              />
              <RevealButton
                shown={codeShown}
                revealLabel={t("card.security-code.reveal")}
                concealLabel={t("card.security-code.conceal")}
                busy={busy}
                onToggle={() => setCodeShown((shown) => !shown)}
              />
            </EditorRow>
            <EditorRow label={t("card.field.pin")} htmlFor="card-pin">
              <Input
                id="card-pin"
                ref={pinRef}
                type={pinShown ? "text" : "password"}
                className={cn(
                  bareField,
                  "font-mono aria-invalid:text-destructive",
                )}
                value={pin}
                onInput={(event) => setPin(event.currentTarget.value)}
                inputMode="numeric"
                autoComplete="off"
                aria-invalid={pinComplete ? undefined : true}
                disabled={busy}
              />
              <RevealButton
                shown={pinShown}
                revealLabel={t("card.pin.reveal")}
                concealLabel={t("card.pin.conceal")}
                busy={busy}
                onToggle={() => setPinShown((shown) => !shown)}
              />
            </EditorRow>
          </div>

          <div className={block}>
            <EditorHint
              open={hintShown}
              text={t("card.hint.bank-site")}
              onDismiss={dismissHint}
            >
              <div className="border-b">
                <EditorRow
                  label={t("card.field.bank-site")}
                  htmlFor="card-bank-site"
                  counter={counter(bankSite, limits?.bankSite, 20)}
                >
                  <Input
                    id="card-bank-site"
                    className={bareField}
                    value={bankSite}
                    onChange={(event) => {
                      setBankSite(event.target.value);
                      dismissHint();
                    }}
                    onBlur={() => void lookUpBank()}
                    placeholder={t("card.field.bank-site.placeholder")}
                    maxLength={limits?.bankSite}
                    autoCapitalize="none"
                    spellCheck={false}
                    disabled={busy || lookingUp}
                  />
                  {lookingUp && (
                    <span
                      className="flex shrink-0 items-center gap-1.5 text-[11px] text-muted-foreground"
                      role="status"
                    >
                      <LoaderCircle
                        className="size-3.5 animate-spin motion-reduce:animate-none"
                        aria-hidden="true"
                      />
                      {t("card.bank.looking-up")}
                    </span>
                  )}
                </EditorRow>
              </div>
            </EditorHint>
            <EditorRow
              label={t("card.field.bank-name")}
              htmlFor="card-bank-name"
              counter={counter(bankName, limits?.bankName, 20)}
            >
              <Input
                id="card-bank-name"
                className={bareField}
                value={bankName}
                onChange={(event) => setBankName(event.target.value)}
                maxLength={limits?.bankName}
                autoComplete="off"
                disabled={busy || lookingUp}
              />
            </EditorRow>
            <ColorRow
              automatic={cardColor(found, network)}
              swatch={swatch}
              busy={busy || lookingUp}
              onPick={pickColor}
            />
          </div>

          <div className={block}>
            <EditorRow label={t("card.block.billing")} htmlFor="card-billing">
              <Select
                value={
                  billingKind === "linked" && link
                    ? linkValue(link)
                    : billingKind
                }
                onValueChange={chooseBilling}
                disabled={busy}
              >
                <SelectTrigger
                  id="card-billing"
                  size="sm"
                  className="h-7 w-full min-w-0 border-0 bg-transparent px-0 text-[13px] shadow-none focus-visible:ring-0"
                >
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={noBilling}>
                    {t("card.billing.none")}
                  </SelectItem>
                  {choices.map((identity) => (
                    <SelectGroup key={identity.identityId}>
                      <SelectSeparator />
                      <SelectLabel>
                        {identity.label || t("identity.untitled")}
                      </SelectLabel>
                      {identity.addresses.map((address, index) => (
                        <SelectItem
                          key={address.id}
                          value={linkValue({
                            identityId: identity.identityId,
                            addressId: address.id,
                          })}
                        >
                          {addressName(address) ||
                            t("identity.address.untitled", {
                              number: index + 1,
                            })}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  ))}
                  <SelectSeparator />
                  <SelectItem value={ownBilling}>
                    {t("card.billing.own")}
                  </SelectItem>
                </SelectContent>
              </Select>
            </EditorRow>
            {billingKind === "own" && (
              <AddressRows
                id="card-billing"
                address={own}
                named={false}
                limits={limits}
                busy={busy}
                onChange={setOwn}
              />
            )}
          </div>

          <div className={block}>
            <TagField
              tags={tags}
              tagging={tagging}
              busy={busy}
              onChange={setTags}
            />
            {groups.length > 0 && (
              <GroupField
                groups={groups}
                membership={membership}
                busy={busy}
                onToggle={(id) =>
                  setMembership((current) => toggled(current, id))
                }
              />
            )}
          </div>

          <NotesField
            id="card-notes"
            value={notes}
            limit={limits?.notes}
            busy={busy}
            onChange={setNotes}
          />

          <p className="shrink-0 px-1 text-[11px] text-faint">
            {t("card.editor.requirement")}
          </p>
          {unfinished && (
            <p className="shrink-0 px-1 text-[11px] text-destructive">
              {t("card.editor.unfinished")}
            </p>
          )}
        </form>
      </ScrollArea>
    </div>
  );
}

function ColorRow({
  automatic,
  swatch,
  busy,
  onPick,
}: {
  /** The colour the automatic choice gives now: the bank site's, else the network's. */
  automatic: string;
  swatch: string | null;
  busy: boolean;
  onPick: (color: string | null) => void;
}) {
  const { t } = useTranslator();
  const options = [
    { key: "automatic", color: null, label: t("card.color.automatic") },
    ...cardSwatches.map((color, index) => ({
      key: color,
      color,
      label: t("card.color.swatch", { number: index + 1 }),
    })),
  ];

  return (
    <div
      role="radiogroup"
      aria-labelledby="card-color-label"
      className={editorRow}
    >
      <span id="card-color-label" className={labelColumn}>
        {t("card.field.color")}
      </span>
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
        {options.map((option) => (
          <label
            key={option.key}
            title={option.label}
            className={cn("relative flex items-center", busy && "opacity-50")}
          >
            <input
              type="radio"
              name="card-color"
              className="peer sr-only"
              checked={swatch === option.color}
              onChange={() => onPick(option.color)}
              disabled={busy}
              aria-label={option.label}
            />
            <span
              className={cn(
                "block rounded-full transition-colors duration-500 motion-reduce:transition-none inset-ring inset-ring-white/15 peer-checked:outline-2 peer-checked:outline-offset-2 peer-checked:outline-foreground peer-focus-visible:ring-[3px] peer-focus-visible:ring-ring/50",
                option.color === null
                  ? "h-5 px-2 text-[11px] leading-5"
                  : "size-5",
              )}
              style={{
                backgroundColor: option.color ?? automatic,
                color: readableOn(option.color ?? automatic),
              }}
            >
              {option.color === null && option.label}
            </span>
          </label>
        ))}
      </div>
    </div>
  );
}
