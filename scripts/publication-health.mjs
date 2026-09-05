import { pathToFileURL } from "node:url";
import { github, verifyPublication } from "./verify-publication.mjs";

const title = "Considered downstream publication needs attention";
const repositoryFullName = "quitepicky/considered";

export async function garden(path, method = "GET", body) {
  const key = process.env.GARDEN_PUBLICATION_KEY;
  if (!key) throw new Error("Missing GARDEN_PUBLICATION_KEY");
  const response = await fetch(`https://garden.gorischek.dev/api/v1${path}`, {
    method,
    headers: { authorization: `Bearer ${key}`, "content-type": "application/json" },
    ...(body ? { body: JSON.stringify(body) } : {}),
    signal: AbortSignal.timeout(30_000),
  });
  if (!response.ok) throw new Error(`Garden ${response.status} for ${path}`);
  return response.json();
}

export async function reportOutcome(
  error,
  request = garden,
  runUrl = "https://github.com/quitepicky/considered/actions",
) {
  const active = await request("/attention?repository=quitepicky%2Fconsidered&status=active");
  const existing = active.items?.find((item) => item.title === title);
  if (!error) {
    if (existing)
      await request(`/attention/${existing.id}`, "PATCH", {
        status: "resolved",
        resolution:
          "Publication verifier confirmed GitHub assets, Homebrew, and WinGet acceptance/submission health.",
      });
    return;
  }
  const description = [
    error.message,
    "Inspect the publishing workflow and repair the existing version.",
    "Do not release a new version solely to hide a failed publication.",
    runUrl,
  ].join("\n\n");
  const marker = "\n\nBetter Stack escalation: accepted.";
  const item =
    existing ??
    (
      await request("/attention", "POST", {
        title,
        description,
        priority: "high",
        repositoryFullName,
        externalUrl: runUrl,
      })
    ).item;
  if (!existing?.description?.includes(marker)) {
    // Persist attention first; an unsuccessful escalation will retry next run.
    await request("/elevate", "POST", { message: `${title}: ${error.message}\n${runUrl}`.slice(0, 2000) });
  }
  if (item.description !== description + marker) {
    await request(`/attention/${item.id}`, "PATCH", {
      description: description + marker,
      externalUrl: runUrl,
    });
  }
}

export async function checkPublication({
  readGithub = github,
  verify = verifyPublication,
  report = reportOutcome,
} = {}) {
  let error;
  try {
    if (process.env.PUBLICATION_FAILURE) throw new Error(process.env.PUBLICATION_FAILURE);
    const tag =
      process.env.RELEASE_TAG || (await readGithub("/repos/quitepicky/considered/releases/latest")).tag_name;
    console.log(JSON.stringify(await verify(tag, readGithub, { monitor: true })));
  } catch (failure) {
    error = failure;
  }
  const runUrl = process.env.GITHUB_RUN_ID
    ? `https://github.com/quitepicky/considered/actions/runs/${process.env.GITHUB_RUN_ID}`
    : undefined;
  await report(error, garden, runUrl);
  if (error) throw error;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  checkPublication().catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
