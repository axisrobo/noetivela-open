import {
  ChatChunk,
  ChatResponse,
  Contract,
  EligibleCandidate,
  Message,
  ReplayResult,
  ValidateResult,
} from "./types.js";

export class UnsatisfiableError extends Error {
  failedGates: string[];
  constructor(message: string, failedGates: string[]) {
    super(message);
    this.name = "UnsatisfiableError";
    this.failedGates = failedGates;
  }
}

export class NoetivelaError extends Error {}

interface ClientOptions {
  contract?: Contract;
  policy?: string;
  tenant?: string;
  principal?: string;
  timeoutMs?: number;
}

/** Governed gateway client. Submit Contracts, never provider model IDs. */
export class Client {
  private baseUrl: string;
  private contract?: Contract;
  private policy: string;
  private tenant: string;
  private principal: string;
  private timeoutMs: number;

  constructor(baseUrl: string, options: ClientOptions = {}) {
    this.baseUrl = baseUrl.replace(/\/$/, "");
    this.contract = options.contract;
    this.policy = options.policy ?? "";
    this.tenant = options.tenant ?? "";
    this.principal = options.principal ?? "";
    this.timeoutMs = options.timeoutMs ?? 120_000;
  }

  private headers(): Record<string, string> {
    const h: Record<string, string> = { "Content-Type": "application/json" };
    if (this.contract) h["X-Noetivela-Contract"] = JSON.stringify(this.contract);
    if (this.policy) h["X-Noetivela-Policy"] = this.policy;
    if (this.tenant) h["X-Noetivela-Tenant"] = this.tenant;
    if (this.principal) h["X-Noetivela-Principal"] = this.principal;
    return h;
  }

  /** Low-level request helper (public for advanced use and tests). */
  async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const resp = await fetch(this.baseUrl + path, {
      method,
      headers: this.headers(),
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: AbortSignal.timeout(this.timeoutMs),
    });
    const text = await resp.text();
    if (!resp.ok) {
      if (resp.status === 422) {
        let gates: string[] = [];
        try {
          const wrapped = JSON.parse(text);
          gates = (wrapped.error?.failed_gates ?? []).map((g: { reason?: string }) => g.reason ?? String(g));
        } catch {
          /* ignore */
        }
        throw new UnsatisfiableError(text, gates);
      }
      throw new NoetivelaError(`HTTP ${resp.status}: ${text}`);
    }
    return text ? (JSON.parse(text) as T) : ({} as T);
  }

  /** Governed chat completion. model is an alias or "auto". */
  async chat(model: string, messages: Message[]): Promise<ChatResponse> {
    const data = await this.request<{
      choices: { message: { content: string }; finish_reason: string }[];
      usage: ChatResponse["usage"];
      noetivela: ChatResponse["noetivela"];
    }>("POST", "/v1/chat/completions", { model, messages, stream: false });
    if (!data.choices?.length) throw new NoetivelaError("empty response");
    return {
      content: data.choices[0].message.content,
      finish_reason: data.choices[0].finish_reason,
      usage: data.usage,
      noetivela: data.noetivela,
    };
  }

  /** Streams chat deltas as an async iterable (SSE). */
  async *chatStream(model: string, messages: Message[]): AsyncGenerator<ChatChunk> {
    const resp = await fetch(this.baseUrl + "/v1/chat/completions", {
      method: "POST",
      headers: this.headers(),
      body: JSON.stringify({ model, messages, stream: true }),
      signal: AbortSignal.timeout(this.timeoutMs),
    });
    if (!resp.ok || !resp.body) throw new NoetivelaError(`HTTP ${resp.status}`);
    const reader = resp.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      let sep: number;
      while ((sep = buffer.indexOf("\n\n")) >= 0) {
        const event = buffer.slice(0, sep);
        buffer = buffer.slice(sep + 2);
        for (const line of event.split("\n")) {
          if (!line.startsWith("data:")) continue;
          const payload = line.slice(5).trim();
          if (payload === "[DONE]") return;
          const obj = JSON.parse(payload) as { done?: boolean; choices?: { delta: { content?: string }; finish_reason?: string }[]; usage?: ChatChunk["usage"] };
          if (obj.done) return;
          for (const c of obj.choices ?? []) {
            if (c.delta.content) yield { delta: c.delta.content, finish_reason: c.finish_reason, usage: obj.usage };
          }
        }
      }
    }
  }

  async validate(contract: Contract, policy = ""): Promise<ValidateResult> {
    const data = await this.request<ValidateResult>(
      "POST",
      "/v1/contracts/validate",
      { contract, policy: policy || this.policy },
    );
    return data;
  }

  async eligible(contract: Contract, policy = ""): Promise<EligibleCandidate[]> {
    const data = await this.request<{ ranked: EligibleCandidate[] }>(
      "POST",
      "/v1/eligible",
      { contract, policy: policy || this.policy },
    );
    return data.ranked ?? [];
  }

  async embeddings(model: string, input: string[], dimensions = 0): Promise<{ vectors: number[][]; dimensions: number; usage: unknown; endpointRef: string }> {
    const data = await this.request<{
      data: number[][];
      dimensions: number;
      usage: unknown;
      noetivela: { endpoint_ref: string };
    }>("POST", "/v1/embeddings", { model, input, dimensions });
    return {
      vectors: data.data,
      dimensions: data.dimensions,
      usage: data.usage,
      endpointRef: data.noetivela?.endpoint_ref ?? "",
    };
  }

  async decisions(limit = 50, offset = 0, task = "", model = ""): Promise<{ decision_id: string; policy_version?: string; selection?: unknown }[]> {
    const q = new URLSearchParams({ limit: String(limit), offset: String(offset) });
    if (task) q.set("task", task);
    if (model) q.set("model", model);
    const data = await this.request<{ decisions: { decision_id: string; policy_version?: string; selection?: unknown }[] }>(
      "GET",
      `/v1/decisions?${q}`,
    );
    return data.decisions ?? [];
  }

  async getDecision(id: string): Promise<unknown> {
    return this.request("GET", `/v1/decisions/${id}`);
  }

  async usage(): Promise<unknown[]> {
    return this.request("GET", "/v1/usage");
  }

  async replay(baselineEndpoint: string, policy = "", limit = 0): Promise<ReplayResult> {
    return this.request("POST", "/v1/replay", {
      baseline_endpoint: baselineEndpoint,
      policy: policy || this.policy,
      limit,
    });
  }

  async train(
    versionId: string,
    opts: { minSamples?: number; tcosWeight?: number; holdOutFrac?: number; samples?: TrainSample[] } = {},
  ): Promise<{ version_id: string; trained_samples: number; validate_mae?: number; min_confidence: number; artifact_digest?: string }> {
    const body: Record<string, unknown> = {
      version_id: versionId,
      min_samples: opts.minSamples ?? 10,
      tcos_weight: opts.tcosWeight ?? 0.5,
      hold_out_frac: opts.holdOutFrac ?? 0,
    };
    if (opts.samples && opts.samples.length) body.samples = opts.samples;
    return this.request("POST", "/v1/train", body);
  }
}

export interface TrainSample {
  features: Record<string, unknown>;
  candidate_ref: string;
  quality: number;
  tcos: number;
}

export * from "./types.js";
