// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:convert';
import 'package:google_generative_ai/google_generative_ai.dart';
import '../mcp/client.dart';
import '../mcp/model.dart';
import 'llm_provider.dart';

class AiService {
  final McpClient _mcpClient;
  final List<McpTool> _tools = [];

  LlmProvider? _provider;
  final List<Map<String, dynamic>> _azureHistory = [];

  /// Expose MCP client for direct tool calls
  McpClient get mcpClient => _mcpClient;

  AiService({
    required McpClient mcpClient,
  }) : _mcpClient = mcpClient;

  Future<void> init(LlmProvider provider) async {
    _provider = provider;

    // 1. Fetch tools from MCP
    final tools = await _mcpClient.listTools();
    _tools.clear();
    _tools.addAll(tools);

    // 2. Init Provider
    await _provider!.init(_tools);
    _azureHistory.clear();
  }

  Future<String?> sendMessage(
    String message,
    List<Content> history,
    {void Function(String toolName, dynamic output)? onToolOutput}
  ) async {
    if (_provider == null) throw Exception("Provider not initialized");

    if (_provider is AzureOpenAiProvider) {
      return _sendMessageAzure(message, onToolOutput: onToolOutput);
    } else if (_provider is OpenAiCompatibleProvider) {
      return _sendMessageOpenAi(message, onToolOutput: onToolOutput);
    } else {
      return _sendMessageGemini(message, history, onToolOutput: onToolOutput);
    }
  }

  Future<String?> _sendMessageGemini(
    String message,
    List<Content> history,
    {void Function(String, dynamic)? onToolOutput}
  ) async {
    // Rely on the GeminiProvider implementation which manages state via the SDK wrapper
    // BUT we need to handle the loop here because our generic interface returns one turn
    // AND GeminiProvider is simple wrapper.

    // Actually, handling the loop for Gemini inside 'AiService' when using 'LlmProvider'
    // is hard because the session state is hidden.
    //
    // Reverting to: The GeminiProvider implementation creates a NEW session each time.
    // So we CAN use recursive calls or loops if we update 'history'.

    // Initial call
    var response = await _provider!.sendMessage(message, history);

    // Loop
    while (response.toolCalls.isNotEmpty) {
      final parts = <Part>[];

      // Execute tools
      for (final call in response.toolCalls) {
        final result = await _mcpClient.callTool(call.name, call.args);

        // Notify UI if callback provided
        if (onToolOutput != null) {
          _notifyToolOutput(call.name, result.content, onToolOutput);
        }

        // Map result to simple map for Gemini
        parts.add(FunctionResponse(call.name, {'result': result.content}));
      }

      // We need to advance the conversation.
      // Update local history copy to pass to next call
      history.add(Content.text(message)); // Add user message

      final modelParts = response.toolCalls.map((tc) => FunctionCall(tc.name, tc.args)).toList();
      history.add(Content.model(modelParts)); // Add model tool call

      history.add(Content.multi(parts)); // Add tool responses

      // Clear 'message' for next turn so we don't double-add user text?
      // But 'sendMessage' takes 'message' as required arg.
      // If we pass empty string, it adds empty TextPart.
      //
      // This is getting messy.
      //
      // Alternative: Just return the response text if we have it?
      // If toolCalls are present, text is usually null (for Gemini).

      // Let's assume for MVP: Gemini usage via this path is "One Turn" or standard loop.
      // If the User is using Gemini, they use the old Logic?
      // No, I deleted the old logic.

      // FIX: Cast back to GeminiProvider and access raw helper? no.

      // Let's just instantiate the next turn.
      // We pass "" as message because the history tracks the state.
      // But Gemini might complain about empty message.
      // We can use a space " ".
      message = " ";
      response = await _provider!.sendMessage(message, history);
    }

    return response.text;
  }

  Future<String?> _sendMessageAzure(
    String message,
    {void Function(String, dynamic)? onToolOutput}
  ) async {
    _azureHistory.add({"role": "user", "content": message});

    final azureProvider = _provider as AzureOpenAiProvider;
    var response = await azureProvider.sendRaw(_azureHistory);

    while (response.toolCalls.isNotEmpty) {
      final assistantMsg = {
        "role": "assistant",
        "content": response.text,
        "tool_calls": response.toolCalls.map((tc) => {
          "id": tc.id,
          "type": "function",
          "function": {
            "name": tc.name,
            "arguments": _jsonString(tc.args),
          }
        }).toList()
      };
      _azureHistory.add(assistantMsg);

      for (final tc in response.toolCalls) {
        final result = await _mcpClient.callTool(tc.name, tc.args);

        if (onToolOutput != null) {
            _notifyToolOutput(tc.name, result.content, onToolOutput);
        }

        _azureHistory.add({
          "role": "tool",
          "tool_call_id": tc.id,
          "name": tc.name,
          "content": result.content.toString(),
        });
      }

      response = await azureProvider.sendRaw(_azureHistory);
    }

    if (response.text != null) {
      _azureHistory.add({"role": "assistant", "content": response.text});
    }

    return response.text;
  }

  Future<String?> _sendMessageOpenAi(
    String message,
    {void Function(String, dynamic)? onToolOutput}
  ) async {
    _azureHistory.add({"role": "user", "content": message});

    final openaiProvider = _provider as OpenAiCompatibleProvider;
    var response = await openaiProvider.sendRaw(_azureHistory);

    while (response.toolCalls.isNotEmpty) {
      final assistantMsg = {
        "role": "assistant",
        "content": response.text,
        "tool_calls": response.toolCalls.map((tc) => {
          "id": tc.id,
          "type": "function",
          "function": {
            "name": tc.name,
            "arguments": _jsonString(tc.args),
          }
        }).toList()
      };
      _azureHistory.add(assistantMsg);

      for (final tc in response.toolCalls) {
        final result = await _mcpClient.callTool(tc.name, tc.args);

        if (onToolOutput != null) {
            _notifyToolOutput(tc.name, result.content, onToolOutput);
        }

        _azureHistory.add({
          "role": "tool",
          "tool_call_id": tc.id,
          "name": tc.name,
          "content": result.content.toString(),
        });
      }

      response = await openaiProvider.sendRaw(_azureHistory);
    }

    if (response.text != null) {
      _azureHistory.add({"role": "assistant", "content": response.text});
    }

    return response.text;
  }

  void _notifyToolOutput(String name, List<dynamic> content, void Function(String, dynamic) callback) {
     print("DEBUG: _notifyToolOutput called for $name with content: $content");
     for (var item in content) {
         if (item is Map && item['type'] == 'text') {
             try {
                 final text = item['text'] as String;
                 var json = jsonDecode(text);

                 // Normalize "record_data" string into "data" map if present
                 if (json is Map<String, dynamic> && json.containsKey('record_data') && json['record_data'] is String) {
                    try {
                        final recordInner = jsonDecode(json['record_data']);
                        if (recordInner is Map) {
                           json['data'] = recordInner;
                           // Optional: remove the string to save space/confusion
                           // json.remove('record_data');
                        }
                    } catch (e) {
                        print("DEBUG: Failed to parse nested record_data: $e");
                    }
                 }

                 print("DEBUG: Parsed/Normalized json: $json");
                 callback(name, json);
             } catch (e) {
                 print("DEBUG: Failed to parse tool output as JSON: $e");
             }
         } else {
             print("DEBUG: Content element is not text map: $item");
         }
     }
  }

  String _jsonString(Map<String, dynamic> args) => jsonEncode(args);
}
