import test from "node:test";
import assert from "node:assert/strict";
import {
  assertUnpublished,
  releaseForTag,
  verifyPublication,
} from "./verify-publication.mjs";

test("finds an unpublished draft when the tag endpoint returns 404", async () => {
  const draft = { tag_name: "v1.2.3", draft: true, assets: [] };
  const request = async (path) => {
    if (path.includes("/tags/"))
      throw Object.assign(new Error("missing"), { status: 404 });
    return [draft];
  };
  assert.deepEqual(await releaseForTag("v1.2.3", request), draft);
  await assertUnpublished("v1.2.3", request);
});

test("release retries only replace unpublished drafts", async () => {
  await assertUnpublished("v1.2.3", async () => ({ draft: true }));
  await assertUnpublished("v1.2.3", async () => {
    throw Object.assign(new Error("missing"), { status: 404 });
  });
  await assert.rejects(
    assertUnpublished("v1.2.3", async () => ({ draft: false })),
    /already-published/,
  );
  await assert.rejects(
    assertUnpublished("v1.2.3", async () => {
      throw Object.assign(new Error("denied"), { status: 403 });
    }),
    /denied/,
  );
});

function fixture({
  missingAsset = false,
  caskVersion = "1.2.3",
  caskTag = "v1.2.3",
  caskChecksum = "a".repeat(64),
  failedValidation = false,
  pulls = true,
  state = "open",
  merged = null,
  manifest = true,
} = {}) {
  const assets = [
    "checksums.txt",
    ...["darwin", "linux", "windows"].flatMap((os) =>
      ["amd64", "arm64"].map(
        (arch) =>
          `considered_v1.2.3_${os}_${arch}.${os === "windows" ? "zip" : "tar.gz"}`,
      ),
    ),
  ];
  return async (path) => {
    if (path.includes("/check-runs?"))
      return {
        check_runs: failedValidation
          ? [{ name: "03. URLs Validation", conclusion: "failure" }]
          : [],
      };
    if (path.includes("/releases/"))
      return {
        tag_name: "v1.2.3",
        assets: (missingAsset ? assets.slice(1) : assets).map((name) => ({
          name,
          size: 100,
          state: "uploaded",
          digest: `sha256:${"a".repeat(64)}`,
        })),
      };
    if (path.includes("/contents/"))
      return {
        encoding: "base64",
        content: Buffer.from(
          `version "${caskVersion}"\nbinary "considered-scc"\n` +
            assets
              .filter((name) => /_(darwin|linux)_/.test(name))
              .map(
                (name) =>
                  `sha256 "${caskChecksum}"\nurl "https://github.com/quitepicky/considered/releases/download/${caskTag}/${name}"`,
              )
              .join("\n"),
        ).toString("base64"),
      };
    if (path.includes("/files?"))
      return manifest
        ? [
            {
              filename:
                "manifests/q/QuitePicky/Considered/1.2.3/QuitePicky.Considered.installer.yaml",
            },
          ]
        : [];
    return pulls
      ? [
          {
            number: 123,
            state,
            merged_at: merged,
            created_at: "2026-09-05T00:00:00Z",
            html_url: "https://github.com/microsoft/winget-pkgs/pull/123",
            head: {
              sha: "abc123",
              ref: "considered-1.2.3",
              repo: { full_name: "quitepicky/winget-pkgs" },
            },
            base: { repo: { full_name: "microsoft/winget-pkgs" } },
          },
        ]
      : [];
  };
}

test("validates submission separately from acceptance", async () => {
  assert.equal(
    (await verifyPublication("v1.2.3", fixture())).winget,
    "submitted",
  );
  assert.equal(
    (
      await verifyPublication(
        "v1.2.3",
        fixture({ state: "closed", merged: "2026-09-05" }),
      )
    ).winget,
    "accepted",
  );
});
for (const [name, options] of Object.entries({
  asset: { missingAsset: true },
  cask: { caskVersion: "1.2.2" },
  staleCaskUrl: { caskTag: "v1.2.2" },
  staleCaskChecksum: { caskChecksum: "b".repeat(64) },
  missingPr: { pulls: false },
  rejectedPr: { state: "closed" },
  manifest: { manifest: false },
})) {
  test(`rejects incomplete ${name}`, async () =>
    assert.rejects(verifyPublication("v1.2.3", fixture(options))));
}
test("escalates stale upstream submissions", async () => {
  await assert.rejects(
    verifyPublication("v1.2.3", fixture(), {
      monitor: true,
      now: Date.parse("2026-09-13"),
    }),
    /7 days/,
  );
});
test("escalates failed upstream validation without waiting seven days", async () => {
  await assert.rejects(
    verifyPublication("v1.2.3", fixture({ failedValidation: true }), {
      monitor: true,
      now: Date.parse("2026-09-06"),
    }),
    /WinGet validation failed/,
  );
});
test("rejects malformed tags before any requests", async () => {
  await assert.rejects(
    verifyPublication("../main", () => assert.fail("request must not run")),
    /semantic version/,
  );
});
