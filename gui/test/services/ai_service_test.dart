
import 'package:flutter_test/flutter_test.dart';
import 'package:google_generative_ai/google_generative_ai.dart';
import 'package:gui/mcp/client.dart';
import 'package:gui/mcp/model.dart';
import 'package:gui/services/ai_service.dart';
import 'package:gui/services/gemini_wrapper.dart';

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
  Future<void> start() async {}

  @override
  Future<void> stop() async {}

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class MockGeminiFactory implements GeminiFactory {
  final MockGenerativeModelWrapper mockModel = MockGenerativeModelWrapper();
  List<Tool>? capturedTools;

  @override
  GenerativeModelWrapper createModel({required String apiKey, required String model, List<Tool>? tools}) {
    capturedTools = tools;
    return mockModel;
  }
}

class MockGenerativeModelWrapper implements GenerativeModelWrapper {
  final MockChatSessionWrapper session = MockChatSessionWrapper();

  @override
  ChatSessionWrapper startChat({List<Content>? history}) {
    return session;
  }
}

class MockChatSessionWrapper implements ChatSessionWrapper {
  List<String> userMessages = [];

  GenerateContentResponse? nextResponse;

  @override
  Future<GenerateContentResponse> sendMessage(Content content) async {
    if (content.parts.isNotEmpty && content.parts.first is TextPart) {
        userMessages.add((content.parts.first as TextPart).text);
    }
    return nextResponse ?? GenerateContentResponse([Candidate(Content.model([TextPart('response')]), null, null, null, null)], null);
  }

  @override
  List<Content> get history => [];
}

void main() {
  group('AiService', () {
    late AiService service;
    late MockMcpClient mockClient;
    late MockGeminiFactory mockFactory;

    setUp(() {
      mockClient = MockMcpClient();
      mockFactory = MockGeminiFactory();
      service = AiService(
        apiKey: 'dummy',
        mcpClient: mockClient,
        geminiFactory: mockFactory,
      );
    });

    test('init calls createModel with tools', () async {
      mockClient.toolsToReturn = [
        McpTool(name: 'tool1', description: 'desc', inputSchema: {'type': 'object'})
      ];

      await service.init();

      expect(mockFactory.capturedTools, isNotNull);
      final tools = mockFactory.capturedTools!;
      expect(tools.length, 1);
      final tool = tools.first;
      expect(tool.functionDeclarations!.first.name, 'tool1');
    });

    test('sendMessage starts chat and sends message', () async {
      mockClient.toolsToReturn = [];
      await service.init();

      await service.sendMessage('hello', []);

      final session = mockFactory.mockModel.session;
      expect(session.userMessages.contains('hello'), isTrue);
    });
  });
}
