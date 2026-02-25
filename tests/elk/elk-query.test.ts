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
  aggregations?: {
    levels?: {
      buckets: Array<{
        key: string;
        doc_count: number;
      }>;
    };
  };
};

function totalValue(total: SearchResponse["hits"]["total"]): number {
  return typeof total === "number" ? total : total.value;
}

function sourceString(hit: SearchHit, field: string): string | undefined {
  const value = hit._source?.[field];
  if (typeof value === "string") {
    return value;
  }
  if (Array.isArray(value) && typeof value[0] === "string") {
    return value[0];
  }
  return undefined;
}

describe("Layer 4: OpenCode query behavior", () => {
  it("bool/filter query with time range returns results", async () => {
    const response = await esRequest<SearchResponse>(
      "/logs-opencode-*/_search",
      {
        method: "POST",
        body: {
          size: 5,
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

    expect(totalValue(response.hits.total)).toBeGreaterThan(0);
    expect(response.hits.hits.length).toBeGreaterThan(0);
  });

  it("sort by @timestamp desc works", async () => {
    const response = await esRequest<SearchResponse>(
      "/logs-opencode-*/_search",
      {
        method: "POST",
        body: {
          size: 10,
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

    expect(response.hits.hits.length).toBeGreaterThan(1);

    const timestamps = response.hits.hits
      .map((hit) => sourceString(hit, "@timestamp"))
      .filter((value): value is string => typeof value === "string")
      .map((value) => Date.parse(value));

    expect(timestamps.length).toBe(response.hits.hits.length);
    for (let i = 1; i < timestamps.length; i += 1) {
      expect(timestamps[i]).toBeLessThanOrEqual(timestamps[i - 1]);
    }
  });

  it("terms filter on level.keyword works", async () => {
    const aggResponse = await esRequest<SearchResponse>(
      "/logs-opencode-*/_search",
      {
        method: "POST",
        body: {
          size: 0,
          query: {
            bool: {
              filter: [
                { range: { "@timestamp": { gte: "now-24h", lte: "now" } } },
              ],
            },
          },
          aggs: {
            levels: {
              terms: { field: "level.keyword", size: 5 },
            },
          },
        },
      },
    );

    const level = aggResponse.aggregations?.levels?.buckets[0]?.key;
    expect(level).toBeTruthy();

    const filtered = await esRequest<SearchResponse>(
      "/logs-opencode-*/_search",
      {
        method: "POST",
        body: {
          size: 10,
          query: {
            bool: {
              filter: [
                { range: { "@timestamp": { gte: "now-24h", lte: "now" } } },
                { terms: { "level.keyword": [level] } },
              ],
            },
          },
        },
      },
    );

    expect(totalValue(filtered.hits.total)).toBeGreaterThan(0);
    expect(filtered.hits.hits.length).toBeGreaterThan(0);

    for (const hit of filtered.hits.hits) {
      expect(sourceString(hit, "level")).toBe(level);
    }
  });

  it("aggregation on level.keyword returns buckets", async () => {
    const response = await esRequest<SearchResponse>(
      "/logs-opencode-*/_search",
      {
        method: "POST",
        body: {
          size: 0,
          query: {
            bool: {
              filter: [
                { range: { "@timestamp": { gte: "now-24h", lte: "now" } } },
              ],
            },
          },
          aggs: {
            levels: {
              terms: { field: "level.keyword", size: 10 },
            },
          },
        },
      },
    );

    const buckets = response.aggregations?.levels?.buckets ?? [];
    expect(buckets.length).toBeGreaterThan(0);
    expect(buckets[0].key).toBeTruthy();
  });

  it("query with size limit returns expected hit count", async () => {
    const size = 3;
    const response = await esRequest<SearchResponse>(
      "/logs-opencode-*/_search",
      {
        method: "POST",
        body: {
          size,
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

    const expected = Math.min(totalValue(response.hits.total), size);
    expect(response.hits.hits.length).toBe(expected);
  });
});
