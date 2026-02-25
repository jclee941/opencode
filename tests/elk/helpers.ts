export const ES_URL = process.env.ES_URL ?? "http://192.168.50.105:9200";

type EsRequestOptions = {
  method?: string;
  body?: unknown;
};

export async function esRequest<T = unknown>(
  path: string,
  options: EsRequestOptions = {},
): Promise<T> {
  const hasBody = options.body !== undefined;
  const method = options.method ?? (hasBody ? "POST" : "GET");

  const response = await fetch(`${ES_URL}${path}`, {
    method,
    headers: hasBody ? { "content-type": "application/json" } : undefined,
    body: hasBody ? JSON.stringify(options.body) : undefined,
  });

  const text = await response.text();

  if (!response.ok) {
    throw new Error(
      `Elasticsearch request failed (${response.status}) ${method} ${path}: ${text}`,
    );
  }

  if (!text.trim()) {
    throw new Error(`Elasticsearch response was empty for ${method} ${path}`);
  }

  try {
    return JSON.parse(text) as T;
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "unknown parse error";
    throw new Error(
      `Failed to parse Elasticsearch JSON for ${method} ${path}: ${message}`,
    );
  }
}

function kstDateParts(offsetDays: number): {
  year: string;
  month: string;
  day: string;
} {
  const date = new Date(Date.now() + offsetDays * 24 * 60 * 60 * 1000);
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Seoul",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(date);

  const year = parts.find((part) => part.type === "year")?.value;
  const month = parts.find((part) => part.type === "month")?.value;
  const day = parts.find((part) => part.type === "day")?.value;

  if (!year || !month || !day) {
    throw new Error("Unable to derive Asia/Seoul calendar date");
  }

  return { year, month, day };
}

export function todayIndex(): string {
  const { year, month, day } = kstDateParts(0);
  return `logs-opencode-${year}.${month}.${day}`;
}

export function yesterdayIndex(): string {
  const { year, month, day } = kstDateParts(-1);
  return `logs-opencode-${year}.${month}.${day}`;
}

export async function recentIndex(): Promise<string> {
  type CatIndexRow = { index?: string };
  const indices = await esRequest<CatIndexRow[]>(
    "/_cat/indices/logs-opencode-*?format=json",
  );
  const indexNames = new Set(
    indices
      .map((index) => index.index)
      .filter(
        (name): name is string => typeof name === "string" && name.length > 0,
      ),
  );

  const today = todayIndex();
  if (indexNames.has(today)) {
    return today;
  }

  return yesterdayIndex();
}
