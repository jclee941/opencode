import { describe, expect, it } from "vitest";

import { esRequest, recentIndex } from "./helpers";

type CatIndexRow = {
  index: string;
  health: string;
  status: string;
  "docs.count": string;
};

describe("Layer 2: OpenCode index presence", () => {
  it("at least one logs-opencode-* index exists", async () => {
    const indices = await esRequest<CatIndexRow[]>(
      "/_cat/indices/logs-opencode-*?format=json",
    );
    expect(indices.length).toBeGreaterThan(0);
  });

  it("index naming follows logs-opencode-YYYY.MM.DD", async () => {
    const indices = await esRequest<CatIndexRow[]>(
      "/_cat/indices/logs-opencode-*?format=json",
    );
    const pattern = /^logs-opencode-\d{4}\.\d{2}\.\d{2}$/;

    for (const index of indices) {
      expect(index.index).toMatch(pattern);
    }
  });

  it("recent index (today or yesterday) exists and is open", async () => {
    const index = await recentIndex();
    const indices = await esRequest<CatIndexRow[]>(
      "/_cat/indices/logs-opencode-*?format=json",
    );
    const target = indices.find((item) => item.index === index);

    expect(target).toBeDefined();
    expect(target?.status).toBe("open");
  });

  it("recent index has non-zero doc count", async () => {
    const index = await recentIndex();
    const indices = await esRequest<CatIndexRow[]>(
      "/_cat/indices/logs-opencode-*?format=json",
    );
    const target = indices.find((item) => item.index === index);

    expect(target).toBeDefined();

    const docCount = Number(target?.["docs.count"] ?? "0");
    expect(Number.isFinite(docCount)).toBe(true);
    expect(docCount).toBeGreaterThan(0);
  });
});
