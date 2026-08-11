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

// 5. Learned-router training with an explicit offline dataset.
const trained = await c.train("v2", {
  samples: [
    {
      features: { task: "contract_clause_extraction", domain: "legal", input_tokens_est: 1200 },
      candidate_ref: "ep-openai-sg/v1",
      quality: 0.9,   // evidence feedback: judged answer quality (0..1)
      tcos: 0.002,    // measured total cost of service for that route
    },
    {
      features: { task: "code_review", input_tokens_est: 3000 },
      candidate_ref: "ep-groq-sg/v1",
      quality: 0.4,
      tcos: 0.0001,
    },
  ],
});
console.log(trained.trained_samples, trained.min_confidence, trained.artifact_digest);
```

Also available: `embeddings`, `decisions`, `getDecision`, `usage`, `eligible`.

## Development

```bash
cd sdk/typescript
npm install
npm test
```
