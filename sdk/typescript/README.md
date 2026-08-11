# NOETIVELA TypeScript SDK

Apache-2.0. Zero runtime dependencies (uses global `fetch`; Node ≥ 18).

Submit Inference Contracts instead of hard-coding a provider model ID.

## Install

```bash
npm install @axisrobo/noetivela
```

## Usage

```typescript
import { Client, Contract } from "@axisrobo/noetivela";

const c = new Client("http://localhost:8080", {
  contract: {
    task: "contract_clause_extraction",
    modality: ["text"],
    domain: "legal",
    policy: {
      data_classification: "confidential",
      allowed_regions: ["sg", "cn-north"],
      provider_training: "prohibited",
      retention: "none",
    },
  },
});

// 1. Feasibility check — no tokens consumed.
const res = await c.validate(contract, "legal-confidential-v2");

// 2. Governed chat — model is an alias / "auto".
const chat = await c.chat("auto", [{ role: "user", content: "..." }]);
console.log(chat.noetivela.endpoint_ref, chat.noetivela.tcos);

// 3. Streaming.
for await (const chunk of c.chatStream("auto", [{ role: "user", content: "hi" }])) {
  process.stdout.write(chunk.delta);
}

// 4. Economic evidence.
const rep = await c.replay("ep-openai-sg", "legal-confidential-v2");
console.log(rep.verdict, `${rep.savings_percent.toFixed(2)}%`);
```

Also available: `embeddings`, `decisions`, `getDecision`, `usage`, `eligible`.

## Development

```bash
cd sdk/typescript
npm install
npm test
```
