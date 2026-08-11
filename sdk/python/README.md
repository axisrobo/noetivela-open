# NOETIVELA Python SDK

Apache-2.0. Zero runtime dependencies (stdlib `urllib`).

Submit Inference Contracts instead of hard-coding a provider model ID.
Routing, policy, budget and credential binding are the gateway's job.

## Install

```bash
pip install -e sdk/python
```

## Usage

```python
from noetivela import Client, Contract, Policy

c = Client("http://localhost:8080", tenant="acme", principal="legal-analyst")

contract = Contract(
    task="contract_clause_extraction",
    modality=["text"],
    domain="legal",
    policy=Policy(
        data_classification="confidential",
        allowed_regions=["sg", "cn-north"],
        provider_training="prohibited",
        retention="none",
    ),
    context={"session_id": "case-8821"},
)

# 1. Feasibility check — consumes no tokens.
res = c.validate(contract, policy="legal-confidential-v2")
print("satisfiable:", res.satisfiable)

# 2. Ranked eligible candidates.
for cand in c.eligible(contract, policy="legal-confidential-v2"):
    print(cand.endpoint_ref, "eligible" if cand.eligible else "ineligible")

# 3. Governed inference — model is an alias / "auto", never a provider ID.
chat = c.chat("auto", [{"role": "user", "content": "Extract the indemnity clause: ..."}])
print(chat.endpoint_ref, chat.decision_id, chat.tcos)

# 4. Streaming.
for delta in c.chat_stream("auto", [{"role": "user", "content": "hi"}]):
    print(delta, end="")

# 5. Economic evidence — quality-adjusted TCoS vs a fixed baseline.
rep = c.replay("ep-openai-sg", policy="legal-confidential-v2")
print(rep.verdict, f"{rep.savings_percent:.2f}%")
```

# 6. Learned-router training with an explicit offline dataset.
samples = [
    {
        "features": {  # prompt-free features, see routing/features.go
            "task": "contract_clause_extraction",
            "domain": "legal",
            "input_tokens_est": 1200,
            "has_session": True,
            "tool_use_required": False,
            "structured_output": False,
            "has_deadline": True,
            "has_latency_p95": True,
        },
        "candidate_ref": "ep-openai-sg/v1",
        "quality": 0.9,     # evidence feedback: judged answer quality (0..1)
        "tcos": 0.002,      # measured total cost of service for that route
    },
    {
        "features": {"task": "code_review", "domain": "software", "input_tokens_est": 3000},
        "candidate_ref": "ep-groq-sg/v1",
        "quality": 0.4,
        "tcos": 0.0001,
    },
]
trained = c.train("v2", samples=samples)
print(trained["trained_samples"], trained["min_confidence"], trained["artifact_digest"])
```

Also available: `embeddings`, `decisions`, `get_decision`, `usage`.

## Development

```bash
cd sdk/python
pip install -e . pytest
python -m pytest tests -q
```
