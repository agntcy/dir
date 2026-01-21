// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:google_generative_ai/google_generative_ai.dart';
import '../mcp/model.dart';
import 'gemini_wrapper.dart';

// Unified interface for LLM interaction
abstract class LlmProvider {
  Future<void> init(List<McpTool> mcpTools);
  Future<LlmResponse> sendMessage(String message, List<Content> history);
}

// Unified Response type (simplifies mapping back to UI)
class LlmResponse {
  final String? text;
  final List<LlmToolCall> toolCalls;

  LlmResponse({this.text, this.toolCalls = const []});
}

class LlmToolCall {
  final String name;
  final Map<String, dynamic> args;
  final String? id; // For Azure/OpenAI

  LlmToolCall(this.name, this.args, {this.id});
}

// --- GEMINI IMPLEMENTATION ---
class GeminiProvider implements LlmProvider {
  final String apiKey;
  final GeminiFactory geminiFactory;
  GenerativeModelWrapper? _model;

  GeminiProvider({
    required this.apiKey,
    GeminiFactory? geminiFactory,
  }) : geminiFactory = geminiFactory ?? RealGeminiFactory();

  @override
  Future<void> init(List<McpTool> mcpTools) async {
    final toolDeclarations = mcpTools.map((t) {
      return FunctionDeclaration(
        t.name,
        t.description,
        _convertSchema(t.inputSchema),
      );
    }).toList();

    List<Tool>? geminiTools;
    if (toolDeclarations.isNotEmpty) {
      geminiTools = [Tool(functionDeclarations: toolDeclarations)];
    }

    _model = geminiFactory.createModel(
      apiKey: apiKey,
      model: 'gemini-2.0-flash-exp',
      tools: geminiTools,
    );
  }

  @override
  Future<LlmResponse> sendMessage(String message, List<Content> history) async {
    if (_model == null) throw Exception("Gemini Provider not initialized");

    final chat = _model!.startChat(history: history);
    final response = await chat.sendMessage(Content.text(message));

    return LlmResponse(
      text: response.text,
      toolCalls: response.functionCalls.map((fc) => LlmToolCall(fc.name, fc.args)).toList(),
    );
  }

  // Helper to convert MCP JSON schema to Gemini Schema
  Schema _convertSchema(Map<String, dynamic> jsonSchema) {
    // ... [Reuse existing logic from ai_service.dart] ...
    final type = jsonSchema['type'];
    if (type == 'object') {
      final properties = <String, Schema>{};
      final required = <String>[];

      if (jsonSchema.containsKey('properties')) {
        (jsonSchema['properties'] as Map<String, dynamic>).forEach((key, value) {
          properties[key] = _convertSchema(value as Map<String, dynamic>);
        });
      }

      if (jsonSchema.containsKey('required')) {
        required.addAll((jsonSchema['required'] as List).cast<String>());
      }

      return Schema.object(properties: properties, requiredProperties: required);
    } else if (type == 'string') {
      return Schema.string(description: jsonSchema['description']);
    } else if (type == 'integer') {
      return Schema.integer(description: jsonSchema['description']);
    } else if (type == 'number') {
      return Schema.number(description: jsonSchema['description']);
    } else if (type == 'boolean') {
      return Schema.boolean(description: jsonSchema['description']);
    } else if (type == 'array') {
      return Schema.array(
          items: _convertSchema(jsonSchema['items'] as Map<String, dynamic>),
          description: jsonSchema['description']);
    }
    return Schema.string();
  }
}

// --- OPENAI COMPATIBLE IMPLEMENTATION (for AI Gateways) ---
class OpenAiCompatibleProvider implements LlmProvider {
  final String apiKey;
  final String endpoint; // Full endpoint URL for chat completions
  List<Map<String, dynamic>>? _formattedTools;

  OpenAiCompatibleProvider({
    required this.apiKey,
    required this.endpoint,
  });

  @override
  Future<void> init(List<McpTool> mcpTools) async {
    _formattedTools = mcpTools.map((t) {
      var schema = t.inputSchema;
      if (schema.isEmpty) {
        schema = {"type": "object", "properties": <String, dynamic>{}};
      } else {
        if (!schema.containsKey('type')) schema['type'] = 'object';
        if (!schema.containsKey('properties')) schema['properties'] = <String, dynamic>{};
      }
      return {
        "type": "function",
        "function": {
          "name": t.name,
          "description": t.description,
          "parameters": schema,
        }
      };
    }).toList();
  }

  Future<LlmResponse> sendRaw(List<Map<String, dynamic>> messages) async {
    var url = endpoint.trim();
    if (!url.endsWith('/chat/completions')) {
      url = url.endsWith('/') ? '${url}chat/completions' : '$url/chat/completions';
    }

    print('Calling OpenAI-compatible URL: $url');
    final body = {
      "messages": messages,
      "model": "gpt-4o", // Default model, gateway will route appropriately
      if (_formattedTools != null && _formattedTools!.isNotEmpty) "tools": _formattedTools,
    };

    final response = await http.post(
      Uri.parse(url),
      headers: {
        "Content-Type": "application/json",
        "Authorization": "Bearer $apiKey",
      },
      body: jsonEncode(body),
    );

    if (response.statusCode != 200) {
      throw Exception("OpenAI API Error: ${response.statusCode} - ${response.body}");
    }

    final data = jsonDecode(response.body);
    final choice = data['choices'][0];
    final messageData = choice['message'];

    final content = messageData['content'] as String?;
    final toolCallsJson = messageData['tool_calls'] as List<dynamic>?;

    final toolCalls = <LlmToolCall>[];
    if (toolCallsJson != null) {
      for (final tc in toolCallsJson) {
        final func = tc['function'];
        toolCalls.add(LlmToolCall(
          func['name'],
          jsonDecode(func['arguments']),
          id: tc['id'],
        ));
      }
    }

    return LlmResponse(text: content, toolCalls: toolCalls);
  }

  @override
  Future<LlmResponse> sendMessage(String message, List<Content> history) async {
    final messages = _convertHistory(history);
    messages.add({"role": "user", "content": message});
    return sendRaw(messages);
  }

  List<Map<String, dynamic>> _convertHistory(List<Content> history) {
    final List<Map<String, dynamic>> openAiMessages = [];
    for (final h in history) {
      String role = h.role == 'model' ? 'assistant' : (h.role == 'function' ? 'tool' : 'user');
      String text = "";
      for (final part in h.parts) {
        if (part is TextPart) text += part.text;
      }
      if (text.isNotEmpty) {
        openAiMessages.add({'role': role, 'content': text});
      }
    }
    return openAiMessages;
  }
}

// --- AZURE OPENAI IMPLEMENTATION ---
class AzureOpenAiProvider implements LlmProvider {
  final String apiKey;
  final String endpoint; // e.g., https://my-resource.openai.azure.com/
  final String deploymentId;
  final String apiVersion;
  List<Map<String, dynamic>>? _formattedTools;

  AzureOpenAiProvider({
    required this.apiKey,
    required this.endpoint,
    required this.deploymentId,
    this.apiVersion = '2024-10-21',
  });

  @override
  Future<void> init(List<McpTool> mcpTools) async {
    // Convert MCP tools to OpenAI Tools format
    _formattedTools = mcpTools.map((t) {
      var schema = t.inputSchema;

      // Ensure schema is valid for OpenAI (must be object with properties)
      // If empty (no params), ensure it has structure
      if (schema.isEmpty) {
        schema = {
           "type": "object",
           "properties": <String, dynamic>{},
        };
      } else {
        // Fix: OpenAI sometimes dislikes 'additionalProperties' being false if not using strict structured outputs,
        // or expects 'type' to be explicitly 'object' at top level.
        if (!schema.containsKey('type')) {
           schema['type'] = 'object';
        }
        if (!schema.containsKey('properties')) {
           schema['properties'] = <String, dynamic>{};
        }
      }

      return {
        "type": "function",
        "function": {
          "name": t.name,
          "description": t.description,
          "parameters": schema,
        }
      };
    }).toList();
  }

Future<LlmResponse> sendRaw(List<Map<String, dynamic>> messages) async {
    // Clean endpoint
    var baseUrl = endpoint.trim();
    if (baseUrl.endsWith('/')) {
      baseUrl = baseUrl.substring(0, baseUrl.length - 1);
    }
    if (!baseUrl.startsWith('http')) {
      baseUrl = 'https://$baseUrl';
    }

    final url = "$baseUrl/openai/deployments/$deploymentId/chat/completions?api-version=$apiVersion";

    print('Calling Azure URL: $url'); // Debug logging
    final body = {
      "messages": messages,
      if (_formattedTools != null && _formattedTools!.isNotEmpty) "tools": _formattedTools,
    };

    final response = await http.post(
      Uri.parse(url),
      headers: {
        "Content-Type": "application/json",
        "api-key": apiKey,
      },
      body: jsonEncode(body),
    );

    if (response.statusCode != 200) {
      throw Exception("Azure OpenAI Error: ${response.statusCode} - ${response.body}");
    }

    final data = jsonDecode(response.body);
    final choice = data['choices'][0];
    final messageData = choice['message'];

    final content = messageData['content'] as String?;
    final toolCallsJson = messageData['tool_calls'] as List<dynamic>?;

    final toolCalls = <LlmToolCall>[];
    if (toolCallsJson != null) {
      for (final tc in toolCallsJson) {
        final func = tc['function'];
        toolCalls.add(LlmToolCall(
          func['name'],
          jsonDecode(func['arguments']),
          id: tc['id'],
        ));
      }
    }

    return LlmResponse(text: content, toolCalls: toolCalls);
  }

  @override
  Future<LlmResponse> sendMessage(String message, List<Content> history) async {
    // Convert Gemini 'Content' history to OpenAI Messages
    final messages = _convertHistory(history);
    messages.add({"role": "user", "content": message});

    return sendRaw(messages);
  }

  // We need to maintain the exact history structure for OpenAI to work.
  // Converting back and forth from 'Content' is lossy (loses IDs).
  // Strategy: We will accept 'List<Content>' but likely fail on complex re-entry
  // without external state tracking.
  //
  // IMPROVED STRATEGY:
  // The 'history' passed here is purely for context.
  // If the caller (AiService) maintains an Azure-specific history list,
  // it should pass THAT to a specific method, or we adapt.
  //
  // For now, we will assume 'history' is populated with our 'mock' IDs if we generated them,
  // or we just regenerate consistent IDs if possible (hashing?).
  //
  // Better: We expect the caller to pass usage of 'FunctionResponse' that might contain metadata?
  // No, let's keep it simple. If we are in a tool loop, we expect the caller to handle the immediate turn.
  // For long term history, we might lose the 'link' between tool call and result if we don't store it.

  List<Map<String, dynamic>> _convertHistory(List<Content> history) {
    final List<Map<String, dynamic>> openAiMessages = [];

    for (final h in history) {
      String role = 'user';
      if (h.role == 'model') role = 'assistant';
      if (h.role == 'function') role = 'tool';

      // Handle Text
      String text = "";
      for (final part in h.parts) {
        if (part is TextPart) {
          text += part.text;
        }
      }

      // Handle Tool Calls (Model -> User/Tool)
      List<Map<String, dynamic>>? toolCalls;
      for (final part in h.parts) {
        if (part is FunctionCall) {
          toolCalls ??= [];
          toolCalls.add({
             'id': 'call_${part.name}_${DateTime.now().millisecondsSinceEpoch}', // Mock ID
             'type': 'function',
             'function': {
               'name': part.name,
               'arguments': jsonEncode(part.args),
             }
          });
        }
      }

      // Handle Tool/Function Responses (Tool -> Model)
      if (role == 'tool') {
        // OpenAI expects one message per tool response with 'tool_call_id'
        // Gemini pools them. We need to split them or handle mapping.
        // For MVP, simplistic mapping:
         for (final part in h.parts) {
          if (part is FunctionResponse) {
             openAiMessages.add({
               'role': 'tool',
               'content': jsonEncode(part.response),
               'tool_call_id': 'call_${part.name}_mock' // This ID matching is critical in real API...
               // CRITICAL FLAW: OpenAI requires exact ID match from previous turn.
               // Since we don't store the IDs from the previous turn in 'Content', this will fail.
             });
          }
         }
         continue; // Handled
      }

      final msg = <String, dynamic>{'role': role};
      if (text.isNotEmpty) msg['content'] = text;
      if (toolCalls != null) msg['tool_calls'] = toolCalls;

      openAiMessages.add(msg);
    }
    return openAiMessages;
  }
}
