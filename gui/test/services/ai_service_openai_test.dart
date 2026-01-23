import 'package:flutter_test/flutter_test.dart';
import 'package:gui/mcp/client.dart';
import 'package:gui/mcp/model.dart';
import 'package:gui/services/ai_service.dart';
import 'package:gui/services/llm_provider.dart';

// Mock McpClient
class MockMcpClient extends McpClient {
  MockMcpClient() : super(executablePath: 'dummy');
  @override
  Future<List<McpTool>> listTools() async => [];
  @override
  Future<McpToolResult> callTool(String name, Map<String, dynamic> args) async {
    return McpToolResult(content: [{'type': 'text', 'text': '{"result": "Result for $name"}'}]);
  }
}

class MockOpenAiProvider extends OpenAiCompatibleProvider {
  MockOpenAiProvider() : super(apiKey: 'k', endpoint: 'http://e');

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
  group('AiService OpenAI Tests', () {
    late AiService aiService;
    late MockMcpClient mcpClient;
    late MockOpenAiProvider provider;

    setUp(() async {
      mcpClient = MockMcpClient();
      aiService = AiService(mcpClient: mcpClient);
      provider = MockOpenAiProvider();
      await aiService.init(provider);
    });

    test('sendMessage handles OpenAI tool loop correctly', () async {
      provider.responseQueue.add(LlmResponse(
        text: 'Thinking...',
        toolCalls: [LlmToolCall('tool1', {'arg': 1}, id: 'call_1')]
      ));
      provider.responseQueue.add(LlmResponse(text: 'Final Answer'));

      bool toolCallbackCalled = false;

      final response = await aiService.sendMessage(
        'Hello',
        [],
        onToolOutput: (name, output) {
          if (name == 'tool1') toolCallbackCalled = true;
        }
      );

      expect(response, 'Final Answer');
      expect(toolCallbackCalled, isTrue);
    });
  });
}
