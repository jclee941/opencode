import { describe, expect, it } from "vitest";

import { esRequest, recentIndex } from "./helpers";

type MappingField = {
  type?: string;
  properties?: Record<string, MappingField>;
  fields?: Record<string, MappingField>;
};

type MappingResponse = Record<
  string,
  {
    mappings?: {
      properties?: Record<string, MappingField>;
    };
  }
>;

function findField(
  properties: Record<string, MappingField> | undefined,
  path: string,
): MappingField | undefined {
  if (!properties) {
    return undefined;
  }

  if (properties[path]) {
    return properties[path];
  }

  const segments = path.split(".");
  let current: Record<string, MappingField> | undefined = properties;

  for (let i = 0; i < segments.length; i += 1) {
    if (!current) {
      return undefined;
    }

    const remaining = segments.slice(i).join(".");
    if (current[remaining]) {
      return current[remaining];
    }

    const field = current[segments[i]];
    if (!field) {
      return undefined;
    }

    if (i === segments.length - 1) {
      return field;
    }

    current = field.properties;
  }

  return undefined;
}

describe("Layer 2: OpenCode index mapping", () => {
  it("required fields exist with expected core types", async () => {
    const index = await recentIndex();
    const mapping = await esRequest<MappingResponse>(`/${index}/_mapping`);
    const properties = mapping[index]?.mappings?.properties;

    expect(findField(properties, "@timestamp")?.type).toBe("date");
    expect(findField(properties, "message")).toBeDefined();
    expect(findField(properties, "level")).toBeDefined();
    expect(findField(properties, "service")).toBeDefined();
    expect(findField(properties, "source")).toBeDefined();
  });

  it("enrichment fields exist", async () => {
    const index = await recentIndex();
    const mapping = await esRequest<MappingResponse>(`/${index}/_mapping`);
    const properties = mapping[index]?.mappings?.properties;

    expect(findField(properties, "tier")).toBeDefined();
    expect(findField(properties, "error_classification")).toBeDefined();
    expect(findField(properties, "error_severity")).toBeDefined();
  });

  it("host fields exist", async () => {
    const index = await recentIndex();
    const mapping = await esRequest<MappingResponse>(`/${index}/_mapping`);
    const properties = mapping[index]?.mappings?.properties;

    expect(findField(properties, "host.hostname")).toBeDefined();
    expect(findField(properties, "host.name")).toBeDefined();
  });

  it("metadata fields exist", async () => {
    const index = await recentIndex();
    const mapping = await esRequest<MappingResponse>(`/${index}/_mapping`);
    const properties = mapping[index]?.mappings?.properties;

    expect(findField(properties, "log.file.path")).toBeDefined();
    expect(findField(properties, "agent.name")).toBeDefined();
    expect(findField(properties, "agent.version")).toBeDefined();
  });

  it("keyword sub-fields exist for message, level, service", async () => {
    const index = await recentIndex();
    const mapping = await esRequest<MappingResponse>(`/${index}/_mapping`);
    const properties = mapping[index]?.mappings?.properties;

    expect(findField(properties, "message")?.fields?.keyword?.type).toBe(
      "keyword",
    );
    expect(findField(properties, "level")?.fields?.keyword?.type).toBe(
      "keyword",
    );
    expect(findField(properties, "service")?.fields?.keyword?.type).toBe(
      "keyword",
    );
  });
});
