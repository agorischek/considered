import { execFileSync } from "node:child_process";
import { pathToFileURL } from "node:url";
import { github } from "./verify-publication.mjs";

export async function prepareRelease(
  tag,
  { request = github, run = execFileSync } = {},
) {
  if (!/^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(tag ?? "")) {
    throw new Error("Garden version must be vMAJOR.MINOR.PATCH");
  }
  try {
    const release = await request(
      `/repos/quitepicky/considered/releases/tags/${tag}`,
    );
    if (release.tag_name !== tag) throw new Error("Release identity mismatch");
    if (!release.draft) return { tag, state: "already-published" };
  } catch (error) {
    if (error.status !== 404) throw error;
  }
  const options = { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] };
  const existing = run("git", ["tag", "--list", tag], options).trim();
  if (!existing) {
    run(
      "git",
      [
        "-c",
        "user.name=github-actions[bot]",
        "-c",
        "user.email=41898282+github-actions[bot]@users.noreply.github.com",
        "tag",
        "-a",
        tag,
        "-m",
        `Release ${tag}`,
      ],
      options,
    );
    run("git", ["push", "origin", `refs/tags/${tag}`], options);
  }
  run(
    "gh",
    ["workflow", "run", "release.yml", "--ref", "main", "-f", `tag=${tag}`],
    options,
  );
  return { tag, state: existing ? "retry-dispatched" : "dispatched" };
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(process.argv[1]).href
) {
  prepareRelease(process.env.VERSION)
    .then((result) => console.log(JSON.stringify(result)))
    .catch((error) => {
      console.error(error.message);
      process.exitCode = 1;
    });
}
