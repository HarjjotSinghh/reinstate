import { spawnSync } from "node:child_process";
import { describe, expect, it } from "vitest";
import { parseVercelDeploymentURL } from "../../scripts/parse-vercel-deployment-url.mjs";

const gate = new URL("../../scripts/vercel-ignore-production-branch.mjs", import.meta.url);

function runGate(environment: string, branch: string) {
  return spawnSync(process.execPath, [gate.pathname], {
    env: {
      ...process.env,
      VERCEL_ENV: environment,
      VERCEL_GIT_COMMIT_REF: branch,
    },
    encoding: "utf8",
  });
}

describe("Vercel production deployment gate", () => {
  it("continues production builds from main", () => {
    const result = runGate("production", "main");
    expect(result.status).toBe(1);
    expect(result.stdout).toContain("Continuing production");
  });

  it("skips production builds from feature branches", () => {
    const result = runGate("production", "feat/landing-credibility");
    expect(result.status).toBe(0);
    expect(result.stdout).toContain("Skipping production");
  });

  it("fails closed when the production branch is unknown", () => {
    const result = runGate("production", "");
    expect(result.status).toBe(0);
    expect(result.stdout).toContain("unknown branch");
  });

  it("does not block preview builds", () => {
    const result = runGate("preview", "feat/example");
    expect(result.status).toBe(1);
    expect(result.stdout).toContain("non-production");
  });
});

describe("Vercel deployment URL parser", () => {
  it("accepts Vercel CLI 57 structured output", () => {
    expect(
      parseVercelDeploymentURL(
        JSON.stringify({
          status: "ok",
          deployment: {
            url: "https://reinstate-example-harjjot.vercel.app",
          },
        }),
      ),
    ).toBe("https://reinstate-example-harjjot.vercel.app");
  });

  it("retains compatibility with legacy bare URL output", () => {
    expect(
      parseVercelDeploymentURL(
        "Inspect: deployment metadata\nhttps://reinstate-legacy-harjjot.vercel.app\n",
      ),
    ).toBe("https://reinstate-legacy-harjjot.vercel.app");
  });

  it("fails closed when no immutable URL is present", () => {
    expect(parseVercelDeploymentURL('{"status":"error"}')).toBe("");
  });
});
