import { pathToFileURL } from "node:url";

const repository = "quitepicky/considered";

export async function github(path, credential = "GITHUB_TOKEN") {
  const token = process.env[credential];
  if (!token) throw new Error(`Missing ${credential}`);
  const response = await fetch(`https://api.github.com${path}`, {
    headers: {
      authorization: `Bearer ${token}`,
      accept: "application/vnd.github+json",
      "X-GitHub-Api-Version": "2026-03-10",
    },
    signal: AbortSignal.timeout(30_000),
  });
  if (!response.ok) {
    const error = new Error(`GitHub ${response.status} for ${path}`);
    error.status = response.status;
    throw error;
  }
  return response.json();
}

export async function releaseForTag(tag, request = github) {
  try {
    return await request(`/repos/${repository}/releases/tags/${tag}`);
  } catch (error) {
    if (error.status !== 404) throw error;
    // GitHub's tag endpoint returns published releases; drafts require listing.
    for (let page = 1; page <= 10; page++) {
      const releases = await request(
        `/repos/${repository}/releases?per_page=100&page=${page}`,
      );
      const match = releases.find((release) => release.tag_name === tag);
      if (match) return match;
      if (releases.length < 100) throw error;
    }
    throw new Error("Release lookup exceeded its pagination limit");
  }
}

export async function assertUnpublished(tag, request = github) {
  if (
    !/^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?$/.test(
      tag ?? "",
    )
  ) {
    throw new Error("Expected a v-prefixed semantic version");
  }
  let release;
  try {
    release = await releaseForTag(tag, request);
  } catch (error) {
    if (error.status === 404) return;
    throw error;
  }
  if (!release.draft)
    throw new Error(`Refusing to replace already-published release ${tag}`);
}

export async function verifyPublication(
  tag,
  request = github,
  { monitor = false, now = Date.now() } = {},
) {
  if (
    !/^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?$/.test(
      tag ?? "",
    )
  ) {
    throw new Error("Expected a v-prefixed semantic version");
  }
  const version = tag.slice(1);
  const release = await releaseForTag(tag, request);
  if (release.tag_name !== tag || (monitor && release.draft))
    throw new Error("Release identity/draft mismatch");
  const expectedAssets = [
    "checksums.txt",
    ...["darwin", "linux", "windows"].flatMap((os) =>
      ["amd64", "arm64"].map(
        (arch) =>
          `considered_${tag}_${os}_${arch}.${os === "windows" ? "zip" : "tar.gz"}`,
      ),
    ),
  ];
  for (const name of expectedAssets) {
    if (
      !release.assets?.some(
        (asset) =>
          asset.name === name && asset.size > 0 && asset.state === "uploaded",
      )
    ) {
      throw new Error(`Missing or incomplete release asset: ${name}`);
    }
  }
  if (tag.includes("-"))
    return {
      tag,
      release: release.html_url,
      distribution: "prerelease: GitHub only",
    };

  const file = await request(
    "/repos/quitepicky/homebrew-tap/contents/Casks/considered.rb",
    "HOMEBREW_TAP_TOKEN",
  );
  if (file.encoding !== "base64" || typeof file.content !== "string")
    throw new Error("Cannot read published Homebrew cask");
  const cask = Buffer.from(file.content, "base64").toString("utf8");
  if (
    !cask.includes(`version "${version}"`) ||
    !cask.includes("considered-scc") ||
    !cask.includes("github.com/quitepicky/considered")
  )
    throw new Error("Homebrew cask does not match this release");
  // Parse the generated declarative entries without executing Ruby from the tap.
  const downloads = [
    ...cask.matchAll(/sha256\s+"([a-f0-9]{64})"\s+url\s+"([^"]+)"/g),
  ];
  for (const asset of release.assets.filter((asset) =>
    /_(darwin|linux)_/.test(asset.name),
  )) {
    const url = `https://github.com/${repository}/releases/download/${tag}/${asset.name}`;
    if (
      !downloads.some(
        ([, checksum, location]) =>
          location.replaceAll("#{version}", version) === url &&
          asset.digest === `sha256:${checksum}`,
      )
    ) {
      throw new Error(
        `Homebrew download URL/checksum does not match release asset: ${asset.name}`,
      );
    }
  }

  const pulls = await request(
    `/repos/microsoft/winget-pkgs/pulls?head=quitepicky:considered-${version}&base=master&state=all&per_page=100`,
    "WINGET_TOKEN",
  );
  const pull = pulls.find(
    (pr) =>
      pr.head?.repo?.full_name === "quitepicky/winget-pkgs" &&
      pr.head.ref === `considered-${version}` &&
      pr.base?.repo?.full_name === "microsoft/winget-pkgs",
  );
  if (!pull || (pull.state !== "open" && !pull.merged_at))
    throw new Error("WinGet submission is missing or closed without merging");
  const files = await request(
    `/repos/microsoft/winget-pkgs/pulls/${pull.number}/files?per_page=100`,
    "WINGET_TOKEN",
  );
  const manifestPrefix = `manifests/q/QuitePicky/Considered/${version}/`;
  if (
    !files.some(
      (file) =>
        file.filename ===
        `${manifestPrefix}QuitePicky.Considered.installer.yaml`,
    )
  ) {
    throw new Error(
      "WinGet submission does not contain this version's installer manifest",
    );
  }
  if (
    monitor &&
    !pull.merged_at &&
    now - Date.parse(pull.created_at) > 7 * 24 * 60 * 60_000
  ) {
    throw new Error(
      `WinGet submission has awaited acceptance for over 7 days: ${pull.html_url}`,
    );
  }
  if (monitor && !pull.merged_at) {
    const checks = await request(
      `/repos/microsoft/winget-pkgs/commits/${pull.head.sha}/check-runs?per_page=100`,
      "WINGET_TOKEN",
    );
    const failed = checks.check_runs.filter((check) =>
      ["failure", "timed_out", "action_required", "cancelled"].includes(
        check.conclusion,
      ),
    );
    if (failed.length) {
      const names = failed.map((check) => check.name).join(", ");
      throw new Error(`WinGet validation failed (${names}): ${pull.html_url}`);
    }
  }
  return {
    tag,
    release: release.html_url,
    homebrew: "published",
    winget: pull.merged_at ? "accepted" : "submitted",
    wingetUrl: pull.html_url,
  };
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(process.argv[1]).href
) {
  try {
    const tag =
      process.env.RELEASE_TAG ||
      (await github(`/repos/${repository}/releases/latest`)).tag_name;
    if (process.argv.includes("--preflight")) {
      await assertUnpublished(tag);
      process.exit(0);
    }
    const result = await verifyPublication(tag, github, {
      monitor: process.argv.includes("--monitor"),
    });
    console.log(JSON.stringify(result));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
