import { describe, expect, it } from "vitest";

import { esRequest } from "./helpers";

type ClusterInfo = {
  cluster_name: string;
  version: {
    number: string;
  };
};

type ClusterHealth = {
  cluster_name: string;
  status: string;
};

describe("Layer 1: Elasticsearch health", () => {
  it("ES cluster is reachable", async () => {
    const info = await esRequest<ClusterInfo>("/");
    expect(info.cluster_name).toBeTruthy();
    expect(info.version.number).toBeTruthy();
  });

  it("cluster status is green or yellow", async () => {
    const health = await esRequest<ClusterHealth>("/_cluster/health");
    expect(["green", "yellow"]).toContain(health.status);
  });

  it("cluster name is homelab-elk", async () => {
    const info = await esRequest<ClusterInfo>("/");
    expect(info.cluster_name).toBe("homelab-elk");
  });

  it("Elasticsearch major version is 8.x", async () => {
    const info = await esRequest<ClusterInfo>("/");
    expect(info.version.number.startsWith("8.")).toBe(true);
  });
});
