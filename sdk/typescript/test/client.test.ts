import { createServer, Server } from "node:http";
import { test } from "node:test";
import assert from "node:assert/strict";

import { Client, Contract, UnsatisfiableError } from "../src/index.js";

function mockGateway(): Promise<{ server: Server; port: number }> {
  return new Promise((resolve) => {
    const server = createServer((req, res) => {
      const chunks: Buffer[] = [];
      req.on("data", (c) => chunks.push(c));
      req.on("end", () => {
        const url = new URL(req.url ?? "/", "http://localhost");
        const path = url.pathname;
        const send = (code: number, obj: unknown) => {
          res.writeHead(code, { "Content-Type": "application/json" });
          res.end(JSON.stringify(obj));
        };
        if (path === "/v1/chat/completions") {
          assert.ok(req.headers["x-noetivela-contract"], "contract header required");
          const body = JSON.parse(Buffer.concat(chunks).toString());
          if (body.stream) {
            res.writeHead(200, { "Content-Type": "text/event-stream" });
            res.write('data: {"choices":[{"delta":{"content":"Hel"},"finish_reason":""}]}\n\n');
            res.write('data: {"choices":[{"delta":{"content":"lo"},"finish_reason":"stop"}],"usage":{"output_tokens":2}}\n\n');
            res.write('data: {"done":true}\n\n');
            res.end();
            return;
          }
          send(200, {
            choices: [{ message: { content: "ok" }, finish_reason: "stop" }],
            usage: { input_tokens: 100, output_tokens: 20, cached_tokens: 0, reasoning_tokens: 0 },
            noetivela: { decision_id: "abc", model_ref: "m1", endpoint_ref: "e1", tcos: { total: 0.0002 } },
          });
        } else if (path === "/v1/contracts/validate") {
          send(200, { valid: true, satisfiable: true, policy: "legal",
            candidates: [{ model_ref: "m1", endpoint_ref: "e1", eligible: true, score: 0.8 }] });
        } else if (path === "/v1/eligible") {
          send(200, { ranked: [{ model_ref: "m1", endpoint_ref: "e1", eligible: true, score: 0.8 }] });
        } else if (path === "/v1/embeddings") {
          send(200, { data: [[0.1, 0.2]], dimensions: 2, usage: { input_tokens: 2 },
            noetivela: { endpoint_ref: "e1" } });
        } else if (path === "/v1/replay") {
          send(200, { records: 5, savings_percent: 40.0, verdict: "PASS",
            baseline: { strategy: "fixed", quality_adjusted_tcos: 0.02, served_count: 5, denied_count: 0 },
            routed: { strategy: "routed", quality_adjusted_tcos: 0.012, served_count: 5, denied_count: 0 } });
        } else if (path === "/v1/train") {
          const body = JSON.parse(Buffer.concat(chunks).toString());
          assert.equal(body.version_id, "v2");
          assert.equal(body.tcos_weight, 0.5);
          assert.ok(Array.isArray(body.samples) && body.samples.length === 2, "dataset samples required");
          assert.ok(body.samples[0].features && body.samples[0].candidate_ref);
          assert.equal(typeof body.samples[0].quality, "number");
          send(200, { version_id: body.version_id, trained_samples: 2, validate_mae: 0.12,
            min_confidence: 0.6, artifact_digest: "sha256:0123456789abcdef0123456789abcdef" });
        } else {
          send(422, { error: { type: "unsatisfiable_contract", message: "no candidate",
            failed_gates: [{ gate: "policy", reason: "region_not_allowed" }] } });
        }
      });
    });
    server.listen(0, () => {
      const addr = server.address();
      resolve({ server, port: typeof addr === "object" && addr ? addr.port : 0 });
    });
  });
}

const contract: Contract = {
  task: "contract_clause_extraction",
  modality: ["text"],
  policy: { allowed_regions: ["sg"], provider_training: "prohibited" },
};

test("chat", async () => {
  const { server, port } = await mockGateway();
  try {
    const c = new Client(`http://127.0.0.1:${port}`, { contract });
    const resp = await c.chat("auto", [{ role: "user", content: "hi" }]);
    assert.equal(resp.content, "ok");
    assert.equal(resp.noetivela.endpoint_ref, "e1");
  } finally {
    server.close();
  }
});

test("chatStream", async () => {
  const { server, port } = await mockGateway();
  try {
    const c = new Client(`http://127.0.0.1:${port}`, { contract });
    let out = "";
    for await (const chunk of c.chatStream("auto", [{ role: "user", content: "hi" }])) {
      out += chunk.delta;
    }
    assert.equal(out, "Hello");
  } finally {
    server.close();
  }
});

test("validate", async () => {
  const { server, port } = await mockGateway();
  try {
    const c = new Client(`http://127.0.0.1:${port}`);
    const res = await c.validate(contract, "legal");
    assert.equal(res.satisfiable, true);
  } finally {
    server.close();
  }
});

test("replay", async () => {
  const { server, port } = await mockGateway();
  try {
    const c = new Client(`http://127.0.0.1:${port}`);
    const res = await c.replay("exp-ep", "legal");
    assert.equal(res.verdict, "PASS");
    assert.ok(res.savings_percent > 0);
  } finally {
    server.close();
  }
});

test("train with dataset upload", async () => {
  const { server, port } = await mockGateway();
  try {
    const c = new Client(`http://127.0.0.1:${port}`);
    const res = await c.train("v2", {
      samples: [
        { features: { task: "contract_clause_extraction", domain: "legal", input_tokens_est: 1200 },
          candidate_ref: "ep-openai-sg/v1", quality: 0.9, tcos: 0.002 },
        { features: { task: "code_review", input_tokens_est: 3000 },
          candidate_ref: "ep-groq-sg/v1", quality: 0.4, tcos: 0.0001 },
      ],
    });
    assert.equal(res.version_id, "v2");
    assert.equal(res.trained_samples, 2);
    assert.equal(res.min_confidence, 0.6);
    assert.ok(res.artifact_digest?.startsWith("sha256:"));
  } finally {
    server.close();
  }
});

test("unsatisfiable raises typed error", async () => {
  const { server, port } = await mockGateway();
  try {
    const c = new Client(`http://127.0.0.1:${port}`);
    await assert.rejects(
      () => c.request("POST", "/v1/whatever", {}),
      (e: unknown) => e instanceof UnsatisfiableError && e.failedGates.includes("region_not_allowed"),
    );
  } finally {
    server.close();
  }
});
