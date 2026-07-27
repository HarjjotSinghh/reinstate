import { pathToFileURL } from "node:url";

export function parseVercelDeploymentURL(output) {
  const body = output.trim();
  try {
    const candidate = JSON.parse(body)?.deployment?.url;
    if (typeof candidate === "string" && candidate.startsWith("https://")) {
      return candidate;
    }
  } catch {
    // Vercel CLI releases before structured output printed one URL per line.
  }

  return (
    body
      .split(/\r?\n/)
      .map((line) => line.trim())
      .find((line) => line.startsWith("https://")) ?? ""
  );
}

if (process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url) {
  let input = "";
  process.stdin.setEncoding("utf8");
  process.stdin.on("data", (chunk) => {
    input += chunk;
  });
  process.stdin.on("end", () => {
    process.stdout.write(parseVercelDeploymentURL(input));
  });
}
