"""NOETIVELA Python SDK — Apache-2.0.

Submit Inference Contracts instead of hard-coding a provider model ID.
Provider credentials and endpoint URLs never appear in this client.
"""

from .client import (
    ChatResponse,
    ChatStreamError,
    Client,
    Contract,
    DecisionSummary,
    EligibleCandidate,
    EmbeddingResponse,
    ReplayResult,
    StrategyResult,
    UnsatisfiableError,
    ValidateResult,
)
from .contract import (
    Context,
    Cost,
    Latency,
    Policy,
    Quality,
    Reliability,
)

__all__ = [
    "ChatResponse",
    "ChatStreamError",
    "Client",
    "Context",
    "Contract",
    "Cost",
    "DecisionSummary",
    "EligibleCandidate",
    "EmbeddingResponse",
    "Latency",
    "Policy",
    "Quality",
    "Reliability",
    "ReplayResult",
    "StrategyResult",
    "UnsatisfiableError",
    "ValidateResult",
]

__version__ = "0.1.0-alpha"
