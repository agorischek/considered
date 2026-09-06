import test from "node:test";
import assert from "node:assert/strict";
import { prepareRelease } from "./prepare-release.mjs";

test("an already-published release is an idempotent success", async () => {
  const result = await prepareRelease("v1.2.3", {
    request: async () => ({ tag_name: "v1.2.3", draft: false }),
    run: () => assert.fail("published artifacts must not change"),
  });
  assert.equal(result.state, "already-published");
});
for (const existing of [false, true]) {
  test(`dispatches an ${existing ? "existing" : "absent"} unpublished tag`, async () => {
    const calls = [];
    const result = await prepareRelease("v1.2.3", {
      request: async () => {
        throw Object.assign(new Error("missing"), { status: 404 });
      },
      run: (command, args) => {
        calls.push([command, ...args]);
        return existing ? "v1.2.3\n" : "";
      },
    });
    assert.equal(result.state, existing ? "retry-dispatched" : "dispatched");
    assert.equal(
      calls.some((call) => call.includes("push")),
      !existing,
    );
    assert.deepEqual(calls.at(-1), [
      "gh",
      "workflow",
      "run",
      "release.yml",
      "--ref",
      "main",
      "-f",
      "tag=v1.2.3",
    ]);
  });
}
test("authorization failures never turn into a new release", async () => {
  await assert.rejects(
    prepareRelease("v1.2.3", {
      request: async () => {
        throw Object.assign(new Error("denied"), { status: 403 });
      },
      run: () => assert.fail("must fail closed"),
    }),
    /denied/,
  );
});
