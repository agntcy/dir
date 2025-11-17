CRITICAL: You MUST call tools FIRST before responding!

STEP 1 - CALL THIS TOOL NOW:
Tool: dir-mcp-server__agntcy_oasf_get_schema_domains
Args: {"version": "0.7.0"}

Wait for response. The response will show top-level domains like:
{"name": "artificial_intelligence", ...}, {"name": "data_science", ...}, {"name": "software_engineering", ...}

STEP 2 - Pick ONE domain "name" from Step 1 (e.g. "artificial_intelligence")

STEP 3 - CALL THIS TOOL NOW:
Tool: dir-mcp-server__agntcy_oasf_get_schema_domains  
Args: {"version": "0.7.0", "parent_domain": "YOUR_CHOICE_FROM_STEP_2"}

Wait for response. The response will show sub-domains with "name" and "id" fields like:
{"name": "machine_learning", "caption": "ML", "id": 101}
{"name": "computer_vision", "caption": "CV", "id": 102}

STEP 4 - Pick 1-3 sub-domains and extract BOTH "name" and "id" from Step 3

DO NOT INVENT NAMES! These DO NOT exist:
❌ "ai_model_development"
❌ "cloud_services"
❌ "web_development"
❌ "mobile_apps"

Real examples (from actual schema):
✓ "technology/internet_of_things" with id 101
✓ "technology/software_engineering" with id 102
✓ "trust_and_safety/online_safety" with its corresponding id 401
✓ "finance_and_business/consumer_goods" with its corresponding id 204

STEP 5 - OUTPUT FORMAT (CRITICAL):
Return ONLY the raw JSON object below. DO NOT wrap in markdown code blocks.
DO NOT use markdown formatting. DO NOT add language tags like "json".
DO NOT add ANY text or explanation before or after the JSON.

Your response must start with "{" and end with "}".

Return exactly this structure:
{
  "domains": [
    {
      "name": "parent_domain/sub_domain",
      "id": 101,
      "confidence": 0.95,
      "reasoning": "Brief explanation"
    }
  ]
}

IMPORTANT: The "id" field MUST be the exact ID returned by the get_schema_domains tool in Step 3.
Do NOT invent or guess IDs. Use only the IDs from the tool response.

Agent record to analyze:

