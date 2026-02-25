import { describe, expect, it } from "vitest";

import { esRequest } from "./helpers";

type SearchHit = {
  _source?: Record<string, unknown>;
};

type SearchResponse = {
  hits: {
    total: number | { value: number };
    hits: SearchHit[];
  };
};

function totalValue(total: SearchResponse["hits"]["total"]): number {
  return typeof total === "number" ? total : total.value;
}

function fieldString(
  source: Record<string, unknown> | undefined,
  key: string,
): string | undefined {
  const value = source?.[key];
  if (typeof value === "string") {
    return value;
  }
  if (Array.isArray(value) && typeof value[0] === "string") {
    return value[0];
  }
  return undefined;
}

describe("Layer 3: OpenCode event ingestion", () => {
  it("events exist in the last 24 hours and volume is at least 10", async () => {
    const response = await esRequest<SearchResponse>(
      "/logs-opencode-*/_search",
      {
        method: "POST",
        body: {
          size: 100,
          sort: [{ "@timestamp": { order: "desc" } }],
          query: {
            bool: {
              filter: [
                { range: { "@timestamp": { gte: "now-24h", lte: "now" } } },
              ],
            },
          },
        },
      },
    );

    const total = totalValue(response.hits.total);
    expect(total).toBeGreaterThan(0);
    expect(total).toBeGreaterThanOrEqual(10);
    expect(response.hits.hits.length).toBeGreaterThan(0);
  });

  it("events include required fields: @timestamp, message, level", async () => {
    const response = await esRequest<SearchResponse>(
      "/logs-opencode-*/_search",
      {
        method: "POST",
        body: {
          size: 25,
          sort: [{ "@timestamp": { order: "desc" } }],
          query: {
            bool: {
              filter: [
                { range: { "@timestamp": { gte: "now-24h", lte: "now" } } },
              ],
            },
          },
        },
      },
    );

    expect(response.hits.hits.length).toBeGreaterThan(0);

    for (const hit of response.hits.hits) {
      const source = hit._source;
      expect(fieldString(source, "@timestamp")).toBeTruthy();
      expect(fieldString(source, "message")).toBeTruthy();
      expect(fieldString(source, "level")).toBeTruthy();
    }
  });

  it("service field contains opencode or follows expected opencode naming", async () => {
    const response = await esRequest<SearchResponse>(
      "/logs-opencode-*/_search",
      {
        method: "POST",
        body: {
          size: 50,
          sort: [{ "@timestamp": { order: "desc" } }],
          query: {
            bool: {
              filter: [
                { range: { "@timestamp": { gte: "now-24h", lte: "now" } } },
              ],
            },
          },
        },
      },
    );

    expect(response.hits.hits.length).toBeGreaterThan(0);

    const servicePattern = /^opencode([._-][a-z0-9]+)*$/i;
    const hasExpectedService = response.hits.hits.some((hit) => {
      const service = fieldString(hit._source, "service");
      if (!service) {
        return false;
      }
      return (
        service.toLowerCase().includes("opencode") ||
        servicePattern.test(service)
      );
    });

    expect(hasExpectedService).toBe(true);
  });
});
