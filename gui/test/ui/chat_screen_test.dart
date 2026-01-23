// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:google_generative_ai/google_generative_ai.dart';
import 'package:gui/mcp/client.dart';
import 'package:gui/mcp/model.dart';
import 'package:gui/services/ai_service.dart';
import 'package:gui/services/llm_provider.dart';
import 'package:gui/ui/chat_screen.dart';
import 'package:gui/ui/widgets/search_results_widget.dart';

// Mock AssetBundle for SVGs
class MockAssetBundle extends CachingAssetBundle {
  @override
  Future<String> loadString(String key, {bool cache = true}) async {
    return '<svg viewBox="0 0 1 1"></svg>';
  }

  @override
  Future<ByteData> load(String key) async {
    return ByteData.view(Uint8List.fromList('<svg viewBox="0 0 1 1"></svg>'.codeUnits).buffer);
  }
}

class MockMcpClient extends McpClient {
  MockMcpClient() : super(executablePath: 'mock');

  @override
  Future<McpToolResult> callTool(String name, Map<String, dynamic> arguments) async {
    if (name == 'agntcy_dir_pull_record') {
       return McpToolResult(content: [
         {'type': 'text', 'text': '{"cid": "cid1", "data": {"name": "Mock Agent"}}'}
       ]);
    }
    return McpToolResult(content: []);
  }

  @override
  Future<void> start({Map<String, String>? environment}) async {}

  @override
  Future<void> stop() async {}
}

class MockAiService implements AiService {
  @override
  late McpClient mcpClient;

  // Configurable Mock Behavior
  final Map<String, dynamic>? toolOutputToTrigger;
  final String? responseText;

  MockAiService({this.toolOutputToTrigger, this.responseText}) {
    mcpClient = MockMcpClient();
  }

  @override
  Future<void> init(LlmProvider provider) async {}

  @override
  Future<String?> sendMessage(
    String message,
    List<Content> history, {
    void Function(String, dynamic)? onToolOutput,
  }) async {
     if (toolOutputToTrigger != null && onToolOutput != null) {
       // Simulate a tool output occurring during generation
       onToolOutput(toolOutputToTrigger!['name'], toolOutputToTrigger!['result']);
    }
    return responseText ?? 'echo: $message';
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

void main() {
  group('ChatScreen Tests', () {
    testWidgets('ChatScreen shows welcome message initially', (WidgetTester tester) async {
      final mockService = MockAiService();
      await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: ChatScreen(aiService: mockService),
        ),
      ));
       expect(find.text('Welcome to AGNTCY Agent Directory GUI'), findsOneWidget);
       expect(find.textContaining('Discover AI agents from the network'), findsOneWidget);
    });

    testWidgets('ChatScreen sends message and shows response', (WidgetTester tester) async {
      final mockService = MockAiService();

      await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: ChatScreen(aiService: mockService),
        ),
      ));

      await tester.pump(); // init

      // Enter text
      await tester.enterText(find.byType(TextField), 'Hello');
      await tester.tap(find.byIcon(Icons.send));

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100)); // drain microtasks

      expect(find.text('Hello'), findsOneWidget);
      expect(find.text('echo: Hello'), findsOneWidget);
    });

    testWidgets('ChatScreen renders search results from tool output', (WidgetTester tester) async {
       // Mock Tool Output for Search
       final searchOutput = {
         'count': 10,
         'has_more': false,
         'record_cids': ['cid1', 'cid2'],
       };

       final mockService = MockAiService(
         toolOutputToTrigger: {'name': 'agntcy_dir_search_local', 'result': searchOutput},
         responseText: "I found some agents.",
       );

       await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: ChatScreen(aiService: mockService),
        ),
      ));

      await tester.pump();

      // Trigger search
      await tester.enterText(find.byType(TextField), 'find agents');
      await tester.tap(find.byIcon(Icons.send));

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Should show SearchResultsWidget
      expect(find.byType(SearchResultsWidget), findsOneWidget);
      expect(find.textContaining('10 agents found'), findsOneWidget);

      // Should eventually show the pulled record name "Mock Agent"
      // Wait for auto-pull microtasks
      await tester.pumpAndSettle();
      expect(find.text('Mock Agent'), findsNWidgets(2));
    });

    testWidgets('ChatScreen renders agent record detail from tool output', (WidgetTester tester) async {
       // Mock Tool Output for Pull
       final pullOutput = {
         'cid': 'cid_single',
         'data': {'name': 'Single Agent', 'description': 'Full detail'}
       };

       final mockService = MockAiService(
         toolOutputToTrigger: {'name': 'agntcy_dir_pull_record', 'result': pullOutput},
         responseText: "Here is the agent.",
       );

       await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: ChatScreen(aiService: mockService),
        ),
      ));

      await tester.pump();

      // Trigger action
      await tester.enterText(find.byType(TextField), 'pull this');
      await tester.tap(find.byIcon(Icons.send));

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('Single Agent'), findsOneWidget);
      expect(find.text('Full detail'), findsOneWidget);
    });

    testWidgets('ChatScreen renders generic record as JsonCodeBlock', (WidgetTester tester) async {
       final recordOutput = {'some': 'data', 'nested': 123};

       final mockService = MockAiService(
         toolOutputToTrigger: {'name': 'generic_tool', 'result': recordOutput},
         responseText: "Here is a record.",
       );

       await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: ChatScreen(aiService: mockService),
        ),
      ));

      await tester.pump();

      await tester.enterText(find.byType(TextField), 'run generic');
      await tester.tap(find.byIcon(Icons.send));

      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('Record from generic_tool'), findsOneWidget);
      // JsonCodeBlock renders the JSON data.
      // Might render as text "some: data" or similar depending on implementation.
      // We look for a more specific string to avoid matching "Ask something..." in the TextField hint
      expect(find.textContaining('"some": "data"'), findsOneWidget);
      expect(find.textContaining('123'), findsOneWidget);
    });

    testWidgets('ChatScreen shows warning when unconfigured', (WidgetTester tester) async {
      await tester.pumpWidget(MaterialApp(
        home: DefaultAssetBundle(
          bundle: MockAssetBundle(),
          child: const ChatScreen(aiService: null),
        ),
      ));

      await tester.pumpAndSettle();

      expect(find.text('AI Provider not configured. Please check settings.'), findsOneWidget);
    });
  });
}
