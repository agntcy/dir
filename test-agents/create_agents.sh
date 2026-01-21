#!/bin/bash

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

SKILLS=(
  "natural_language_processing/search/query_understanding"
  "natural_language_processing/information_retrieval/document_search"
  "machine_learning/ranking/search_ranking"
)

DOMAINS=(
  "technology/search_engines"
  "data_science/information_retrieval"
)

AUTHORS=("Search Labs" "AI Search Inc" "AGNTCY Contributors" "OpenSearch Team" "SearchAI Corp")

for i in "${!AGENTS[@]}"; do
  IFS=':' read -r name desc version <<< "${AGENTS[$i]}"
  author="${AUTHORS[$((i % 5))]}"
  skill_id=$((10100 + i))
  domain_id=$((400 + i % 10))
  created_date="2025-0$((1 + i % 9))-$((10 + i % 20))T10:00:00Z"
  
  cat > "/tmp/agent_${i}.json" << EOF
{
  "name": "directory.agntcy.org/test/${name}",
  "version": "${version}",
  "description": "${desc}. This agent provides powerful search capabilities for various use cases.",
  "authors": ["${author}"],
  "schema_version": "0.8.0",
  "created_at": "${created_date}",
  "skills": [
    {"id": ${skill_id}, "name": "${SKILLS[$((i % 3))]}"},
    {"id": $((skill_id + 100)), "name": "natural_language_processing/text_processing/tokenization"}
  ],
  "domains": [
    {"id": ${domain_id}, "name": "${DOMAINS[$((i % 2))]}"}
  ],
  "locators": [
    {"type": "docker_image", "url": "ghcr.io/agntcy/${name}:${version}"}
  ],
  "modules": [
    {
      "id": $((20000 + i)),
      "name": "core/search/engine",
      "data": {"type": "search_module"}
    }
  ],
  "annotations": {
    "category": "search",
    "index": "${i}"
  }
}
EOF

  echo "Created agent_${i}.json: ${name}"
done

echo ""
echo "Now pushing agents to directory server..."
