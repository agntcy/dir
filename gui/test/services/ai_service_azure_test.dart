import 'package:flutter_test/flutter_test.dart';
import 'package:gui/mcp/client.dart';
import 'package:gui/mcp/model.dart';
import 'package:gui/services/ai_service.dart';
import 'package:gui/services/llm_provider.dart';

// Mock McpClient (reuse from ai_service_test.dart idea or simple mock)
class MockMcpClient extends McpClient {
  MockMcpClient() : super(executablePath: 'dummy');

  @override
  Future<List<McpTool>> listTools() async => [];

  @override
  Future<McpToolResult> callTool(String name, Map<String, dynamic> args) async {
    return McpToolResult(content: [{'type': 'text', 'text': '{"result": "Result for $name"}'}]);
  }
}

class MockAzureOpenAiProvider extends AzureOpenAiProvider {
  MockAzureOpenAiProvider() : super(
    apiKey: 'k',
    endpoint: 'http://e',
    deploymentId: 'd'
  );

  // Queue of responses to allow simulation of multi-turn
  final List<LlmResponse> responseQueue = [];

  @override
  Future<LlmResponse> sendRaw(List<Map<String, dynamic>> messages) async {
    if (responseQueue.isEmpty) {
      return LlmResponse(text: 'No response');
    }
    return responseQueue.removeAt(0);
  }

  @override
  Future<void> init(List<McpTool> mcpTools) async {}
}

void main() {
  group('AiService Azure Tests', () {
    late AiService aiService;
    late MockMcpClient mcpClient;
    late MockAzureOpenAiProvider provider;

    setUp(() async {
      mcpClient = MockMcpClient();
      aiService = AiService(mcpClient: mcpClient);
      provider = MockAzureOpenAiProvider();
      await aiService.init(provider);
    });

    test('sendMessage handles Azure tool loop correctly', () async {
      // 1. First response requests a tool call
      provider.responseQueue.add(LlmResponse(
        text: 'Thinking...',
        toolCalls: [
          LlmToolCall('tool1', {'arg': 1}, id: 'call_1')
        ]
      ));

      // 2. Second response (after tool) provides final answer
      provider.responseQueue.add(LlmResponse(text: 'Final Answer'));

      bool toolCallbackCalled = false;

      final response = await aiService.sendMessage(
        'Hello',
        [], // History irrelevant for Azure path in AiService (it uses internal _azureHistory)
        onToolOutput: (name, output) {
          if (name == 'tool1') toolCallbackCalled = true;
        }
      );

      expect(response, 'Final Answer');
      expect(toolCallbackCalled, isTrue);
      // Verify provider was called twice
      expect(provider.responseQueue, isEmpty);
    });
  });
}
