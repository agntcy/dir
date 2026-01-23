// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'package:flutter_test/flutter_test.dart';
import 'package:google_generative_ai/google_generative_ai.dart';
import 'package:gui/mcp/client.dart';
import 'package:gui/mcp/model.dart';
import 'package:gui/services/ai_service.dart';
import 'package:gui/services/llm_provider.dart';

class MockMcpClient extends McpClient {
  MockMcpClient() : super(executablePath: 'mock');

  @override
  Future<List<McpTool>> listTools() async {
    return [
      McpTool(name: 'tool1', description: 'desc', inputSchema: {})
    ];
  }

  @override
  Future<McpToolResult> callTool(String name, Map<String, dynamic> args) async {
    return McpToolResult(content: '{"status": "ok", "result": "$name"}');
  }
}

class MockLlmProvider implements LlmProvider {
  bool initialized = false;

  // State for sequential responses
  int _callCount = 0;
  final List<LlmResponse> responses;

  MockLlmProvider({required this.responses});

  @override
  Future<void> init(List<McpTool> tools) async {
    initialized = true;
  }

  @override
  Future<LlmResponse> sendMessage(String message, List<Content> history) async {
    if (_callCount < responses.length) {
      return responses[_callCount++];
    }
    return LlmResponse(text: 'EndOfConversation');
  }
}

void main() {
  group('AiService Tests', () {
    test('init initializes provider with tools', () async {
      final mcp = MockMcpClient();
      final service = AiService(mcpClient: mcp);
      final provider = MockLlmProvider(responses: []);

      await service.init(provider);

      expect(provider.initialized, true);
    });

    test('sendMessage returns simple text', () async {
      final mcp = MockMcpClient();
      final service = AiService(mcpClient: mcp);
      final provider = MockLlmProvider(responses: [
        LlmResponse(text: 'Hello User')
      ]);
      await service.init(provider);

      final res = await service.sendMessage('Hi', []);
      expect(res, 'Hello User');
    });

    test('sendMessage handles tool execution loop', () async {
      final mcp = MockMcpClient();
      final service = AiService(mcpClient: mcp);

      // 1st response: Call tool
      // 2nd response: Final answer
      final provider = MockLlmProvider(responses: [
        LlmResponse(toolCalls: [
          LlmToolCall('tool1', {'arg': 'val'})
        ]),
        LlmResponse(text: 'Tool Result Used')
      ]);

      await service.init(provider);

      bool toolCallbackCalled = false;
      final res = await service.sendMessage('Use tool', [], onToolOutput: (name, res) {
        expect(name, 'tool1');
        expect(res['result'], 'tool1');
        toolCallbackCalled = true;
      });

      expect(res, 'Tool Result Used');
      expect(toolCallbackCalled, true);
    });
  });
}
