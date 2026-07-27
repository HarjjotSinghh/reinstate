const environment = process.env.VERCEL_TARGET_ENV || process.env.VERCEL_ENV || "";
const branch = process.env.VERCEL_GIT_COMMIT_REF || "";

if (environment !== "production") {
  console.log(`Continuing non-production Vercel build (${environment || "unknown"}).`);
  process.exit(1);
}

if (branch === "main") {
  console.log("Continuing production Vercel build from main.");
  process.exit(1);
}

console.log(
  `Skipping production Vercel build from ${branch || "an unknown branch"}; only main may publish production.`,
);
process.exit(0);
