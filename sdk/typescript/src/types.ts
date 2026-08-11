/** NOETIVELA Inference Contract types (mirrors contracts/schema/inference-contract.json). */

export interface Quality {
  minimum_eval_grade?: string;
  grounding_required?: boolean;
}

export interface Latency {
  ttft_ms?: number;
  p95_ms?: number;
  deadline_ms?: number;
  streaming?: boolean;
}

export interface Cost {
  ceiling_per_request?: string;
  budget_account?: string;
}

export interface Context {
  session_id?: string;
  cache_key?: string;
  prefer_cache_locality?: boolean;
}

export interface Policy {
  data_classification?: string;
  allowed_regions?: string[];
  denied_providers?: string[];
  provider_training?: string;
  retention?: string;
}

export interface Reliability {
  retry_budget?: number;
  fallback?: string;
}

export interface Contract {
  task: string;
  modality: string[];
  domain?: string;
  language?: string[];
  response_schema?: string;
  quality?: Quality;
  latency?: Latency;
  cost?: Cost;
  context?: Context;
  policy?: Policy;
  reliability?: Reliability;
}

export interface Message {
  role: string;
  content: string;
}

export interface Usage {
  input_tokens: number;
  output_tokens: number;
  cached_tokens: number;
  reasoning_tokens: number;
}

export interface TCoS {
  direct_inference: number;
  router_overhead: number;
  context_movement: number;
  cache_loss: number;
  reliability_cost: number;
  total: number;
}

export interface RoutingEvidence {
  decision_id: string;
  model_ref: string;
  endpoint_ref: string;
  tcos: TCoS;
}

export interface ChatResponse {
  content: string;
  finish_reason: string;
  usage: Usage;
  noetivela: RoutingEvidence;
}

export interface ChatChunk {
  delta: string;
  finish_reason?: string;
  usage?: Usage;
}

export interface EligibleCandidate {
  model_ref: string;
  endpoint_ref: string;
  eligible: boolean;
  filter_reasons?: string[];
  score?: number;
}

export interface ValidateResult {
  valid: boolean;
  satisfiable: boolean;
  candidates: EligibleCandidate[];
  policy: string;
}

export interface StrategyResult {
  strategy: string;
  total_tcos: number;
  total_quality: number;
  quality_adjusted_tcos: number;
  served_count: number;
  denied_count: number;
}

export interface ReplayResult {
  records: number;
  savings_percent: number;
  verdict: string;
  baseline: StrategyResult;
  routed: StrategyResult;
}
