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

Also available: `embeddings`, `decisions`, `get_decision`, `usage`.

## Development

```bash
cd sdk/python
pip install -e . pytest
python -m pytest tests -q
```
