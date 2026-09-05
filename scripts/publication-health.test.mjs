import test from "node:test";
import assert from "node:assert/strict";
import { checkPublication, reportOutcome } from "./publication-health.mjs";

test("does not resolve a failed draft by checking an older healthy release", async () => {
  let reported;
  await assert.rejects(
    checkPublication({
      readGithub: async (path) =>
        path.includes("?per_page")
          ? [{ tag_name: "v1.2.4", draft: true, prerelease: false }]
          : { tag_name: "v1.2.3" },
      verify: async () =>
        assert.fail("older release must not hide unfinished publication"),
      report: async (error) => {
        reported = error;
      },
    }),
    /v1.2.4.*draft/,
  );
  assert.match(reported.message, /draft/);
});

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
    return {
      items: [
        {
          id: "item",
          title: "Considered downstream publication needs attention",
        },
      ],
    };
  };
  await reportOutcome(null, request);
  assert.equal(calls[1][2].status, "resolved");
});
