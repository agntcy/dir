#!/bin/bash

# ── Colors / Logging ───────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
log()  { echo -e "${BLUE}[SETUP]${NC} $*"; }
ok()   { echo -e "${GREEN}[OK]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()  { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# Create 20 fake agents with "search" in the title

AGENTS=(
  "search-assistant-basic:Basic search functionality for documents:v1.0.0"
  "search-engine-pro:Advanced search engine with ML capabilities:v2.1.0"
  "search-indexer:Fast document indexing and search:v1.5.0"
  "deep-search-ai:Deep learning powered search:v3.0.0"
  "search-optimizer:Search result optimization agent:v1.2.0"
  "semantic-search:Semantic understanding for search queries:v2.0.0"
  "search-aggregator:Multi-source search aggregation:v1.8.0"
  "realtime-search:Real-time search updates:v1.0.1"
  "search-analytics:Search analytics and insights:v2.5.0"
  "federated-search:Cross-platform federated search:v1.3.0"
  "search-ranker:ML-based search result ranking:v2.2.0"
  "voice-search-agent:Voice-enabled search assistant:v1.1.0"
  "image-search-bot:Visual similarity search:v1.4.0"
  "code-search-helper:Source code search and analysis:v2.0.1"
  "search-summarizer:Search result summarization:v1.6.0"
  "search-cache-agent:Intelligent search caching:v1.0.2"
  "search-filter-pro:Advanced search filtering:v2.3.0"
  "search-suggest:Smart search suggestions:v1.7.0"
  "hybrid-search:Hybrid keyword and semantic search:v2.4.0"
  "search-monitor:Search performance monitoring:v1.9.0"
)

# Valid skills from OASF schema
SKILLS=(
  "10201:natural_language_processing/natural_language_generation/text_completion"
  "10702:natural_language_processing/analytical_reasoning/problem_solving"
)

DOMAINS=(
  "301:life_science/biotechnology"
)

AUTHORS=("Search Labs" "AI Search Inc" "AGNTCY Contributors" "OpenSearch Team" "SearchAI Corp" "Cisco Systems")

# Module templates with valid data
LLM_MODULE_GPT='{"name": "core/llm/model", "id": 10201, "data": {"models": [{"provider": "openai", "model": "gpt-4", "api_base": "https://api.openai.com/v1"}]}}'
LLM_MODULE_CLAUDE='{"name": "core/llm/model", "id": 10201, "data": {"models": [{"provider": "anthropic", "model": "claude-3-opus", "api_base": "https://api.anthropic.com"}]}}'
LLM_MODULE_GEMINI='{"name": "core/llm/model", "id": 10201, "data": {"models": [{"provider": "google", "model": "gemini-pro", "api_base": "https://generativelanguage.googleapis.com"}]}}'

# Protocol types for annotations (detected by UI for module icons)
PROTOCOLS=("mcp" "a2a" "mcp,a2a" "")

for i in "${!AGENTS[@]}"; do
  IFS=':' read -r name desc version <<< "${AGENTS[$i]}"
  author="${AUTHORS[$((i % 6))]}"

  # Determine number of skills (some agents have many skills for +n testing)
  # Every 4th agent (0, 4, 8, 12, 16) has 5 skills
  # Every 5th agent (5, 10, 15) has 6 skills  
  if [ $((i % 4)) -eq 0 ]; then
    # 5 skills
    skills_json='{"id": 10201, "name": "natural_language_processing/natural_language_generation/text_completion"},{"id": 10702, "name": "natural_language_processing/analytical_reasoning/problem_solving"},{"id": 10201, "name": "natural_language_processing/natural_language_generation/text_completion"},{"id": 10702, "name": "natural_language_processing/analytical_reasoning/problem_solving"},{"id": 10201, "name": "natural_language_processing/natural_language_generation/text_completion"}'
    num_skills=5
  elif [ $((i % 5)) -eq 0 ]; then
    # 6 skills
    skills_json='{"id": 10201, "name": "natural_language_processing/natural_language_generation/text_completion"},{"id": 10702, "name": "natural_language_processing/analytical_reasoning/problem_solving"},{"id": 10201, "name": "natural_language_processing/natural_language_generation/text_completion"},{"id": 10702, "name": "natural_language_processing/analytical_reasoning/problem_solving"},{"id": 10201, "name": "natural_language_processing/natural_language_generation/text_completion"},{"id": 10702, "name": "natural_language_processing/analytical_reasoning/problem_solving"}'
    num_skills=6
  elif [ $((i % 3)) -eq 0 ]; then
    # Two skills
    skills_json='{"id": 10201, "name": "natural_language_processing/natural_language_generation/text_completion"},{"id": 10702, "name": "natural_language_processing/analytical_reasoning/problem_solving"}'
    num_skills=2
  else
    # One skill
    IFS=':' read -r skill_id skill_name <<< "${SKILLS[$((i % 2))]}"
    skills_json="{\"id\": ${skill_id}, \"name\": \"${skill_name}\"}"
    num_skills=1
  fi

  # Parse Domain
  IFS=':' read -r domain_id domain_name <<< "${DOMAINS[0]}"

  created_date="2025-0$((1 + i % 9))-$((10 + i % 20))T10:00:00Z"
  
  # Determine modules based on index for variety in LLM types
  modules_json=""
  case $((i % 4)) in
    0) # GPT-4
      modules_json="${LLM_MODULE_GPT}"
      ;;
    1) # Claude
      modules_json="${LLM_MODULE_CLAUDE}"
      ;;
    2) # Gemini
      modules_json="${LLM_MODULE_GEMINI}"
      ;;
    3) # No modules (basic OASF)
      modules_json=""
      ;;
  esac

  if [ -n "$modules_json" ]; then
    modules_section="\"modules\": [${modules_json}],"
  else
    modules_section=""
  fi
  
  # Get protocol for this agent (for A2A/MCP icons)
  protocol="${PROTOCOLS[$((i % 4))]}"
  if [ -n "$protocol" ]; then
    protocol_annotation="\"protocol\": \"${protocol}\","
  else
    protocol_annotation=""
  fi

  cat > "/tmp/agent_${i}.json" << EOF
{
  "name": "directory.agntcy.org/test/${name}",
  "version": "${version}",
  "description": "${desc}. This agent provides powerful search capabilities.",
  "authors": ["${author}"],
  "schema_version": "0.8.0",
  "created_at": "${created_date}",
  "skills": [
    ${skills_json}
  ],
  "domains": [
    {"id": ${domain_id}, "name": "${domain_name}"}
  ],
  ${modules_section}
  "locators": [
    {"type": "docker_image", "url": "ghcr.io/agntcy/${name}:${version}"}
  ],
  "annotations": {
    ${protocol_annotation}
    "category": "search",
    "index": "${i}"
  }
}
EOF

  ok "Created agent_${i}.json: ${name} (skills: ${num_skills}, protocol: ${protocol:-none})"
done

log "Now pushing agents to directory server..."

for f in /tmp/agent_*.json; do
  log "Pushing $f..."
  if dirctl push "$f" --server-addr 127.0.0.1:8888; then
    ok "Successfully pushed $f"
  else
    err "Failed to push $f"
  fi
done


for f in /tmp/agent_*.json; do
  echo "Pushing $f..."
  dirctl push "$f" --server-addr 127.0.0.1:8888
done