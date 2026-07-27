import { spawnSync } from "node:child_process";
import { describe, expect, it } from "vitest";

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
