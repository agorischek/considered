import test from "node:test";
import assert from "node:assert/strict";
import { reportOutcome } from "./publication-health.mjs";

test("opens durable attention and escalates a new publication failure", async () => {
  const calls = [];
  const request = async (...args) => {
    calls.push(args);
    return { items: [], item: { id: "item" } };
  };
  await reportOutcome(new Error("missing WinGet PR"), request);
  assert.deepEqual(
    calls.map((call) => call[0]),
    [
      "/attention?repository=quitepicky%2Fconsidered&status=active",
      "/attention",
      "/elevate",
      "/attention/item",
    ],
  );
  assert.equal(calls[1][2].priority, "high");
});
test("updates existing attention without repeated escalation", async () => {
  const calls = [];
  const request = async (...args) => {
    calls.push(args);
    return {
      items: [
        {
          id: "item",
          title: "Considered downstream publication needs attention",
          description: "\n\nBetter Stack escalation: accepted.",
        },
      ],
    };
  };
  await reportOutcome(new Error("still awaiting acceptance"), request);
  assert.equal(calls.length, 2);
  assert.equal(calls[1][0], "/attention/item");
});
test("resolves attention after publication recovers", async () => {
  const calls = [];
  const request = async (...args) => {
    calls.push(args);
    return { items: [{ id: "item", title: "Considered downstream publication needs attention" }] };
  };
  await reportOutcome(null, request);
  assert.equal(calls[1][2].status, "resolved");
});
