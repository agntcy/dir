CRITICAL: You MUST call tools FIRST before responding!

STEP 1 - CALL THIS TOOL NOW:
Tool: dir-mcp-server__agntcy_oasf_get_schema_skills
Args: {"version": "0.7.0"}

Wait for response. The response will show top-level skills like:
{"name": "analytical_skills", ...}, {"name": "retrieval_augmented_generation", ...}, {"name": "natural_language_processing", ...}

STEP 2 - Pick ONE skill "name" from Step 1 (e.g. "retrieval_augmented_generation")

STEP 3 - CALL THIS TOOL NOW:
Tool: dir-mcp-server__agntcy_oasf_get_schema_skills  
Args: {"version": "0.7.0", "parent_skill": "YOUR_CHOICE_FROM_STEP_2"}

Wait for response. The response will show sub-skills with "name" and "id" fields like:
{"name": "retrieval_of_information", "caption": "Indexing", "id": 601}
{"name": "document_or_database_question_answering", "caption": "Q&A", "id": 602}

STEP 4 - Pick 1-5 sub-skills and extract BOTH "name" and "id" from Step 3

DO NOT INVENT NAMES! These DO NOT exist:
❌ "information_retrieval_synthesis"
❌ "api_server_operations"  
❌ "statistical_analysis"
❌ "data_visualization"
❌ "code_generation"
❌ "data_retrieval"

Real examples (from actual schema):
✓ "retrieval_augmented_generation/retrieval_of_information" with id 601
✓ "retrieval_augmented_generation/document_or_database_question_answering" with id 602
✓ "natural_language_processing/ethical_interaction" with its corresponding id 108
✓ "analytical_skills/mathematical_reasoning" with its corresponding id 501

STEP 5 - OUTPUT FORMAT (CRITICAL):
Return ONLY the raw JSON object below. DO NOT wrap in markdown code blocks.
DO NOT use markdown formatting. DO NOT add language tags like "json".
DO NOT add ANY text or explanation before or after the JSON.

Your response must start with "{" and end with "}".

Return exactly this structure:
{
  "skills": [
    {
      "name": "parent_skill/sub_skill",
      "id": 601,
      "confidence": 0.95,
      "reasoning": "Brief explanation"
    }
  ]
}

IMPORTANT: The "id" field MUST be the exact ID returned by the get_schema_skills tool in Step 3.
Do NOT invent or guess IDs. Use only the IDs from the tool response.

Agent record to analyze:

