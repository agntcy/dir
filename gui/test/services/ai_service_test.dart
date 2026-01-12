// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0


import 'package:flutter_test/flutter_test.dart';
import 'package:google_generative_ai/google_generative_ai.dart';
import 'package:gui/mcp/client.dart';
import 'package:gui/mcp/model.dart';
import 'package:gui/services/ai_service.dart';
import 'package:gui/services/llm_provider.dart';

// Mocks
class MockMcpClient implements McpClient {
  List<McpTool> toolsToReturn = [];
  Map<String, McpToolResult> toolResults = {};

  @override
  String get executablePath => '';

  @override
  Future<void> initialize() async {}

  @override
  Future<List<McpTool>> listTools() async => toolsToReturn;

  @override
  Future<McpToolResult> callTool(String name, Map<String, dynamic> arguments) async {
    return toolResults[name] ?? McpToolResult(content: 'default');
  }

  @override
  Future<void> start({Map<String, String>? environment}) async {}

  @override
  Future<void> stop() async {}

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class MockLlmProvider implements LlmProvider {
  List<McpTool>? initializedTools;
  String? lastMessage;

  @override
  Future<void> init(List<McpTool> mcpTools) async {
    initializedTools = mcpTools;
  }

  @override
  Future<LlmResponse> sendMessage(String message, List<Content> history) async {
    lastMessage = message;
    return LlmResponse(text: 'response');
  }
}

void main() {
  group('AiService', () {
    late AiService service;
    late MockMcpClient mockClient;
    late MockLlmProvider mockProvider;

    setUp(() {
      mockClient = MockMcpClient();
      mockProvider = MockLlmProvider();
      service = AiService(
        mcpClient: mockClient,
      );
    });

    test('init calls provider.init with tools', () async {
      mockClient.toolsToReturn = [
        McpTool(name: 'tool1', description: 'desc', inputSchema: {'type': 'object'})
      ];

      await service.init(mockProvider);

      expect(mockProvider.initializedTools, isNotNull);
      final tools = mockProvider.initializedTools!;
      expect(tools.length, 1);
      final tool = tools.first;
      expect(tool.name, 'tool1');
    });

    test('sendMessage calls provider.sendMessage', () async {
      mockClient.toolsToReturn = [];
      await service.init(mockProvider);

      await service.sendMessage('hello', []);

      expect(mockProvider.lastMessage, 'hello');
    });
  });
}
