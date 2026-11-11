import assert from "node:assert/strict";
import test from "node:test";
import { ClickGate, type GatedEvent, stillMs } from "./click-gate.ts";

function clickGate() {
  const clock = { time: 1_000 };
  const gate = new ClickGate({ now: () => clock.time });
  return { clock, gate };
}

/** A click that records whether the gate stopped it. */
function click() {
  const stopped = { defaultPrevented: false, propagation: true };
  const event: GatedEvent = {
    isTrusted: true,
    preventDefault: () => {
      stopped.defaultPrevented = true;
    },
    stopImmediatePropagation: () => {
      stopped.propagation = false;
    },
  };
  return { event, stopped };
}

test("a click is refused while Chrome has not reported the menu visible", () => {
  const { clock, gate } = clickGate();
  clock.time += stillMs * 4;

  const { event, stopped } = click();
  gate.screen(event);

  assert.equal(gate.accepts(), false);
  assert.deepEqual(stopped, { defaultPrevented: true, propagation: false });
});

test("a click is refused within 500 ms of the menu becoming visible, and accepted after", () => {
  const { clock, gate } = clickGate();
  gate.visibility(true);

  clock.time += stillMs - 1;
  assert.equal(gate.accepts(), false);

  clock.time += 1;
  const { event, stopped } = click();
  gate.screen(event);
  assert.equal(gate.accepts(), true);
  assert.deepEqual(stopped, { defaultPrevented: false, propagation: true });
});

test("a click is refused once the page hides or covers the menu", () => {
  const { clock, gate } = clickGate();
  gate.visibility(true);
  clock.time += stillMs;

  gate.visibility(false);

  assert.equal(gate.accepts(), false);
});

test("a menu shown again waits 500 ms again", () => {
  const { clock, gate } = clickGate();
  gate.visibility(true);
  clock.time += stillMs;
  gate.visibility(false);
  clock.time += stillMs;

  gate.visibility(true);
  assert.equal(gate.accepts(), false);
  clock.time += stillMs;
  assert.equal(gate.accepts(), true);
});

test("a repeated visible report does not restart the wait", () => {
  const { clock, gate } = clickGate();
  gate.visibility(true);
  clock.time += stillMs;

  gate.visibility(true);

  assert.equal(gate.accepts(), true);
});

test("the still period counts from the end of the entrance, which the frame reports", () => {
  const { clock, gate } = clickGate();
  gate.visibility(true);
  clock.time += 200;

  gate.moved();
  clock.time += stillMs - 1;
  assert.equal(gate.accepts(), false);

  clock.time += 1;
  assert.equal(gate.accepts(), true);
});

test("a click is refused within 500 ms of the menu moving", () => {
  const { clock, gate } = clickGate();
  gate.visibility(true);
  clock.time += stillMs * 2;

  gate.moved();
  clock.time += stillMs - 1;
  assert.equal(gate.accepts(), false);

  clock.time += 1;
  assert.equal(gate.accepts(), true);
});

test("engines without isVisible accept full intersection after the still period", () => {
  const clock = { time: 0 };
  const gate = new ClickGate({
    now: () => clock.time,
    nativeVisibility: false,
  });
  gate.report({ isIntersecting: true, intersectionRatio: 1 });
  clock.time += stillMs;
  const { event, stopped } = click();
  gate.screen(event);
  assert.equal(gate.accepts(), true);
  assert.equal(stopped.defaultPrevented, false);
});

test("native occlusion refusal is never replaced with geometric intersection", () => {
  const clock = { time: 0 };
  const gate = new ClickGate({ now: () => clock.time, nativeVisibility: true });
  for (const isVisible of [undefined, false]) {
    gate.report({ isVisible, isIntersecting: true, intersectionRatio: 1 });
    clock.time += stillMs;
    assert.equal(gate.accepts(), false);
  }
});

test("fallback rejects partial, nonintersecting and inactive documents", () => {
  let active = true;
  let time = 1000;
  const gate = new ClickGate({
    now: () => time,
    nativeVisibility: false,
    active: () => active,
  });
  for (const entry of [
    { isIntersecting: true, intersectionRatio: 0.99 },
    { isIntersecting: false, intersectionRatio: 1 },
  ]) {
    gate.report(entry);
    assert.equal(gate.accepts(), false);
  }
  gate.report({ isIntersecting: true, intersectionRatio: 1 });
  time += stillMs;
  assert.equal(gate.accepts(), true);
  active = false;
  assert.equal(gate.accepts(), false);
});

test("a synthetic click is refused even in a stable visible menu", () => {
  const { clock, gate } = clickGate();
  gate.visibility(true);
  clock.time += stillMs;
  const { event, stopped } = click();
  gate.screen({ ...event, isTrusted: false });
  assert.equal(stopped.defaultPrevented, true);
});
